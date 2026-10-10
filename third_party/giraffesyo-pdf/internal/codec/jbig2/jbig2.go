// Package jbig2 decodes the JBIG2 bi-level images embedded in PDF files
// through the JBIG2Decode filter (ISO 32000-1 §7.4.7): the "embedded
// stream" organisation of ITU-T T.88 Annex D.3, a sequence of segments
// without a file header, optionally preceded by the shared segments of a
// /JBIG2Globals stream.
//
// Supported (the subset real-world PDF producers emit): segment headers
// (§7.2) including the long referred-to form and unknown data lengths,
// page information and end-of-stripe/page/file segments, the MQ
// arithmetic decoder (Annex E) and integer procedures (Annex A), generic
// regions (§6.2) with all four GB templates, adaptive template pixels and
// TPGDON, MMR (T.6) generic regions via the ccitt package, symbol
// dictionaries (§6.5) and text regions (§6.4) with arithmetic coding in
// all reference corners and both orientations, and the OR/AND/XOR/XNOR/
// replace composition operators.
//
// Unsupported, reported with an error wrapping errors.ErrUnsupported:
// Huffman-coded symbol dictionaries and text regions (SDHUFF/SBHUFF = 1),
// refinement (region types 40–43, SDREFAGG = 1, and refined text region
// instances), pattern dictionaries and halftone regions (types 16, 20,
// 22, 23), the extended 12-pixel generic template, and the colour
// extension. Tables (53), profiles (52) and extension (62) segments are
// skipped.
//
// Decode never panics on malformed input: every size read from the
// stream is validated before allocation, each bitmap is capped at
// MaxPixels pixels, and the total decoding work per call is bounded.
package jbig2

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/giraffesyo/pdf/internal/codec/ccitt"
)

// MaxPixels caps the page (width*height) and every region or symbol
// bitmap a stream may ask Decode to allocate.
const MaxPixels = 1 << 28

// Work budget: the total pixels decoded and composed per Decode call is
// bounded by workFactor times the page area (at least workFloor, so tiny
// pages can still carry symbol dictionaries), so hostile streams cannot
// make many maximal regions cost time unrelated to the image the caller
// asked for. Real pages need roughly one page area of region decoding
// plus small symbol bitmaps.
const (
	workFactor = 16
	workFloor  = 1 << 24
	maxWork    = 4 * MaxPixels
)

// maxSegments bounds the number of segment headers parsed per stream.
const maxSegments = 1 << 16

var errBudget = errors.New("decoding work exceeds limit")

// Segment types (7.3).
const (
	segSymbolDict             = 0
	segTextRegionIntermediate = 4
	segTextRegionImmediate    = 6
	segTextRegionLossless     = 7
	segPatternDict            = 16
	segHalftoneIntermediate   = 20
	segHalftoneImmediate      = 22
	segHalftoneLossless       = 23
	segGenericIntermediate    = 36
	segGenericImmediate       = 38
	segGenericLossless        = 39
	segRefinementIntermediate = 40
	segRefinementImmediate    = 42
	segRefinementLossless     = 43
	segPageInfo               = 48
	segEndOfPage              = 49
	segEndOfStripe            = 50
	segEndOfFile              = 51
	segProfiles               = 52
	segTables                 = 53
	segColourPalette          = 54
	segExtension              = 62
)

// Decode decodes a PDF-embedded JBIG2 stream (the segment sequence of a
// JBIG2Decode filter's data) together with the optional contents of its
// /JBIG2Globals stream, and returns the page bitmap packed one bit per
// pixel, rows padded to a byte boundary (stride (width+7)/8), MSB first,
// with 1 = black (JBIG2's native convention; the PDF filter layer inverts
// this to DeviceGray). width and height come from the PDF image
// dictionary and size the result. Features outside the supported subset
// return an error wrapping errors.ErrUnsupported. Decode never panics on
// malformed input and bounds its memory by width*height and by the
// declared sizes of symbols (symbol or region dimensions that exceed
// MaxPixels pixels are rejected).
func Decode(data, globals []byte, width, height int) ([]byte, error) {
	if width <= 0 || height <= 0 || width > MaxPixels || height > MaxPixels/width {
		return nil, fmt.Errorf("jbig2: invalid image size %dx%d", width, height)
	}
	d := &decoder{
		dicts: make(map[uint32]*symbolDict),
		work:  min(max(workFactor*width*height, workFloor), maxWork),
	}
	var err error
	if d.page, err = newBitmap(width, height); err != nil {
		return nil, fmt.Errorf("jbig2: %w", err)
	}
	if err := d.run(globals); err != nil {
		return nil, fmt.Errorf("jbig2: globals: %w", err)
	}
	if err := d.run(data); err != nil {
		return nil, fmt.Errorf("jbig2: %w", err)
	}
	return d.page.data, nil
}

