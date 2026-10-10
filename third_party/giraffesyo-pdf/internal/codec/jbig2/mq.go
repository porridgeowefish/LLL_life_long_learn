package jbig2

// MQ arithmetic decoder (T.88 Annex E, shared with JPEG 2000) and the
// arithmetic integer decoding procedures of Annex A.

// qeEntry is one row of the Qe probability-estimation table (Table E.1).
type qeEntry struct {
	qe   uint32
	nmps uint8
	nlps uint8
	sw   uint8
}

var qeTable = [47]qeEntry{
	{0x5601, 1, 1, 1}, {0x3401, 2, 6, 0}, {0x1801, 3, 9, 0}, {0x0AC1, 4, 12, 0}, {0x0521, 5, 29, 0}, {0x0221, 38, 33, 0},
	{0x5601, 7, 6, 1}, {0x5401, 8, 14, 0}, {0x4801, 9, 14, 0}, {0x3801, 10, 14, 0}, {0x3001, 11, 17, 0}, {0x2401, 12, 18, 0},
	{0x1C01, 13, 20, 0}, {0x1601, 29, 21, 0}, {0x5601, 15, 14, 1}, {0x5401, 16, 14, 0}, {0x5101, 17, 15, 0}, {0x4801, 18, 16, 0},
	{0x3801, 19, 17, 0}, {0x3401, 20, 18, 0}, {0x3001, 21, 19, 0}, {0x2801, 22, 19, 0}, {0x2401, 23, 20, 0}, {0x2201, 24, 21, 0},
	{0x1C01, 25, 22, 0}, {0x1801, 26, 23, 0}, {0x1601, 27, 24, 0}, {0x1401, 28, 25, 0}, {0x1201, 29, 26, 0}, {0x1101, 30, 27, 0},
	{0x0AC1, 31, 28, 0}, {0x09C1, 32, 29, 0}, {0x08A1, 33, 30, 0}, {0x0521, 34, 31, 0}, {0x0441, 35, 32, 0}, {0x02A1, 36, 33, 0},
	{0x0221, 37, 34, 0}, {0x0141, 38, 35, 0}, {0x0111, 39, 36, 0}, {0x0085, 40, 37, 0}, {0x0049, 41, 38, 0}, {0x0025, 42, 39, 0},
	{0x0015, 43, 40, 0}, {0x0009, 44, 41, 0}, {0x0005, 45, 42, 0}, {0x0001, 45, 43, 0}, {0x5601, 46, 46, 0},
}

// mqCx is one adaptive context: its Qe table index and current MPS.
type mqCx struct {
	i   uint8
	mps uint8
}

// mqDecoder holds the decoder registers of the software conventions
// decoder (E.3.2 ff). C is kept as one 32-bit register whose upper half is
// Chigh; reading past the end of data yields 0xFF bytes, which the
// decoder treats as a marker, so exhausted input never stops a decode —
// callers bound every loop by declared dimensions and counts.
type mqDecoder struct {
	data []byte
	bp   int
	c    uint32
	a    uint32
	ct   int
}

func newMQDecoder(data []byte) *mqDecoder {
	d := &mqDecoder{data: data}
	d.c = uint32(d.byteAt(0)) << 16
	d.byteIn()
	d.c <<= 7
	d.ct -= 7
	d.a = 0x8000
	return d
}

func (d *mqDecoder) byteAt(i int) byte {
	if i < len(d.data) {
		return d.data[i]
	}
	return 0xFF
}

// byteIn implements BYTEIN (Figure E.19 / G.3): a 0xFF followed by a byte
// above 0x8F is a marker and is not consumed; otherwise a stuffed bit is
// skipped after 0xFF.
func (d *mqDecoder) byteIn() {
	if d.byteAt(d.bp) == 0xFF {
		if d.byteAt(d.bp+1) > 0x8F {
			d.c += 0xFF00
			d.ct = 8
		} else {
			d.bp++
			d.c += uint32(d.byteAt(d.bp)) << 9
			d.ct = 7
		}
	} else {
		d.bp++
		d.c += uint32(d.byteAt(d.bp)) << 8
		d.ct = 8
	}
}

// decode returns the next decision for context cx (DECODE, Figure E.17,
// with the MPS/LPS exchange and renormalisation inlined).
func (d *mqDecoder) decode(cx *mqCx) uint32 {
	q := &qeTable[cx.i]
	qe := q.qe
	d.a -= qe
	var bit uint32
	if (d.c >> 16) < qe {
		// LPS exchange (Figure E.17's LPS path)
		if d.a < qe {
			d.a = qe
			bit = uint32(cx.mps)
			cx.i = q.nmps
		} else {
			d.a = qe
			bit = uint32(1 - cx.mps)
			if q.sw == 1 {
				cx.mps = 1 - cx.mps
			}
			cx.i = q.nlps
		}
	} else {
		d.c -= qe << 16
		if d.a&0x8000 != 0 {
			return uint32(cx.mps)
		}
		// MPS exchange
		if d.a < qe {
			bit = uint32(1 - cx.mps)
			if q.sw == 1 {
				cx.mps = 1 - cx.mps
			}
			cx.i = q.nlps
		} else {
			bit = uint32(cx.mps)
			cx.i = q.nmps
		}
	}
	// RENORMD
	for {
		if d.ct == 0 {
			d.byteIn()
		}
		d.a <<= 1
		d.c <<= 1
		d.ct--
		if d.a&0x8000 != 0 {
			break
		}
	}
	return bit
}

// intClasses are the value widths and offsets of Table A.1, indexed by
// the number of leading 1 bits in the prefix.
var intClasses = [6]struct{ n, offset uint32 }{{2, 0}, {4, 4}, {6, 20}, {8, 84}, {12, 340}, {32, 4436}}

// intCtx is the 512-entry context set of one arithmetic integer decoding
// procedure (IADH, IADW, ... of Annex A).
type intCtx [512]mqCx

// decodeInt implements Annex A.2. ok is false for OOB.
func (d *mqDecoder) decodeInt(cx *intCtx) (v int32, ok bool) {
	prev := uint32(1)
	bit := func() uint32 {
		b := d.decode(&cx[prev])
		if prev < 256 {
			prev = prev<<1 | b
		} else {
			prev = (((prev << 1) | b) & 511) | 256
		}
		return b
	}
	s := bit()
	// The prefix of Table A.1: up to five 1 bits, terminated by a 0 (or
	// the fifth 1), select the value width and offset.
	k := 0
	for k < len(intClasses)-1 && bit() == 1 {
		k++
	}
	n, offset := intClasses[k].n, intClasses[k].offset
	var val uint32
	for range n {
		val = val<<1 | bit()
	}
	val += offset
	if s == 1 {
		if val == 0 {
			return 0, false // OOB
		}
		// Clamp so the negation cannot overflow int32 on hostile input.
		if val > 1<<31-1 {
			val = 1<<31 - 1
		}
		return -int32(val), true
	}
	if val > 1<<31-1 {
		val = 1<<31 - 1
	}
	return int32(val), true
}

// decodeIAID implements Annex A.3: a symCodeLen-bit symbol ID read with a
// binary tree of contexts. cx must have 1<<(symCodeLen+1) entries.
func (d *mqDecoder) decodeIAID(cx []mqCx, symCodeLen uint) uint32 {
	prev := uint32(1)
	for range symCodeLen {
		prev = prev<<1 | d.decode(&cx[prev])
	}
	return prev - 1<<symCodeLen
}
