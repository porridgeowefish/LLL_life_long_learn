package object

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/giraffesyo/pdf/internal/safeio"
)

// readXref locates and parses the cross-reference data, following /Prev
// and hybrid /XRefStm chains. On any failure it falls back to rebuilding
// the table by scanning the whole file.
func (r *Reader) readXref() error {
	r.xref = nil // growXref sizes it from the first subsection
	start, err := r.startxref()
	if err == nil {
		if err = r.readXrefSection(start, map[int64]bool{}); err == nil && r.trailer != nil {
			return nil
		}
	}
	// Corrupt or missing xref: reconstruct from object headers.
	return r.rebuildXref()
}

// startxref reads the trailing "startxref <offset>" pointer.
func (r *Reader) startxref() (int64, error) {
	n := min(int64(2048), r.size)
	tail := r.slice(r.size-n, n)
	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, errors.New("pdf: no startxref")
	}
	lex := newLexer(tail[idx+len("startxref"):])
	t := lex.next()
	if t.kind != tokInt || t.num < 0 || t.num >= r.size {
		return 0, errors.New("pdf: bad startxref offset")
	}
	return t.num, nil
}

// readXrefSection parses one xref section (classic table or xref stream)
// at off and follows its /Prev and /XRefStm links. First-seen entries win,
// so newer sections (read first) shadow older ones.
func (r *Reader) readXrefSection(off int64, seen map[int64]bool) error {
	for len(seen) < maxPrevChain {
		if off < 0 || off >= r.size || seen[off] {
			return nil
		}
		seen[off] = true

		w := r.slice(off, 16)
		var trailer dict
		var prev, xrefStm int64
		var err error
		if bytes.HasPrefix(bytes.TrimLeft(w, " \t\r\n"), []byte("xref")) {
			trailer, err = r.readClassicXref(off)
		} else {
			trailer, err = r.readXrefStream(off)
		}
		if err != nil {
			return err
		}
		if r.trailer == nil {
			r.trailer = trailer
		}
		if v, ok := trailer["XRefStm"]; ok { // hybrid-reference file
			if n, ok := v.(int64); ok {
				xrefStm = n
			}
		}
		if v, ok := trailer["Prev"]; ok {
			if n, ok := v.(int64); ok {
				prev = n
			}
		}
		if xrefStm > 0 {
			// Process the xref stream before /Prev so its entries take
			// precedence over the older classic sections it supplements.
			_ = r.readXrefSection(xrefStm, seen)
		}
		if prev <= 0 {
			return nil
		}
		off = prev
	}
	return nil
}

// readClassicXref parses a classic "xref" table and its trailer.
func (r *Reader) readClassicXref(off int64) (dict, error) {
	limit := min(r.size-off, int64(safeio.MaxStreamBytes))
	buf := r.slice(off, limit)
	// Skip the "xref" keyword.
	lex := newLexer(buf)
	if t := lex.next(); t.kind != tokKeyword || t.str != "xref" {
		return nil, errors.New("pdf: expected xref keyword")
	}

	for {
		start := lex.pos
		t := lex.next()
		if t.kind == tokKeyword && t.str == "trailer" {
			break
		}
		if t.kind != tokInt {
			return nil, errors.New("pdf: malformed xref subsection")
		}
		count := lex.next()
		if count.kind != tokInt {
			return nil, errors.New("pdf: malformed xref subsection header")
		}
		first, num := int(t.num), int(count.num)
		if num < 0 || first < 0 || first+num > maxObjects {
			return nil, fmt.Errorf("pdf: xref subsection %d+%d exceeds limits", first, num)
		}
		r.growXref(first + num)
		for i := range num {
			offTok := lex.next()
			genTok := lex.next()
			typeTok := lex.next()
			if offTok.kind != tokInt || genTok.kind != tokInt || typeTok.kind != tokKeyword {
				return nil, errors.New("pdf: malformed xref entry")
			}
			id := first + i
			if typeTok.str == "n" && r.xref[id].kind == 0 {
				r.xref[id] = xrefEntry{kind: 1, offset: offTok.num, gen: int(genTok.num)}
			}
		}
		_ = start
	}

	// Parse the trailer dictionary.
	p := &objParser{lex: lex, r: r}
	tok := lex.next()
	if tok.kind != tokDictOpen {
		return nil, errors.New("pdf: xref trailer is not a dictionary")
	}
	d, err := p.parseDict(0)
	if err != nil {
		return nil, err
	}
	td, _ := d.(dict)
	return td, nil
}

