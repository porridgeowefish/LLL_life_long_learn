// Package ccitt decodes CCITT Group 3 and Group 4 fax coding: the
// one-dimensional modified Huffman (MH) and two-dimensional modified READ
// (MR) schemes of ITU-T T.4 and the modified modified READ (MMR) scheme
// of ITU-T T.6. This is the coding used by the PDF CCITTFaxDecode filter
// (ISO 32000-1 §7.4.6) and by MMR-coded regions in JBIG2 (T.88 §6.2.6).
//
// The decoder is written from the T.4/T.6 code tables and the PDF filter
// parameters alone, in pure Go. It is deliberately permissive where real
// encoders disagree: EOL codes and fill bits are optional in every mode,
// rows whose runs overflow the width are clamped, and the 2-D/1-D tag bit
// in mixed mode (K > 0) is read after an EOL when one is present and at
// the row start otherwise. Malformed input never panics and never causes
// unbounded allocation.
package ccitt

import (
	"errors"
	"fmt"
	"io"
)

// DefaultMaxPixels bounds the output size when the row count is unknown.
const DefaultMaxPixels = 1 << 28

// maxColumns rejects absurd widths before any allocation happens.
const maxColumns = 1 << 20

// Options mirrors the CCITTFaxDecode /DecodeParms (ISO 32000-1 §7.4.6).
// The zero value decodes pure 1-D Group 3 at 1728 columns, black as 0 bits.
type Options struct {
	K                int  // <0: pure two-dimensional (Group 4); 0: one-dimensional MH; >0: mixed (Group 3 2-D)
	Columns          int  // pixels per row; 0 means the spec default 1728
	Rows             int  // expected rows; 0 means unknown (decode until EOFB/RTC or end of data)
	BlackIs1         bool // emit black pixels as 1 bits (PDF default false: 0 bits are black)
	EncodedByteAlign bool
	MaxPixels        int // cap on Columns*rows when Rows is 0; 0 means DefaultMaxPixels
}

var (
	errBadCode      = errors.New("invalid code")
	errBadMode      = errors.New("invalid mode code")
	errExtension    = errors.New("extension (uncompressed) mode is not supported")
	errBackwards    = errors.New("changing element moves left of a0")
	errTooManyRuns  = errors.New("too many changing elements in row")
	errTooManyRows  = errors.New("decoded image exceeds MaxPixels")
	errColumns      = errors.New("columns must be between 1 and 1<<20")
	errNegativeRows = errors.New("rows must not be negative")
)

// Decode decodes data and returns the rows as packed 1-bit-per-pixel
// bitmaps, each row padded to a byte boundary (stride = (Columns+7)/8),
// together with the number of rows decoded. Bits are MSB first, the
// leftmost pixel being bit 7 of the row's first byte; padding bits are
// zero. Malformed or truncated input returns the rows decoded so far
// together with a non-nil error; truncation wraps io.ErrUnexpectedEOF.
// Decode never panics and its memory use is bounded by Columns × Rows (or
// by MaxPixels when Rows is unknown).
func Decode(data []byte, o Options) (bitmap []byte, rows int, err error) {
	if o.Columns == 0 {
		o.Columns = 1728
	}
	if o.Columns < 0 || o.Columns > maxColumns {
		return nil, 0, fmt.Errorf("ccitt: %w", errColumns)
	}
	if o.Rows < 0 {
		return nil, 0, fmt.Errorf("ccitt: %w", errNegativeRows)
	}
	if o.MaxPixels <= 0 {
		o.MaxPixels = DefaultMaxPixels
	}
	d := decoder{
		br:     bitReader{data: data, nbits: len(data) * 8},
		o:      o,
		cols:   o.Columns,
		stride: (o.Columns + 7) / 8,
	}
	// Changing-element buffers hold at most cols+4 entries (see decode2D).
	// Start small so absurd widths on tiny inputs stay cheap.
	initial := min(d.cols+4, 1024)
	d.ref = make([]int, 0, initial)
	d.cur = make([]int, 0, initial)

	for o.Rows == 0 || d.rows < o.Rows {
		if o.Rows == 0 && (d.rows+1)*d.cols > o.MaxPixels {
			return d.out, d.rows, fmt.Errorf("ccitt: row %d: %w", d.rows, errTooManyRows)
		}
		twoD, end, err := d.beginRow()
		if err != nil {
			return d.out, d.rows, fmt.Errorf("ccitt: row %d: %w", d.rows, err)
		}
		if end {
			if o.Rows != 0 {
				// Known row count but the data ran out: truncated or cut short by an EOFB/RTC.
				return d.out, d.rows, fmt.Errorf("ccitt: %d of %d rows: %w", d.rows, o.Rows, io.ErrUnexpectedEOF)
			}
			break
		}
		if twoD {
			err = d.decode2D()
		} else {
			err = d.decode1D()
		}
		if err != nil {
			return d.out, d.rows, fmt.Errorf("ccitt: row %d: %w", d.rows, err)
		}
		d.emitRow()
		d.ref, d.cur = d.cur, d.ref[:0]
		d.rows++
	}
	return d.out, d.rows, nil
}

