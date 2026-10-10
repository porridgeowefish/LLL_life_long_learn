package jbig2

import (
	"errors"
	"fmt"
)

var errTruncated = errors.New("unexpected end of data")

// reader is a bounds-checked big-endian cursor over a segment's bytes.
// Every read reports truncation instead of panicking.
type reader struct {
	data []byte
	pos  int
}

func (r *reader) remaining() int { return len(r.data) - r.pos }

func (r *reader) u8() (uint8, error) {
	if r.pos >= len(r.data) {
		return 0, errTruncated
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *reader) u16() (uint16, error) {
	if r.remaining() < 2 {
		return 0, errTruncated
	}
	v := uint16(r.data[r.pos])<<8 | uint16(r.data[r.pos+1])
	r.pos += 2
	return v, nil
}

func (r *reader) u32() (uint32, error) {
	if r.remaining() < 4 {
		return 0, errTruncated
	}
	b := r.data[r.pos : r.pos+4]
	r.pos += 4
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
}

// i8 reads a signed byte (adaptive template coordinates).
func (r *reader) i8() (int, error) {
	v, err := r.u8()
	if v >= 0x80 {
		return int(v) - 256, err
	}
	return int(v), err
}

// bytes returns the next n bytes without copying.
func (r *reader) bytes(n int) ([]byte, error) {
	if n < 0 || r.remaining() < n {
		return nil, errTruncated
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

// rest returns everything after the cursor.
func (r *reader) rest() []byte { return r.data[r.pos:] }

// dim converts a 32-bit size field to int, rejecting values that cannot
// describe a decodable bitmap. The spec's 0xFFFFFFFF "unknown" markers are
// handled by callers before reaching here.
func dim(v uint32, what string) (int, error) {
	if v > MaxPixels {
		return 0, fmt.Errorf("%s %d exceeds limit", what, v)
	}
	return int(v), nil
}
