package jbig2

import (
	"errors"
	"fmt"
)

// Generic region decoding procedure (T.88 §6.2) with arithmetic coding.

// atPixel is an adaptive template pixel: its offset from the current pixel
// and the CONTEXT bit it occupies (Figures 4–7 with the bit numbering of
// Figure 8 and 6.2.5.7).
type atPixel struct {
	dx, dy int
	bit    uint
}

// genericParams are the inputs of 6.2.2 relevant to arithmetic decoding.
type genericParams struct {
	w, h     int
	template uint8     // GBTEMPLATE 0–3
	at       [4][2]int // AT1..AT4 as (dx, dy); only AT1 is used by templates 1–3
	tpgdon   bool
}

// ctxWindows describes how a template's fixed pixels are gathered into
// three sliding windows (rows y-2, y-1 and y) so that CONTEXT is updated
// incrementally per pixel instead of being rebuilt from up to 16 reads.
// Each window holds n bits ending at dx = right of the row, shifted into
// CONTEXT by shift. AT pixels that fall at their nominal positions are
// folded into the windows; the rest are read individually.
type ctxWindows struct {
	n2, r2, s2 uint // row y-2 window: width, rightmost dx, shift
	n1, r1, s1 uint
	n0         uint // row y window: width (always ends at dx = -1), shift 0
	at         []atPixel
}

// nominalAT holds the nominal AT pixel locations (6.2.5.4, Figure 8..11)
// per template; when all ATs are nominal the fast folded windows apply.
var nominalAT = [4][4][2]int{
	{{3, -1}, {-3, -1}, {2, -2}, {-2, -2}},
	{{3, -1}},
	{{2, -1}},
	{{2, -1}},
}

// tpgdContext is the special context used for the SLTP decision of
// typical prediction (6.2.5.7, per template).
var tpgdContext = [4]uint32{0x9B25, 0x0795, 0x00E5, 0x0195}

func windowsFor(p *genericParams) ctxWindows {
	nominal := true
	nAT := 1
	if p.template == 0 {
		nAT = 4
	}
	for i := range nAT {
		if p.at[i] != nominalAT[p.template][i] {
			nominal = false
		}
	}
	var w ctxWindows
	switch p.template {
	case 0:
		if nominal {
			return ctxWindows{n2: 5, r2: 2, s2: 11, n1: 7, r1: 3, s1: 4, n0: 4}
		}
		w = ctxWindows{n2: 3, r2: 1, s2: 12, n1: 5, r1: 2, s1: 5, n0: 4}
		w.at = []atPixel{
			{p.at[0][0], p.at[0][1], 4},
			{p.at[1][0], p.at[1][1], 10},
			{p.at[2][0], p.at[2][1], 11},
			{p.at[3][0], p.at[3][1], 15},
		}
	case 1:
		if nominal {
			return ctxWindows{n2: 4, r2: 2, s2: 9, n1: 6, r1: 3, s1: 3, n0: 3}
		}
		w = ctxWindows{n2: 4, r2: 2, s2: 9, n1: 5, r1: 2, s1: 4, n0: 3}
		w.at = []atPixel{{p.at[0][0], p.at[0][1], 3}}
	case 2:
		if nominal {
			return ctxWindows{n2: 3, r2: 1, s2: 7, n1: 5, r1: 2, s1: 2, n0: 2}
		}
		w = ctxWindows{n2: 3, r2: 1, s2: 7, n1: 4, r1: 1, s1: 3, n0: 2}
		w.at = []atPixel{{p.at[0][0], p.at[0][1], 2}}
	default: // 3: a single reference row
		if nominal {
			return ctxWindows{n1: 6, r1: 2, s1: 4, n0: 4}
		}
		w = ctxWindows{n1: 5, r1: 1, s1: 5, n0: 4}
		w.at = []atPixel{{p.at[0][0], p.at[0][1], 4}}
	}
	return w
}

// decodeGeneric runs the arithmetic generic region decoding procedure
// (6.2.5.7) into a fresh bitmap, using the caller's decoder and context
// set (symbol dictionaries share both across all their symbols).
func decodeGeneric(p *genericParams, mq *mqDecoder, cx []mqCx) (*bitmap, error) {
	if p.template > 3 {
		return nil, fmt.Errorf("invalid GBTEMPLATE %d", p.template)
	}
	if len(cx) < 1<<16 {
		return nil, errors.New("generic context set too small")
	}
	bm, err := newBitmap(p.w, p.h)
	if err != nil {
		return nil, err
	}
	if p.w == 0 || p.h == 0 {
		return bm, nil // nothing to decode; never iterate a huge empty dimension
	}
	win := windowsFor(p)
	m2 := uint32(1)<<win.n2 - 1
	m1 := uint32(1)<<win.n1 - 1
	m0 := uint32(1)<<win.n0 - 1
	r2, r1 := int(win.r2), int(win.r1)
	w := p.w
	ltp := uint32(0)
	for y := range p.h {
		if p.tpgdon {
			ltp ^= mq.decode(&cx[tpgdContext[p.template]])
			if ltp == 1 {
				// Typical row: identical to the one above (or all zero for
				// the first row, which the fresh bitmap already is).
				if y > 0 {
					copy(bm.row(y), bm.row(y-1))
				}
				continue
			}
		}
		row2 := bm.row(y - 2)
		row1 := bm.row(y - 1)
		row := bm.row(y)
		// Prime the windows for x = 0: the window bits cover dx in
		// (right-n, right], so collect pixels 0..right of each row.
		var w2, w1, w0 uint32
		for dx := r2 - int(win.n2) + 1; dx <= r2; dx++ {
			w2 = w2<<1 | rowPix(row2, dx, w)
		}
		for dx := r1 - int(win.n1) + 1; dx <= r1; dx++ {
			w1 = w1<<1 | rowPix(row1, dx, w)
		}
		for x := range w {
			ctx := w2<<win.s2 | w1<<win.s1 | w0
			for i := range win.at {
				a := &win.at[i]
				ctx |= bm.get(x+a.dx, y+a.dy) << a.bit
			}
			bit := mq.decode(&cx[ctx])
			if bit != 0 {
				row[x>>3] |= 0x80 >> uint(x&7)
			}
			w2 = (w2<<1 | rowPix(row2, x+1+r2, w)) & m2
			w1 = (w1<<1 | rowPix(row1, x+1+r1, w)) & m1
			w0 = (w0<<1 | bit) & m0
		}
	}
	return bm, nil
}