type decoder struct {
	br     bitReader
	o      Options
	cols   int
	stride int
	rows   int
	out    []byte
	// ref and cur hold the changing elements of the reference and coding
	// lines: the pixel positions where the colour changes, alternating
	// starting with a change to black. Positions are in [0, cols].
	ref, cur []int
	// prevEOL records whether the previous row was introduced by an EOL,
	// which decides the order of alignment and EOL detection (see beginRow).
	prevEOL bool
	// noAlign is set when EncodedByteAlign was requested but the bits
	// being skipped are not zero: the producer lied, so stop aligning.
	noAlign bool
}

// beginRow consumes whatever separates rows — fill bits, EOL codes and
// the 1-D/2-D tag bit of mixed mode — and reports whether the next row is
// two-dimensional, or whether the data has ended (end of data, RTC or
// EOFB).
//
// EncodedByteAlign is ambiguous in the wild. The PDF wording says each
// row begins on a byte boundary; libtiff's "fill" option instead pads so
// that each EOL *ends* on one, and Adobe producers have shipped both with
// the flag set. Following xpdf: when the stream uses EOLs (the previous
// row had one) the fill bits before the EOL make alignment unnecessary and
// aligning first could land inside the EOL, so the EOL is looked for first;
// when it does not, the row is aligned first, which keeps a row starting
// with a seven-zero extended makeup code from being mistaken for an EOL.
func (d *decoder) beginRow() (twoD, end bool, err error) {
	aligned := false
	if d.o.EncodedByteAlign && !d.prevEOL {
		d.align()
		aligned = true
	}
	eols := 0
	twoD = d.o.K < 0
	for {
		// A window of 12 zero bits can only be fill (or trailing padding):
		// no code has more than seven leading zeros.
		for d.br.pos < d.br.nbits && d.br.peek(12) == 0 {
			d.br.pos++
		}
		if d.br.pos >= d.br.nbits {
			return false, true, nil
		}
		if d.br.peek(12) != 1 {
			break
		}
		d.br.pos += 12
		eols++
		// T.4 §4.2.1.3.1: in mixed mode the bit after EOL tags the next
		// row, 1 = 1-D. An EOL directly after an EOL (an EOFB written
		// without tags) is not a tag, and no row can start with eleven
		// zero bits, so peeking for it is safe.
		if d.o.K > 0 && d.br.peek(12) != 1 {
			twoD = d.br.peek(1) == 0
			d.br.pos++
		}
	}
	d.prevEOL = eols > 0
	if eols >= 2 {
		// RTC (6×EOL, G3) or EOFB (2×EOL, G4). Two are enough to be sure.
		return false, true, nil
	}
	if eols == 0 {
		if d.o.EncodedByteAlign && !aligned {
			d.align()
		}
		if d.o.K > 0 {
			// No EOL, so the tag bit stands alone at the row start. T.4 makes
			// the EOL mandatory, but PDF producers omit it and every major
			// reader reads the tag anyway.
			if d.br.pos >= d.br.nbits {
				return false, true, nil
			}
			twoD = d.br.peek(1) == 0
			d.br.pos++
		}
	}
	return twoD, false, nil
}

// align skips to the next byte boundary, but only over zero bits: non-zero
// bits mean the data is not actually aligned, in which case alignment is
// abandoned for the rest of the stream.
func (d *decoder) align() {
	if d.noAlign {
		return
	}
	n := -d.br.pos & 7
	if n == 0 {
		return
	}
	if d.br.peek(uint(n)) != 0 {
		d.noAlign = true
		return
	}
	d.br.pos += n
}

// decode1D decodes a modified Huffman row: alternating white and black
// run lengths starting with white.
func (d *decoder) decode1D() error {
	cols := d.cols
	cur := d.cur[:0]
	pos := 0
	white := true
	for pos < cols {
		if len(cur) > cols+2 {
			return errTooManyRuns
		}
		run, err := d.readRun(white)
		if err != nil {
			return err
		}
		// Clamp rather than fail: over-long rows are common in real fax data.
		pos = min(pos+run, cols)
		cur = append(cur, pos)
		white = !white
	}
	d.cur = cur
	return nil
}