// decoder is the per-call state: the page being composed, the symbol
// dictionaries decoded so far keyed by segment number (globals and page
// stream share one number space), and the remaining work budget.
type decoder struct {
	page     *bitmap
	dicts    map[uint32]*symbolDict
	work     int
	pageInfo bool // a page information segment has been seen
}

// charge deducts a w×h bitmap from the work budget, rejecting sizes
// beyond MaxPixels.
func (d *decoder) charge(w, h int) error {
	if w < 0 || h < 0 || w > MaxPixels || h > MaxPixels || (w > 0 && h > MaxPixels/w) {
		return fmt.Errorf("bitmap %dx%d exceeds limit", w, h)
	}
	d.work -= w * h
	if d.work < 0 {
		return errBudget
	}
	return nil
}

// segment is a parsed segment header (7.2) plus its data.
type segment struct {
	number   uint32
	typ      uint8
	referred []uint32
	data     []byte
}

// run parses and processes every segment of one embedded stream.
func (d *decoder) run(stream []byte) error {
	r := &reader{data: stream}
	for n := 0; r.remaining() > 0; n++ {
		if n >= maxSegments {
			return errors.New("too many segments")
		}
		seg, err := readSegment(r)
		if err != nil {
			return err
		}
		if err := d.process(seg); err != nil {
			return fmt.Errorf("segment %d (type %d): %w", seg.number, seg.typ, err)
		}
		if seg.typ == segEndOfFile {
			return nil
		}
	}
	return nil
}

// readSegment parses one segment header (7.2.2–7.2.7) and slices off its
// data, resolving the unknown-length case of 7.2.7 for immediate generic
// regions.
func readSegment(r *reader) (*segment, error) {
	seg := &segment{}
	var err error
	if seg.number, err = r.u32(); err != nil {
		return nil, err
	}
	flags, err := r.u8()
	if err != nil {
		return nil, err
	}
	seg.typ = flags & 0x3F
	pageAssoc4 := flags&0x40 != 0

	rts, err := r.u8()
	if err != nil {
		return nil, err
	}
	count := int(rts >> 5)
	if count == 7 {
		// Long form: 29-bit count then ceil((count+1)/8) retain-bit bytes.
		r.pos--
		v, err := r.u32()
		if err != nil {
			return nil, err
		}
		count = int(v & 0x1FFFFFFF)
		if _, err := r.bytes((count + 8) / 8); err != nil {
			return nil, err
		}
	}
	refSize := 1
	switch {
	case seg.number > 65536:
		refSize = 4
	case seg.number > 256:
		refSize = 2
	}
	if count > r.remaining()/refSize {
		return nil, errTruncated
	}
	seg.referred = make([]uint32, count)
	for i := range seg.referred {
		switch refSize {
		case 1:
			v, err := r.u8()
			if err != nil {
				return nil, err
			}
			seg.referred[i] = uint32(v)
		case 2:
			v, err := r.u16()
			if err != nil {
				return nil, err
			}
			seg.referred[i] = uint32(v)
		default:
			if seg.referred[i], err = r.u32(); err != nil {
				return nil, err
			}
		}
	}
	if pageAssoc4 {
		_, err = r.u32()
	} else {
		_, err = r.u8()
	}
	if err != nil {
		return nil, err
	}
	length, err := r.u32()
	if err != nil {
		return nil, err
	}
	if length == 0xFFFFFFFF {
		n, err := unknownLength(r.rest(), seg.typ)
		if err != nil {
			return nil, err
		}
		seg.data, err = r.bytes(n)
		return seg, err
	}
	if int64(length) > int64(r.remaining()) {
		return nil, fmt.Errorf("segment %d data length %d exceeds remaining %d bytes", seg.number, length, r.remaining())
	}
	seg.data, err = r.bytes(int(length))
	return seg, err
}

