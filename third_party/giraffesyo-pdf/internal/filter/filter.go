// Package filter implements the PDF stream decode filters of ISO 32000-1
// §7.4 that text extraction needs: FlateDecode, LZWDecode, ASCIIHexDecode,
// ASCII85Decode and RunLengthDecode, with the PNG and TIFF predictors.
// The image codecs (DCTDecode, JPXDecode, CCITTFaxDecode, JBIG2Decode) are
// not stream filters here: Apply rejects them, and ImageCodec identifies
// them so a caller can stop a filter chain at the codec and hand the
// still-encoded image data to a decoder.
//
// The package knows nothing about the PDF object model: callers extract
// /DecodeParms fields into Params.
package filter

import (
	"bytes"
	"encoding/ascii85"
	"errors"
	"fmt"
	"io"
)

// Params carries the /DecodeParms fields a filter consumes. The zero
// value means the spec defaults (Predictor 1, Colors 1, BitsPerComponent
// 8, Columns 1, EarlyChange 1).
type Params struct {
	Predictor        int
	Colors           int
	BitsPerComponent int
	Columns          int
	NoEarlyChange    bool // LZW /EarlyChange 0 (the non-default variant)
}

func (p Params) withDefaults() Params {
	if p.Predictor == 0 {
		p.Predictor = 1
	}
	if p.Colors == 0 {
		p.Colors = 1
	}
	if p.BitsPerComponent == 0 {
		p.BitsPerComponent = 8
	}
	if p.Columns == 0 {
		p.Columns = 1
	}
	return p
}

// ImageCodec reports whether name (full or abbreviated) is one of the
// image codecs, returning its full name: DCTDecode, JPXDecode,
// CCITTFaxDecode or JBIG2Decode.
func ImageCodec(name string) (string, bool) {
	switch name {
	case "DCTDecode", "DCT":
		return "DCTDecode", true
	case "JPXDecode":
		return "JPXDecode", true
	case "CCITTFaxDecode", "CCF":
		return "CCITTFaxDecode", true
	case "JBIG2Decode":
		return "JBIG2Decode", true
	default:
		return "", false
	}
}

// Apply wraps r with the named decode filter. The returned reader fails
// on malformed data rather than panicking; callers bound total output.
func Apply(r io.Reader, filterName string, p Params) (io.Reader, error) {
	p = p.withDefaults()
	switch filterName {
	case "FlateDecode", "Fl":
		zr, err := newFlateReader(r)
		if err != nil {
			return nil, fmt.Errorf("FlateDecode: %w", err)
		}
		return newPredictorReader(zr, p)
	case "LZWDecode", "LZW":
		return newPredictorReader(newLZWReader(r, !p.NoEarlyChange), p)
	case "ASCIIHexDecode", "AHx":
		return &asciiHexReader{src: r}, nil
	case "ASCII85Decode", "A85":
		return ascii85.NewDecoder(&ascii85Body{src: r}), nil
	case "RunLengthDecode", "RL":
		return &runLengthReader{src: r}, nil
	default:
		return nil, fmt.Errorf("unsupported stream filter /%s", filterName)
	}
}

// asciiHexReader decodes ASCIIHexDecode: hex pairs, whitespace ignored,
// '>' is end-of-data, an odd final digit is padded with 0.
type asciiHexReader struct {
	src io.Reader
	buf [512]byte
	eod bool
	hi  byte // pending high nibble (0 = none; stored as digit+1)
}

func (h *asciiHexReader) Read(p []byte) (int, error) {
	if h.eod {
		if h.hi != 0 && len(p) > 0 { // odd digit pending at EOD: pad with 0
			p[0] = (h.hi - 1) << 4
			h.hi = 0
			return 1, nil
		}
		return 0, io.EOF
	}
	n := 0
	for n < len(p) {
		m, err := h.src.Read(h.buf[:min(len(h.buf), len(p)-n)])
		for _, c := range h.buf[:m] {
			if h.eod {
				continue // everything after '>' is ignored
			}
			switch {
			case c == '>':
				h.eod = true
			case isHexDigit(c):
				d := hexVal(c)
				if h.hi == 0 {
					h.hi = d + 1
				} else {
					p[n] = (h.hi-1)<<4 | d
					n++
					h.hi = 0
				}
			case isSpace(c):
			default:
				return n, fmt.Errorf("ASCIIHexDecode: invalid byte %#02x", c)
			}
		}
		if h.eod || err != nil {
			if h.hi != 0 && n < len(p) { // odd digit count: pad with 0
				p[n] = (h.hi - 1) << 4
				n++
				h.hi = 0
			}
			h.eod = true
			if err != nil && !errors.Is(err, io.EOF) {
				return n, err
			}
			return n, nil
		}
		if m == 0 {
			break // let the outer guard handle stalls
		}
	}
	return n, nil
}

// ascii85Body strips the optional <~ prefix and stops at the ~> EOD
// marker, feeding clean ASCII85 to the stdlib decoder (which itself
// ignores whitespace).
type ascii85Body struct {
	src     io.Reader
	started bool
	prev    byte // pending '<' or '~' seen at a chunk boundary
	eod     bool
}

func (a *ascii85Body) Read(p []byte) (int, error) {
	if a.eod {
		return 0, io.EOF
	}
	var raw [256]byte
	m, err := a.src.Read(raw[:min(len(raw), len(p))])
	n := 0
	for _, c := range raw[:m] {
		switch a.prev {
		case '<':
			a.prev = 0
			if c == '~' && !a.started { // leading <~
				a.started = true
				continue
			}
			p[n] = '<'
			n++
		case '~':
			a.prev = 0
			if c == '>' {
				a.eod = true
				return n, io.EOF
			}
			p[n] = '~'
			n++
		}
		switch c {
		case '<':
			if a.started {
				p[n] = c
				n++
			} else {
				a.prev = '<'
			}
		case '~':
			a.prev = '~'
		default:
			if !isSpace(c) {
				a.started = true
			}
			p[n] = c
			n++
		}
	}
	if err != nil {
		if a.prev == '<' && n < len(p) {
			p[n] = '<'
			n++
		}
		a.eod = true
	}
	return n, err
}

// runLengthReader decodes RunLengthDecode: a length byte L, then either
// L+1 literal bytes (L ≤ 127) or one byte repeated 257-L times (L ≥ 129);
// 128 is end-of-data.
type runLengthReader struct {
	src io.Reader
	buf []byte
	eod bool
}

func (rl *runLengthReader) Read(p []byte) (int, error) {
	for len(rl.buf) == 0 {
		if rl.eod {
			return 0, io.EOF
		}
		var hdr [1]byte
		if _, err := io.ReadFull(rl.src, hdr[:]); err != nil {
			rl.eod = true
			return 0, io.EOF // truncated run: yield what was decoded
		}
		switch l := hdr[0]; {
		case l == 128:
			rl.eod = true
		case l <= 127:
			lit := make([]byte, int(l)+1)
			n, err := io.ReadFull(rl.src, lit)
			rl.buf = lit[:n]
			if err != nil {
				rl.eod = true
			}
		default:
			var b [1]byte
			if _, err := io.ReadFull(rl.src, b[:]); err != nil {
				rl.eod = true
				break
			}
			rl.buf = bytes.Repeat(b[:], 257-int(l))
		}
	}
	n := copy(p, rl.buf)
	rl.buf = rl.buf[n:]
	return n, nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func hexVal(c byte) byte {
	switch {
	case c <= '9':
		return c - '0'
	case c >= 'a':
		return c - 'a' + 10
	default:
		return c - 'A' + 10
	}
}