// decode2D decodes a modified READ row against the reference line d.ref
// (T.4 §4.2.1.3, T.6 §2.2). a0 starts on an imaginary white element just
// before the row; b1 is the first changing element of the reference line
// right of a0 with the opposite colour of a0, b2 the one after it.
func (d *decoder) decode2D() error {
	cols := d.cols
	ref := d.ref
	cur := d.cur[:0]
	a0 := -1
	color := 0 // 0 white, 1 black; always equals len(cur)&1
	ri := 0    // index of the first reference element right of a0; a0 never decreases
	for a0 < cols {
		if len(cur) > cols+2 {
			return errTooManyRuns
		}
		for ri < len(ref) && ref[ri] <= a0 {
			ri++
		}
		// Even indices are changes to black, whose colour is the opposite of
		// white, so b1's index must have the parity of the current colour.
		bi := ri
		if bi&1 != color {
			bi++
		}
		b1, b2 := cols, cols
		if bi < len(ref) {
			b1 = ref[bi]
			if bi+1 < len(ref) {
				b2 = ref[bi+1]
			}
		}

		e := modeTable[d.br.peek(7)]
		if e.n == 0 {
			if d.br.nbits-d.br.pos < 7 {
				return io.ErrUnexpectedEOF
			}
			return errBadMode
		}
		d.br.pos += int(e.n)
		if d.br.pos > d.br.nbits {
			return io.ErrUnexpectedEOF
		}
		switch e.run {
		case modePass:
			a0 = b2
		case modeHoriz:
			run1, err := d.readRun(color == 0)
			if err != nil {
				return err
			}
			run2, err := d.readRun(color != 0)
			if err != nil {
				return err
			}
			a1 := min(max(a0, 0)+run1, cols)
			a2 := min(a1+run2, cols)
			cur = append(cur, a1, a2)
			a0 = a2
		case modeExt:
			return errExtension
		default:
			a1 := b1 + int(e.run) - modeV0
			if a1 < 0 || a1 < a0 {
				return errBackwards
			}
			a1 = min(a1, cols)
			cur = append(cur, a1)
			a0 = a1
			color ^= 1
		}
	}
	d.cur = cur
	return nil
}

// readRun reads one complete run length of the given colour: any number
// of makeup codes followed by a terminating code.
func (d *decoder) readRun(white bool) (int, error) {
	tbl, width := blackTable, uint(blackBits)
	if white {
		tbl, width = whiteTable, uint(whiteBits)
	}
	total := 0
	for {
		e := tbl[d.br.peek(width)]
		if e.n == 0 {
			if d.br.nbits-d.br.pos < int(width) {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, errBadCode
		}
		d.br.pos += int(e.n)
		if d.br.pos > d.br.nbits {
			return 0, io.ErrUnexpectedEOF
		}
		total += int(e.run)
		if e.run < 64 {
			return total, nil
		}
	}
}

// emitRow appends the coding line d.cur to the output as packed pixels.
func (d *decoder) emitRow() {
	if len(d.out)+d.stride > cap(d.out) {
		d.out = growOut(d.out, d.stride, d.o.Rows, len(d.br.data))
	}
	row := d.out[len(d.out) : len(d.out)+d.stride]
	d.out = d.out[:len(d.out)+d.stride]
	clear(row)
	cur := d.cur
	for i := 0; i+1 < len(cur); i += 2 {
		setBits(row, cur[i], cur[i+1])
	}
	if len(cur)&1 == 1 {
		setBits(row, cur[len(cur)-1], d.cols)
	}
	if !d.o.BlackIs1 {
		for i := range row {
			row[i] = ^row[i]
		}
		if pad := d.cols & 7; pad != 0 {
			row[len(row)-1] &= 0xFF << (8 - uint(pad))
		}
	}
}

// growOut grows the output buffer geometrically. The first allocation is
// sized from the input length (fax coding rarely expands more than 256×
// outside runs of blank rows) and the capacity is capped at the declared
// row count, so a hostile Rows never allocates up front more than the data
// can plausibly fill.
func growOut(out []byte, stride, rows, inputLen int) []byte {
	need := len(out) + stride
	newCap := max(2*cap(out), need, 16*stride)
	if cap(out) == 0 {
		newCap = max(newCap, 256*inputLen)
	}
	if rows > 0 {
		newCap = min(newCap, rows*stride)
	}
	grown := make([]byte, len(out), newCap)
	copy(grown, out)
	return grown
}

// setBits sets the pixel bits in [from, to) of a packed row to 1.
func setBits(row []byte, from, to int) {
	if from >= to {
		return
	}
	first, last := from>>3, (to-1)>>3
	firstMask := byte(0xFF) >> (uint(from) & 7)
	lastMask := byte(0xFF) << (7 - (uint(to-1) & 7))
	if first == last {
		row[first] |= firstMask & lastMask
		return
	}
	row[first] |= firstMask
	for i := first + 1; i < last; i++ {
		row[i] = 0xFF
	}
	row[last] |= lastMask
}

// bitReader reads MSB-first bits. Reads past the end yield zero bits, so
// callers detect truncation by comparing pos with nbits afterwards.
type bitReader struct {
	data  []byte
	pos   int // next bit to read
	nbits int
}

// peek returns the next n bits (n <= 24) without consuming them.
func (b *bitReader) peek(n uint) uint32 {
	i := b.pos >> 3
	var w uint32
	if i+4 <= len(b.data) {
		w = uint32(b.data[i])<<24 | uint32(b.data[i+1])<<16 | uint32(b.data[i+2])<<8 | uint32(b.data[i+3])
	} else {
		for j := range 4 {
			w <<= 8
			if i+j < len(b.data) {
				w |= uint32(b.data[i+j])
			}
		}
	}
	return w << (uint(b.pos) & 7) >> (32 - n)
}