// readXrefStream parses a PDF 1.5 cross-reference stream at off.
func (r *Reader) readXrefStream(off int64) (dict, error) {
	obj, err := r.parseObjectAt(off, r.objNumAt(off), 0)
	if err != nil {
		return nil, err
	}
	s, ok := obj.(*stream)
	if !ok {
		return nil, errors.New("pdf: xref offset is not a stream")
	}
	data, err := r.rawStreamBytes(s)
	if err != nil {
		return nil, err
	}

	w0, w1, w2, err := xrefWidths(s.d)
	if err != nil {
		return nil, err
	}
	size := int(r.asInt(s.d["Size"]))
	index := xrefIndex(s.d, size)
	rowLen := w0 + w1 + w2
	if rowLen == 0 {
		return nil, errors.New("pdf: xref stream /W is all zero")
	}

	pos := 0
	for pair := 0; pair+1 < len(index); pair += 2 {
		first, count := index[pair], index[pair+1]
		if first < 0 || count < 0 || first+count > maxObjects {
			return nil, errors.New("pdf: xref stream /Index exceeds limits")
		}
		r.growXref(first + count)
		for i := range count {
			if pos+rowLen > len(data) {
				break
			}
			f0 := readField(data[pos:], w0, 1) // default type 1
			f1 := readField(data[pos+w0:], w1, 0)
			f2 := readField(data[pos+w0+w1:], w2, 0)
			pos += rowLen
			id := first + i
			if r.xref[id].kind != 0 {
				continue // first-seen wins
			}
			switch f0 {
			case 1:
				r.xref[id] = xrefEntry{kind: 1, offset: f1, gen: int(f2)}
			case 2:
				r.xref[id] = xrefEntry{kind: 2, stmNum: int(f1), stmIdx: int(f2)}
			}
		}
	}
	return s.d, nil
}

// rawStreamBytes decodes a stream that must not itself be decrypted
// (xref streams, object streams during bootstrap use the same path but
// object streams may be encrypted — see objStmObject).
func (r *Reader) rawStreamBytes(s *stream) ([]byte, error) {
	rc, err := r.streamReader(s)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return safeio.ReadAllGuarded(rc), nil
}

func xrefWidths(d dict) (int, int, int, error) {
	wv, ok := d["W"].([]Value)
	if !ok || len(wv) < 3 {
		return 0, 0, 0, errors.New("pdf: xref stream missing /W")
	}
	w := make([]int, 3)
	for i := range 3 {
		n, ok := wv[i].Int64()
		if !ok || n < 0 || n > 8 {
			return 0, 0, 0, errors.New("pdf: invalid /W field width")
		}
		w[i] = int(n)
	}
	return w[0], w[1], w[2], nil
}

func xrefIndex(d dict, size int) []int {
	if iv, ok := d["Index"].([]Value); ok {
		idx := make([]int, 0, len(iv))
		for _, e := range iv {
			n, _ := e.Int64()
			idx = append(idx, int(n))
		}
		if len(idx) >= 2 {
			return idx
		}
	}
	return []int{0, size}
}

// readField reads a big-endian integer of w bytes; a zero-width field
// takes the given default.
func readField(b []byte, w int, def int64) int64 {
	if w == 0 {
		return def
	}
	var v int64
	for i := 0; i < w && i < len(b); i++ {
		v = v<<8 | int64(b[i])
	}
	return v
}

// growXref extends the xref slice to hold object number n-1.
func (r *Reader) growXref(n int) {
	if n > maxObjects {
		n = maxObjects
	}
	if n > len(r.xref) {
		// One allocation per subsection: appending entry by entry from the
		// initial 1024 reallocated the table many times for large files.
		r.xref = append(r.xref, make([]xrefEntry, n-len(r.xref))...)
	}
}

// objNumAt reads the object number of the "N G obj" header at off, for
// xref streams whose own number we don't yet know.
func (r *Reader) objNumAt(off int64) int {
	lex := newLexer(r.slice(off, 64))
	if t := lex.next(); t.kind == tokInt {
		return int(t.num)
	}
	return 0
}

// rebuildXref reconstructs the xref table by scanning the whole file for
// "N G obj" headers (last occurrence wins, matching incremental updates)
// and locating a usable trailer.
func (r *Reader) rebuildXref() error {
	data := r.slice(0, r.size)
	r.xref = r.xref[:0]
	r.cache = map[int]any{}

	headers := objHeaderRE(data)
	bound := rebuildObjectBound(len(headers))
	for _, m := range headers {
		if m.num >= bound {
			continue
		}
		r.growXref(m.num + 1)
		r.xref[m.num] = xrefEntry{kind: 1, offset: m.off, gen: m.gen}
	}
	r.indexObjectStreams(data, headers, bound)
	// Prefer a real trailer; otherwise synthesize one from a located
	// /Catalog so page navigation still works.
	if td := r.lastTrailer(data); td != nil {
		r.trailer = td
	} else if root := r.findCatalog(); root > 0 {
		r.trailer = dict{"Root": ref{root, 0}}
	} else {
		return errors.New("pdf: no cross-reference data and no catalog found")
	}
	return nil
}