// unknownLength finds the extent of an immediate generic region whose
// header declared the 0xFFFFFFFF length (7.2.7): after the region segment
// info and flags, the data ends with a terminator (0xFF 0xAC for MQ, 0x00
// 0x00 for MMR) followed by a four-byte row count.
func unknownLength(rest []byte, typ uint8) (int, error) {
	if typ != segGenericImmediate && typ != segGenericLossless {
		return 0, errors.New("unknown data length on a non-generic-region segment")
	}
	const hdr = regionInfoSize + 1
	if len(rest) < hdr {
		return 0, errTruncated
	}
	term := []byte{0xFF, 0xAC}
	if rest[regionInfoSize]&1 != 0 { // MMR
		term = []byte{0x00, 0x00}
	}
	i := bytes.Index(rest[hdr:], term)
	if i < 0 || hdr+i+2+4 > len(rest) {
		return 0, errors.New("unterminated unknown-length generic region")
	}
	return hdr + i + 2 + 4, nil
}

// process dispatches one segment by type.
func (d *decoder) process(seg *segment) error {
	switch seg.typ {
	case segSymbolDict:
		sd, err := d.decodeSymbolDict(seg.data, d.referredDicts(seg))
		if err != nil {
			return err
		}
		d.dicts[seg.number] = sd
		return nil
	case segTextRegionIntermediate, segTextRegionImmediate, segTextRegionLossless:
		bm, info, err := d.decodeTextRegion(seg.data, d.referredDicts(seg))
		if err != nil {
			return err
		}
		return d.composeRegion(bm, info)
	case segGenericIntermediate, segGenericImmediate, segGenericLossless:
		bm, info, err := d.decodeGenericRegion(seg.data)
		if err != nil {
			return err
		}
		return d.composeRegion(bm, info)
	case segPageInfo:
		return d.readPageInfo(seg.data)
	case segPatternDict, segHalftoneIntermediate, segHalftoneImmediate, segHalftoneLossless:
		return fmt.Errorf("halftone region: %w", errors.ErrUnsupported)
	case segRefinementIntermediate, segRefinementImmediate, segRefinementLossless:
		return fmt.Errorf("refinement region: %w", errors.ErrUnsupported)
	case segColourPalette:
		return fmt.Errorf("colour extension: %w", errors.ErrUnsupported)
	default:
		// End of page/stripe/file, tables, profiles, extensions and
		// unknown types carry nothing the page needs (tables only matter
		// to Huffman-coded regions, which are unsupported anyway).
		return nil
	}
}

// referredDicts returns the symbol dictionaries among seg's referred-to
// segments, in referral order. Other referred segments (tables) are
// ignored.
func (d *decoder) referredDicts(seg *segment) []*symbolDict {
	var out []*symbolDict
	for _, n := range seg.referred {
		if sd, ok := d.dicts[n]; ok {
			out = append(out, sd)
		}
	}
	return out
}

// readPageInfo handles a page information segment (7.4.8). The PDF image
// dictionary already fixed the page size, so only the default pixel value
// matters here; the default combination operator is informational because
// each region carries its own external operator (8.2 requires them to
// agree unless overriding is allowed).
func (d *decoder) readPageInfo(data []byte) error {
	r := &reader{data: data}
	for range 4 { // width, height, x resolution, y resolution
		if _, err := r.u32(); err != nil {
			return err
		}
	}
	flags, err := r.u8()
	if err != nil {
		return err
	}
	if flags&0x80 != 0 {
		return fmt.Errorf("page colour extension: %w", errors.ErrUnsupported)
	}
	if d.pageInfo {
		// A second page in one embedded stream is not meaningful for a
		// PDF image; keep composing onto the same page.
		return nil
	}
	d.pageInfo = true
	if flags>>2&1 != 0 {
		d.page.fill(1)
	}
	return nil
}

// regionInfo is the region segment information field (7.4.1).
type regionInfo struct {
	w, h int
	x, y int
	op   combOp
}

const regionInfoSize = 17

func readRegionInfo(r *reader) (regionInfo, error) {
	var ri regionInfo
	fields := [4]uint32{}
	for i := range fields {
		v, err := r.u32()
		if err != nil {
			return ri, err
		}
		fields[i] = v
	}
	flags, err := r.u8()
	if err != nil {
		return ri, err
	}
	if flags&0x08 != 0 {
		return ri, fmt.Errorf("region colour extension: %w", errors.ErrUnsupported)
	}
	if ri.w, err = dim(fields[0], "region width"); err != nil {
		return ri, err
	}
	if fields[1] != 0xFFFFFFFF { // unknown height, resolved by the row count (7.2.7)
		if ri.h, err = dim(fields[1], "region height"); err != nil {
			return ri, err
		}
	} else {
		ri.h = -1
	}
	// Locations are unsigned 32-bit; anything beyond MaxPixels cannot
	// intersect the page, so clamp rather than overflow.
	ri.x = int(min(fields[2], MaxPixels))
	ri.y = int(min(fields[3], MaxPixels))
	ri.op = combOp(flags & 7)
	if ri.op > opReplace {
		return ri, fmt.Errorf("invalid combination operator %d", ri.op)
	}
	return ri, nil
}

