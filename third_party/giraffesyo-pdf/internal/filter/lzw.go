package filter

import (
	"fmt"
	"io"
)

// lzwReader decodes LZWDecode data (ISO 32000-1 §7.4.4.2): MSB-first
// variable-width codes from 9 to 12 bits, code 256 clears the table, 257
// is end-of-data. With /EarlyChange 1 (the default) the code width grows
// one code earlier than the table strictly requires — the off-by-one that
// keeps stdlib compress/lzw (which implements only /EarlyChange 0) from
// decoding most PDF streams.
type lzwReader struct {
	src   io.Reader
	early int // 1 or 0

	acc   uint32 // MSB-first bit accumulator
	nbits uint

	suffix [4096]byte   // last byte of each table entry
	prefix [4096]uint16 // previous entry index, or 0xFFFF for roots
	next   int          // next free table slot
	width  uint

	prev int    // previous code, -1 after clear
	out  []byte // decoded bytes not yet delivered
	err  error
}

const (
	lzwClear   = 256
	lzwEOD     = 257
	lzwFirst   = 258
	lzwInvalid = -1
)

func newLZWReader(r io.Reader, earlyChange bool) io.Reader {
	lr := &lzwReader{src: r, early: 0, prev: lzwInvalid, width: 9, next: lzwFirst}
	if earlyChange {
		lr.early = 1
	}
	return lr
}

func (l *lzwReader) Read(p []byte) (int, error) {
	for len(l.out) == 0 {
		if l.err != nil {
			return 0, l.err
		}
		l.err = l.decodeCode()
	}
	n := copy(p, l.out)
	l.out = l.out[n:]
	return n, nil
}

func (l *lzwReader) readCode() (int, error) {
	for l.nbits < l.width {
		var b [1]byte
		if _, err := l.src.Read(b[:]); err != nil {
			return 0, eofOr(err)
		}
		l.acc = l.acc<<8 | uint32(b[0])
		l.nbits += 8
	}
	l.nbits -= l.width
	code := int(l.acc >> l.nbits)
	l.acc &= 1<<l.nbits - 1
	return code, nil
}

func (l *lzwReader) decodeCode() error {
	code, err := l.readCode()
	if err != nil {
		return err
	}
	switch {
	case code == lzwClear:
		l.next, l.width, l.prev = lzwFirst, 9, lzwInvalid
		return nil
	case code == lzwEOD:
		return io.EOF
	case code > l.next || (code == l.next && l.prev == lzwInvalid):
		return fmt.Errorf("LZWDecode: code %d beyond table size %d", code, l.next)
	}

	var entry []byte
	if code == l.next { // KwKwK: prev + first byte of prev
		entry = l.expand(l.prev)
		entry = append(entry, entry[0])
	} else {
		entry = l.expand(code)
	}
	if l.prev != lzwInvalid && l.next < len(l.suffix) {
		l.prefix[l.next] = uint16(l.prev & 0xFFFF)
		l.suffix[l.next] = entry[0]
		l.next++
	}
	l.prev = code
	if l.next+l.early >= 1<<l.width && l.width < 12 {
		l.width++
	}
	l.out = append(l.out, entry...)
	return nil
}

// expand reconstructs a table entry by walking the prefix chain.
func (l *lzwReader) expand(code int) []byte {
	var rev [4096]byte
	n := 0
	for code >= lzwFirst && n < len(rev) {
		rev[n] = l.suffix[code]
		n++
		code = int(l.prefix[code])
	}
	buf := make([]byte, 0, n+1)
	buf = append(buf, byte(code&0xFF)) // chain roots are literals < 256
	for i := n - 1; i >= 0; i-- {
		buf = append(buf, rev[i])
	}
	return buf
}
