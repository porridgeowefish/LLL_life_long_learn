package jbig2

import (
	"errors"
	"fmt"
)

// Text region segments (T.88 §6.4, §7.4.4) with arithmetic coding.

// maxInstances bounds SBNUMINSTANCES; the work budget bounds the
// compositing cost of each instance on top of that.
const maxInstances = 1 << 24

// textRegion holds the parsed text region segment data header (7.4.4.1).
type textRegion struct {
	info       regionInfo
	logStrips  uint
	refCorner  uint8
	transposed bool
	combOp     combOp
	defPixel   uint32
	dsOffset   int
	numInst    uint32
	refine     bool // SBREFINE: an RI value precedes each instance
}

// decodeTextRegion parses and decodes a type 4/6/7 segment body using the
// exports of the referred-to symbol dictionaries as SBSYMS.
func (d *decoder) decodeTextRegion(data []byte, dicts []*symbolDict) (*bitmap, *regionInfo, error) {
	r := &reader{data: data}
	var t textRegion
	var err error
	if t.info, err = readRegionInfo(r); err != nil {
		return nil, nil, err
	}
	flags, err := r.u16()
	if err != nil {
		return nil, nil, err
	}
	sbHuff := flags&1 != 0
	refine := flags>>1&1 != 0
	t.logStrips = uint(flags >> 2 & 3)
	t.refCorner = uint8(flags >> 4 & 3)
	t.transposed = flags>>6&1 != 0
	t.combOp = combOp(flags >> 7 & 3)
	t.defPixel = uint32(flags >> 9 & 1)
	t.dsOffset = int(flags >> 10 & 0x1F)
	if t.dsOffset > 15 {
		t.dsOffset -= 32 // signed 5-bit field
	}
	rTemplate := flags >> 15 & 1
	if sbHuff {
		return nil, nil, fmt.Errorf("huffman-coded text region: %w", errors.ErrUnsupported)
	}
	if refine && rTemplate == 0 {
		// Refinement AT pixels; only needed when an instance actually
		// refines (RI != 0), which is out of scope, so they are skipped.
		if _, err := r.bytes(4); err != nil {
			return nil, nil, err
		}
	}
	t.refine = refine
	if t.numInst, err = r.u32(); err != nil {
		return nil, nil, err
	}
	if t.numInst > maxInstances {
		return nil, nil, fmt.Errorf("text region declares %d instances", t.numInst)
	}
	var syms []*bitmap
	for _, sd := range dicts {
		syms = append(syms, sd.exported...)
	}
	if len(syms) == 0 {
		return nil, nil, errors.New("text region refers to no symbols")
	}
	// Symbols are placed by coordinates, not context, so the region bitmap
	// only needs its page-visible part.
	t.info.h = d.visibleRows(&t.info)
	t.info.w = min(t.info.w, max(d.page.w-t.info.x, 0))
	if err := d.charge(t.info.w, t.info.h); err != nil {
		return nil, nil, err
	}
	bm, err := newBitmap(t.info.w, t.info.h)
	if err != nil {
		return nil, nil, err
	}
	if t.defPixel != 0 {
		bm.fill(1)
	}
	if err := d.decodeTextInstances(&t, bm, syms, newMQDecoder(r.rest())); err != nil {
		return nil, nil, err
	}
	return bm, &t.info, nil
}

// symCodeLen returns SBSYMCODELEN for n symbols (7.4.4.1.1): ceil(log2(n)),
// which is 0 for a single symbol — every ID then costs no bits. T.88's
// Huffman erratum raises this to at least 1, but for arithmetic coding
// real encoders (jbig2enc) and the reference decoder use the plain
// logarithm, and a one-symbol stream only decodes with 0.
func symCodeLen(n int) uint {
	l := uint(0)
	for 1<<l < n {
		l++
	}
	return l
}

// decodeTextInstances runs the text region decoding procedure (6.4.5)
// with arithmetic integer decoding, placing symbols onto bm.
func (d *decoder) decodeTextInstances(t *textRegion, bm *bitmap, syms []*bitmap, mq *mqDecoder) error {
	strips := 1 << t.logStrips
	codeLen := symCodeLen(len(syms))
	iaid := make([]mqCx, 1<<(codeLen+1))
	var iadt, iafs, iads, iait, iari intCtx

	v, ok := mq.decodeInt(&iadt)
	if !ok {
		return errors.New("OOB initial STRIPT")
	}
	stript := -int(v) * strips
	firsts := 0
	inst := 0
	for inst < int(t.numInst) {
		dt, ok := mq.decodeInt(&iadt)
		if !ok {
			return errors.New("OOB strip delta T")
		}
		stript += int(dt) * strips
		dfs, ok := mq.decodeInt(&iafs)
		if !ok {
			return errors.New("OOB first symbol S")
		}
		firsts += int(dfs)
		curs := firsts
		first := true
		for {
			if !first {
				ids, ok := mq.decodeInt(&iads)
				if !ok {
					break // OOB: end of strip
				}
				curs += int(ids) + t.dsOffset
			}
			first = false
			if inst >= int(t.numInst) {
				return errors.New("text region decodes more instances than declared")
			}
			curt := 0
			if strips > 1 {
				ct, ok := mq.decodeInt(&iait)
				if !ok {
					return errors.New("OOB symbol T")
				}
				curt = int(ct)
			}
			tt := stript + curt
			id := mq.decodeIAID(iaid, codeLen)
			if int(id) >= len(syms) {
				return fmt.Errorf("symbol ID %d out of range (%d symbols)", id, len(syms))
			}
			sym := syms[id]
			if t.refine {
				// 6.4.11: RI != 0 means this instance is refined, which is
				// out of scope; RI == 0 uses the symbol as is.
				ri, ok := mq.decodeInt(&iari)
				if !ok || ri != 0 {
					return fmt.Errorf("refined text region symbol: %w", errors.ErrUnsupported)
				}
			}
			// 6.4.5 step 3 c x: in non-transposed mode S runs along x and
			// the right-corner cases advance CURS before drawing, which
			// places the symbol's left edge at the original CURS either
			// way; likewise for S along y when transposed.
			var x, y int
			if !t.transposed {
				x = curs
				y = tt
				if t.refCorner == refBottomLeft || t.refCorner == refBottomRight {
					y = tt - sym.h + 1
				}
				curs += sym.w - 1
			} else {
				y = curs
				x = tt
				if t.refCorner == refTopRight || t.refCorner == refBottomRight {
					x = tt - sym.w + 1
				}
				curs += sym.h - 1
			}
			d.work -= bm.compose(sym, x, y, t.combOp)
			if d.work < 0 {
				return errBudget
			}
			inst++
		}
	}
	return nil
}

// REFCORNER values (7.4.4.1.1).
const (
	refBottomLeft  = 0
	refTopLeft     = 1
	refBottomRight = 2
	refTopRight    = 3
)
