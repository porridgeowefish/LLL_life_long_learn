package filter

import (
	"errors"
	"fmt"
	"io"
)

// maxRowBytes bounds predictor row buffers against hostile /Columns.
var maxRowBytes = 1 << 20

// newPredictorReader undoes the /Predictor pre-filtering applied before
// Flate/LZW compression (ISO 32000-1 §7.4.4.4): 1 = none, 2 = TIFF
// horizontal differencing, 10-15 = the PNG filters. For PNG the Params
// value only selects the family; each row's tag byte picks its filter.
func newPredictorReader(r io.Reader, p Params) (io.Reader, error) {
	switch {
	case p.Predictor == 1:
		return r, nil
	case p.Predictor == 2:
		if p.BitsPerComponent != 8 {
			return nil, fmt.Errorf("TIFF predictor with BitsPerComponent %d not supported", p.BitsPerComponent)
		}
	case p.Predictor >= 10 && p.Predictor <= 15:
	default:
		return nil, fmt.Errorf("unknown /Predictor %d", p.Predictor)
	}
	rowBytes := (p.Colors*p.BitsPerComponent*p.Columns + 7) / 8
	if rowBytes <= 0 || rowBytes > maxRowBytes {
		return nil, fmt.Errorf("predictor row of %d bytes out of range", rowBytes)
	}
	bpp := max(1, p.Colors*p.BitsPerComponent/8)
	return &predictorReader{
		src:  r,
		png:  p.Predictor >= 10,
		bpp:  bpp,
		row:  make([]byte, rowBytes),
		prev: make([]byte, rowBytes),
		pos:  rowBytes, // start empty
	}, nil
}

type predictorReader struct {
	src  io.Reader
	png  bool
	bpp  int
	row  []byte // current decoded row
	prev []byte // previous decoded row (PNG)
	pos  int    // read cursor within row
	err  error
}

// Release forwards to the wrapped decompressor, if pooled.
func (pr *predictorReader) Release() {
	if r, ok := pr.src.(Releaser); ok {
		r.Release()
	}
}

func (pr *predictorReader) Read(p []byte) (int, error) {
	if pr.pos == len(pr.row) {
		if pr.err != nil {
			return 0, pr.err
		}
		if err := pr.nextRow(); err != nil {
			pr.err = err
			return 0, err
		}
	}
	n := copy(p, pr.row[pr.pos:])
	pr.pos += n
	return n, nil
}

func (pr *predictorReader) nextRow() error {
	if !pr.png {
		if _, err := io.ReadFull(pr.src, pr.row); err != nil {
			return eofOr(err)
		}
		// TIFF predictor 2: each byte is a delta from bpp bytes back.
		for i := pr.bpp; i < len(pr.row); i++ {
			pr.row[i] += pr.row[i-pr.bpp]
		}
		pr.pos = 0
		return nil
	}

	var tag [1]byte
	if _, err := io.ReadFull(pr.src, tag[:]); err != nil {
		return eofOr(err)
	}
	if _, err := io.ReadFull(pr.src, pr.row); err != nil {
		return eofOr(err)
	}
	cur, prev, bpp := pr.row, pr.prev, pr.bpp
	switch tag[0] {
	case 0: // None
	case 1: // Sub
		for i := bpp; i < len(cur); i++ {
			cur[i] += cur[i-bpp]
		}
	case 2: // Up
		for i := range cur {
			cur[i] += prev[i]
		}
	case 3: // Average
		for i := range cur {
			var left int
			if i >= bpp {
				left = int(cur[i-bpp])
			}
			cur[i] += byte(((left + int(prev[i])) / 2) & 0xFF) // mean of two bytes
		}
	case 4: // Paeth
		for i := range cur {
			var left, upLeft byte
			if i >= bpp {
				left, upLeft = cur[i-bpp], prev[i-bpp]
			}
			cur[i] += paeth(left, prev[i], upLeft)
		}
	default:
		return fmt.Errorf("invalid PNG row filter %d", tag[0])
	}
	copy(pr.prev, cur)
	pr.pos = 0
	return nil
}

// paeth is the PNG Paeth predictor function (RFC 2083 §6.6).
func paeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	switch {
	case pa <= pb && pa <= pc:
		return a
	case pb <= pc:
		return b
	default:
		return c
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// eofOr maps a partial-row EOF to clean EOF (trailing garbage rows are
// truncated, matching lenient readers) and passes other errors through.
func eofOr(err error) error {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return io.EOF
	}
	return err
}
