package jbig2

import "fmt"

// bitmap is a bi-level image packed one bit per pixel, rows padded to a
// byte boundary, most significant bit first, 1 = black. It is the native
// JBIG2 layout and also what Decode returns, so the page bitmap's data is
// handed back without copying.
type bitmap struct {
	w, h   int
	stride int
	data   []byte
}

// newBitmap allocates a zeroed w×h bitmap. Callers validate w and h
// against MaxPixels (via the decoder's budget) before calling; this only
// guards against negative sizes so that a slip can never panic.
func newBitmap(w, h int) (*bitmap, error) {
	if w < 0 || h < 0 || w > MaxPixels || h > MaxPixels || (w > 0 && h > MaxPixels/w) {
		return nil, fmt.Errorf("bitmap %dx%d exceeds limits", w, h)
	}
	stride := (w + 7) / 8
	return &bitmap{w: w, h: h, stride: stride, data: make([]byte, stride*h)}, nil
}

// get returns the pixel at (x, y), or 0 outside the bitmap. Out-of-range
// reads are how the spec defines template pixels beyond the edges.
func (b *bitmap) get(x, y int) uint32 {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return 0
	}
	return uint32(b.data[y*b.stride+x>>3]>>(7-uint(x&7))) & 1
}

// row returns row y's bytes, or nil when y is outside the bitmap.
func (b *bitmap) row(y int) []byte {
	if y < 0 || y >= b.h {
		return nil
	}
	return b.data[y*b.stride : (y+1)*b.stride]
}

// rowPix reads pixel x of a row returned by row (nil = all zero).
func rowPix(row []byte, x, w int) uint32 {
	if row == nil || x < 0 || x >= w {
		return 0
	}
	return uint32(row[x>>3]>>(7-uint(x&7))) & 1
}

// fill sets every pixel to v (0 or 1).
func (b *bitmap) fill(v uint32) {
	fv := byte(0)
	if v != 0 {
		fv = 0xFF
	}
	for i := range b.data {
		b.data[i] = fv
	}
}

// combOp is a JBIG2 combination operator (T.88 Table 3 / 7.4.1.5).
type combOp uint8

const (
	opOr combOp = iota
	opAnd
	opXor
	opXnor
	opReplace
)

// compose draws src onto b with its top-left corner at (x0, y0) using op,
// clipping to b. It works a destination byte at a time: the eight source
// bits that line up with a destination byte are gathered from the (at
// most two) source bytes they straddle, then masked to the clipped span.
// It returns the number of destination pixels touched so callers can
// charge their work budget.
func (b *bitmap) compose(src *bitmap, x0, y0 int, op combOp) int {
	if src.w == 0 || src.h == 0 {
		return 0
	}
	sy0, sy1 := 0, src.h
	if y0 < 0 {
		sy0 = -y0
	}
	if y0+sy1 > b.h {
		sy1 = b.h - y0
	}
	sx0, sx1 := 0, src.w
	if x0 < 0 {
		sx0 = -x0
	}
	if x0+sx1 > b.w {
		sx1 = b.w - x0
	}
	if sy1 <= sy0 || sx1 <= sx0 {
		return 0
	}
	dx0, dx1 := x0+sx0, x0+sx1 // destination pixel span [dx0, dx1)
	bi0, bi1 := dx0>>3, (dx1-1)>>3
	firstMask := byte(0xFF) >> uint(dx0&7)
	lastMask := byte(0xFF) << uint(7-((dx1-1)&7))
	// Bit offset of a destination byte's first pixel within its source byte:
	// destination bit bi*8 is source pixel bi*8-x0, i.e. offset (-x0) mod 8.
	shift := uint((-x0) & 7)
	for sy := sy0; sy < sy1; sy++ {
		srow := src.data[sy*src.stride : (sy+1)*src.stride]
		drow := b.data[(y0+sy)*b.stride : (y0+sy+1)*b.stride]
		for bi := bi0; bi <= bi1; bi++ {
			// Source pixel index of destination bit bi*8 is s = bi*8 - x0;
			// floor-divide to find the straddled source bytes.
			s := bi*8 - x0
			si := s >> 3 // arithmetic shift = floor division
			var v uint16
			if si >= 0 && si < src.stride {
				v = uint16(srow[si]) << 8
			}
			if si+1 >= 0 && si+1 < src.stride {
				v |= uint16(srow[si+1])
			}
			sb := byte(v >> (8 - shift) & 0xFF)
			mask := byte(0xFF)
			if bi == bi0 {
				mask &= firstMask
			}
			if bi == bi1 {
				mask &= lastMask
			}
			d := drow[bi]
			switch op {
			case opOr:
				d |= sb & mask
			case opAnd:
				d = (d &^ mask) | (d & sb & mask)
			case opXor:
				d ^= sb & mask
			case opXnor:
				d = (d &^ mask) | (^(d ^ sb) & mask)
			default: // opReplace
				d = (d &^ mask) | (sb & mask)
			}
			drow[bi] = d
		}
	}
	return (sx1 - sx0) * (sy1 - sy0)
}