// visibleRows returns how many of a region's rows can reach the page.
func (d *decoder) visibleRows(info *regionInfo) int {
	if info.x >= d.page.w || info.y >= d.page.h {
		return 0
	}
	return min(info.h, d.page.h-info.y)
}

// composeRegion draws a decoded region onto the page (8.2 step 5).
func (d *decoder) composeRegion(bm *bitmap, info *regionInfo) error {
	d.work -= d.page.compose(bm, info.x, info.y, info.op)
	if d.work < 0 {
		return errBudget
	}
	return nil
}

// decodeGenericRegion parses a generic region segment (7.4.6) and decodes
// its bitmap with MMR or the arithmetic procedure.
func (d *decoder) decodeGenericRegion(data []byte) (*bitmap, *regionInfo, error) {
	r := &reader{data: data}
	info, err := readRegionInfo(r)
	if err != nil {
		return nil, nil, err
	}
	flags, err := r.u8()
	if err != nil {
		return nil, nil, err
	}
	mmr := flags&1 != 0
	gp := genericParams{template: flags >> 1 & 3, tpgdon: flags>>3&1 != 0}
	if flags&0x10 != 0 {
		return nil, nil, fmt.Errorf("extended generic template: %w", errors.ErrUnsupported)
	}
	if !mmr {
		nAT := 1
		if gp.template == 0 {
			nAT = 4
		}
		for i := range nAT {
			if gp.at[i][0], err = r.i8(); err != nil {
				return nil, nil, err
			}
			if gp.at[i][1], err = r.i8(); err != nil {
				return nil, nil, err
			}
		}
	}
	body := r.rest()
	if info.h < 0 {
		// Unknown height (7.2.7): the true row count trails the
		// terminator, which readSegment included in the data.
		if len(body) < 4 {
			return nil, nil, errTruncated
		}
		rows, err := dim(uint32(body[len(body)-4])<<24|uint32(body[len(body)-3])<<16|uint32(body[len(body)-2])<<8|uint32(body[len(body)-1]), "region row count")
		if err != nil {
			return nil, nil, err
		}
		info.h = rows
		body = body[:len(body)-4]
	}
	// Rows below the page's bottom edge (or any row of a region starting
	// right of the page) never influence visible pixels: a pixel's context
	// only reaches upward, so decoding stops at the last visible row. This
	// keeps the cost of hostile region sizes tied to the page.
	info.h = d.visibleRows(&info)
	if err := d.charge(info.w, info.h); err != nil {
		return nil, nil, err
	}
	if mmr {
		bm, err := decodeMMR(body, info.w, info.h)
		if err != nil {
			return nil, nil, err
		}
		return bm, &info, nil
	}
	gp.w, gp.h = info.w, info.h
	bm, err := decodeGeneric(&gp, newMQDecoder(body), make([]mqCx, 1<<16))
	if err != nil {
		return nil, nil, err
	}
	return bm, &info, nil
}

// decodeMMR decodes an MMR-coded (T.6) generic region (6.2.6) through the
// CCITT Group 4 decoder; its packed-row output is already the bitmap
// layout used here.
func decodeMMR(data []byte, w, h int) (*bitmap, error) {
	bm, err := newBitmap(w, h)
	if err != nil {
		return nil, err
	}
	if w == 0 || h == 0 {
		return bm, nil
	}
	out, rows, err := ccitt.Decode(data, ccitt.Options{K: -1, Columns: w, Rows: h, BlackIs1: true, MaxPixels: MaxPixels})
	if err != nil && (!errors.Is(err, io.ErrUnexpectedEOF) || rows == 0) {
		return nil, fmt.Errorf("MMR: %w", err)
	}
	// Data that ends before the declared height (EOFB or truncation)
	// leaves the remaining rows white rather than failing the page.
	copy(bm.data, out[:min(len(out), min(rows, h)*bm.stride)])
	return bm, nil
}