// indexObjectStreams adds the objects packed in the file's object streams
// to a rebuilt table: a header scan sees only direct objects, and a PDF
// 1.5 file keeps most of its objects, the catalog and page tree among
// them, in object streams. A direct object keeps precedence, and among
// streams the one later in the file wins, as incremental updates append.
// Streams that do not decode — encrypted ones, before the key exists —
// are skipped.
func (r *Reader) indexObjectStreams(data []byte, headers []objHeader, bound int) {
	direct := make(map[int]bool, len(headers))
	for _, m := range headers {
		direct[m.num] = true
	}
	for _, m := range headers {
		if m.num >= bound || r.xref[m.num].offset != m.off {
			continue // not the live definition of its number
		}
		window := data[m.off:min(int64(len(data)), m.off+256)]
		if !bytes.Contains(window, []byte("ObjStm")) {
			continue
		}
		os, err := r.loadObjStm(m.num)
		if err != nil {
			continue
		}
		for idx, num := range os.nums {
			if num <= 0 || num >= bound || direct[num] {
				continue
			}
			r.growXref(num + 1)
			r.xref[num] = xrefEntry{kind: 2, stmNum: m.num, stmIdx: idx}
		}
	}
}

// Repair rebuilds the cross-reference table by scanning the file for
// object headers, for a table that parsed but locates objects wrongly —
// offsets that land on another object or none, or a /Prev chain that
// loops past the sections it needed. Objects the scan cannot see, those
// packed in object streams, keep their previous entries. Repair resets
// the object caches and error state, and keeps the decryption key; call
// it before cloning. On failure r is left as it was.
func (r *Reader) Repair() error {
	old := slices.Clone(r.xref)
	oldTrailer := r.trailer
	if err := r.rebuildXref(); err != nil {
		r.xref, r.trailer = old, oldTrailer
		return err
	}
	for num, e := range old {
		if e.kind == 2 && (num >= len(r.xref) || r.xref[num].kind == 0) {
			r.growXref(num + 1)
			r.xref[num] = e
		}
	}
	r.cache = map[int]any{}
	r.resolving = map[int]bool{}
	r.objStms = newObjStmCache()
	r.err = nil
	return nil
}

// rebuildObjectBound is the highest object number (exclusive) a rebuild
// trusts from a file holding n object headers. Object numbers of a real
// file track how many objects it has, with room for the gaps incremental
// updates leave; a header claiming a number far beyond that is damage
// or hostility, and admitting it would size the table — forty bytes an
// entry, up to maxObjects entries — for an object that does not exist.
func rebuildObjectBound(n int) int {
	const (
		minBound = 4096 // small files may still number sparsely
		slack    = 64   // entries per header present
	)
	return min(maxObjects, max(minBound, slack*n))
}

type objHeader struct {
	num, gen int
	off      int64
}

// objHeaderRE scans for "<int> <int> obj" headers without a regexp
// dependency.
func objHeaderRE(data []byte) []objHeader {
	var out []objHeader
	i := 0
	for {
		j := bytes.Index(data[i:], []byte(" obj"))
		if j < 0 {
			return out
		}
		at := i + j
		i = at + 4
		// Walk backward over "<num> <gen>".
		k := at
		gen, ok1, k := scanIntBackward(data, k)
		num, ok2, start := scanIntBackward(data, k)
		if ok1 && ok2 {
			out = append(out, objHeader{num: num, gen: gen, off: int64(start)})
		}
	}
}

// scanIntBackward reads a run of digits (with surrounding spaces) ending
// at pos-1, returning the value and the index where it starts.
func scanIntBackward(data []byte, pos int) (val int, ok bool, start int) {
	for pos > 0 && (data[pos-1] == ' ' || data[pos-1] == '\r' || data[pos-1] == '\n' || data[pos-1] == '\t') {
		pos--
	}
	end := pos
	for pos > 0 && data[pos-1] >= '0' && data[pos-1] <= '9' {
		pos--
	}
	if pos == end {
		return 0, false, pos
	}
	n, err := strconv.Atoi(string(data[pos:end]))
	if err != nil {
		return 0, false, pos
	}
	return n, true, pos
}

// lastTrailer returns the last "trailer <<…>>" dictionary in the file.
func (r *Reader) lastTrailer(data []byte) dict {
	idx := bytes.LastIndex(data, []byte("trailer"))
	if idx < 0 {
		return nil
	}
	lex := newLexer(data[idx+len("trailer"):])
	if t := lex.next(); t.kind != tokDictOpen {
		return nil
	}
	p := &objParser{lex: lex, r: r}
	d, err := p.parseDict(0)
	if err != nil {
		return nil
	}
	td, _ := d.(dict)
	if _, ok := td["Root"]; !ok {
		return nil
	}
	return td
}

// findCatalog locates a /Type /Catalog object by scanning resolved
// objects (used only when no trailer is available).
func (r *Reader) findCatalog() int {
	for num := 1; num < len(r.xref); num++ {
		if r.xref[num].kind == 0 {
			continue // nothing to parse; object() would only allocate an error
		}
		obj, err := r.object(num, r.xref[num].gen)
		if err != nil {
			continue
		}
		if d, ok := obj.(dict); ok {
			if n, ok := d["Type"].(name); ok && n == "Catalog" {
				return num
			}
		}
	}
	return 0
}
