package object

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/giraffesyo/pdf/internal/crypt"
	"github.com/giraffesyo/pdf/internal/safeio"
)

// xrefEntry locates one object: either at a byte offset in the file
// (kind 1) or packed inside an object stream (kind 2). Free entries
// (kind 0) are left zero.
type xrefEntry struct {
	kind   uint8 // 0 free, 1 offset, 2 in object stream
	offset int64 // kind 1
	stmNum int   // kind 2: object number of the containing ObjStm
	stmIdx int   // kind 2: index within it
	gen    int
}

// A Reader parses a PDF file's object graph. It is not safe for
// concurrent use.
type Reader struct {
	ra   io.ReaderAt
	size int64

	xref    []xrefEntry
	objBuf  []byte // parse window reused by parseObjectAt; see sliceInto
	trailer dict
	dec     *crypt.Decryptor
	encNum  int // object number of the /Encrypt dict (strings never decrypt)

	cache     map[int]any  // resolved objects, keyed by number
	resolving map[int]bool // objects mid-resolution, to break cycles
	objStms   *objStmCache
	err       error // first resolution error (sticky)
}

// NewReader parses the file's cross-reference data and trailer, preparing
// it for object access. It returns an error only for structural failures
// that make the document unreadable.
func NewReader(ra io.ReaderAt, size int64) (*Reader, error) {
	return NewReaderWithPassword(ra, size, nil)
}

// NewReaderWithPassword is NewReader with an optional user or owner password
// for documents using the standard security handler.
func NewReaderWithPassword(ra io.ReaderAt, size int64, password []byte) (*Reader, error) {
	if size <= 0 {
		return nil, errors.New("pdf: empty file")
	}
	base, err := headerOffset(ra, size)
	if err != nil {
		return nil, err
	}
	if base > 0 { // leading junk: shift so stored offsets line up
		ra = io.NewSectionReader(ra, base, size-base)
		size -= base
	}
	r := &Reader{
		ra:        ra,
		size:      size,
		cache:     map[int]any{},
		resolving: map[int]bool{},
		objStms:   newObjStmCache(),
	}
	if err := r.readXref(); err != nil {
		return nil, err
	}
	if err := r.initEncrypt(password); err != nil {
		return nil, err
	}
	return r, nil
}

// Clone returns a Reader over the same file that resolves objects
// independently of r. The cross-reference table, trailer, and decryption
// key are immutable once parsed and are shared; the object caches, parse
// window, and error state are not, so a clone may be used from another
// goroutine while r is in use. io.ReaderAt permits the parallel ReadAt
// calls this implies.
//
// A clone re-parses the objects it touches, so cloning is worthwhile to
// give a worker its own reader, not to read a single object.
func (r *Reader) Clone() *Reader {
	return &Reader{
		ra:        r.ra,
		size:      r.size,
		xref:      r.xref,
		trailer:   r.trailer,
		dec:       r.dec,
		encNum:    r.encNum,
		cache:     map[int]any{},
		resolving: map[int]bool{},
		objStms:   newObjStmCache(),
	}
}

// Object returns the indirect object with the given number as a value
// rooted in r, or null if r has no such object. It lets a caller that
// located an object through one Reader reach the same object through
// another — a clone, say — without walking to it again.
func (r *Reader) Object(num int) Value {
	if num < 0 || num >= len(r.xref) {
		return Value{}
	}
	return Value{r: r, data: ref{num: num, gen: r.xref[num].gen}}
}

// Trailer returns the document trailer dictionary.
func (r *Reader) Trailer() Value {
	return Value{r: r, data: r.trailer}
}

// Err returns the first object-resolution error observed, if any. Missing
// or malformed objects degrade to null during navigation; Err lets a
// caller distinguish a corrupt document from an intentionally sparse one.
func (r *Reader) Err() error { return r.err }

func (r *Reader) setErr(err error) {
	if r.err == nil && err != nil {
		r.err = err
	}
}

// readAt reads exactly len(p) bytes at off, or fewer at EOF.
func (r *Reader) readAt(p []byte, off int64) (int, error) {
	n, err := r.ra.ReadAt(p, off)
	if errors.Is(err, io.EOF) && n > 0 {
		return n, nil
	}
	return n, err
}

// slice reads [off, off+n) clamped to the file, for windowed lexing.
func (r *Reader) slice(off, n int64) []byte {
	if off < 0 || off >= r.size {
		return nil
	}
	if off+n > r.size {
		n = r.size - off
	}
	buf := make([]byte, n)
	m, _ := r.readAt(buf, off)
	return buf[:m]
}

// sliceInto is slice reading into *scratch, which it grows as needed and
// keeps for reuse. The lexer copies every string and name it produces,
// so a parsed object never aliases the window; parseObjectAt reuses one
// window per Reader instead of allocating 4 KiB for every object.
func (r *Reader) sliceInto(scratch *[]byte, off, n int64) []byte {
	if off < 0 || off >= r.size {
		return nil
	}
	if off+n > r.size {
		n = r.size - off
	}
	if int64(cap(*scratch)) < n {
		*scratch = make([]byte, n)
	}
	buf := (*scratch)[:n]
	m, _ := r.readAt(buf, off)
	return buf[:m]
}

// headerOffset finds the "%PDF-" marker within the first part of the
// file, tolerating leading junk. Both 1.x and 2.x are accepted.
func headerOffset(ra io.ReaderAt, size int64) (int64, error) {
	n := min(int64(1024), size)
	buf := make([]byte, n)
	m, err := ra.ReadAt(buf, 0)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	if err != nil {
		return 0, err
	}
	if i := bytes.Index(buf[:m], []byte("%PDF-")); i >= 0 {
		return int64(i), nil
	}
	// No header: assume offset 0 and let xref recovery try.
	return 0, nil
}

// object resolves an object by number, using the cache, the xref table,
// and (for packed objects) the object-stream cache.
func (r *Reader) object(num, gen int) (any, error) {
	if num < 0 || num >= len(r.xref) {
		return nil, fmt.Errorf("pdf: object %d out of range", num)
	}
	if v, ok := r.cache[num]; ok {
		return v, nil
	}
	// Break reference cycles: an object whose own parsing needs itself
	// (e.g. a stream /Length that points back at the stream object).
	if r.resolving[num] {
		return nil, fmt.Errorf("pdf: cyclic reference to object %d", num)
	}
	r.resolving[num] = true
	defer delete(r.resolving, num)

	e := r.xref[num]
	var (
		obj any
		err error
	)
	switch e.kind {
	case 1:
		obj, err = r.parseObjectAt(e.offset, num, gen)
	case 2:
		obj, err = r.objStmObject(e.stmNum, e.stmIdx, num)
	default:
		return nil, fmt.Errorf("pdf: object %d is free or missing", num)
	}
	if err != nil {
		return nil, err
	}
	r.cache[num] = obj
	return obj, nil
}

// parseObjectAt parses "N G obj … (endobj|stream)" at a file offset. For
// streams it records the absolute data offset without reading the data.
// Strings in a directly-parsed object are decrypted here (object-stream
// contents arrive already decrypted and skip this path).
func (r *Reader) parseObjectAt(off int64, wantNum, wantGen int) (any, error) {
	// Most objects are a few hundred bytes; start small and grow for the
	// rare large direct object rather than reading 64 KB per object.
	window := int64(4 << 10)
	for {
		buf := r.sliceInto(&r.objBuf, off, window)
		obj, consumed, isStream, err := r.parseIndirect(buf, off, wantNum, wantGen)
		if errors.Is(err, errNeedMore) && window < r.size-off {
			window *= 4
			continue
		}
		if err != nil {
			return nil, err
		}
		if isStream {
			return obj, nil
		}
		_ = consumed
		if r.dec != nil && wantNum != r.encNum {
			obj = r.decryptStrings(obj, wantNum, wantGen)
		}
		return obj, nil
	}
}

var errNeedMore = errors.New("pdf: object window too small")

// parseIndirect lexes the object header and body from buf (a window
// starting at file offset base). For streams it computes the absolute
// data offset and resolves /Length.
func (r *Reader) parseIndirect(buf []byte, base int64, wantNum, wantGen int) (obj any, consumed int, isStream bool, err error) {
	lex := newLexer(buf)
	n := lex.next()
	g := lex.next()
	kw := lex.next()
	if n.kind != tokInt || g.kind != tokInt || kw.kind != tokKeyword || kw.str != "obj" {
		return nil, 0, false, fmt.Errorf("pdf: object %d: bad header", wantNum)
	}
	if int(n.num) != wantNum {
		return nil, 0, false, fmt.Errorf("pdf: object %d: found %d instead", wantNum, n.num)
	}
	p := &objParser{lex: lex, r: r}
	val, perr := p.parseObject(0)
	if errors.Is(perr, io.EOF) && int64(len(buf)) < r.size-base {
		return nil, 0, false, errNeedMore
	}
	if perr != nil && !errors.Is(perr, io.EOF) {
		return nil, 0, false, perr
	}

	// A stream follows when the next keyword is "stream".
	save := lex.pos
	t := lex.next()
	// The parser closes open arrays and dictionaries at the end of its
	// input, as a truncated file needs, so a window ending inside the
	// object parses as a shorter, complete one. Only a lookahead that ends
	// inside the window — with room for the EOL after "stream" — shows
	// the object ended there.
	if lex.pos+2 > len(buf) && int64(len(buf)) < r.size-base {
		return nil, 0, false, errNeedMore
	}
	if t.kind == tokKeyword && t.str == "stream" {
		d, ok := val.(dict)
		if !ok {
			return nil, 0, false, fmt.Errorf("pdf: object %d: stream without dict", wantNum)
		}
		// Data begins after the "stream" keyword and its EOL (CRLF or LF).
		dataStart := int64(lex.pos)
		if dataStart < int64(len(buf)) && buf[dataStart] == '\r' {
			dataStart++
		}
		if dataStart < int64(len(buf)) && buf[dataStart] == '\n' {
			dataStart++
		}
		s := &stream{d: d, offset: base + dataStart, length: -1, owner: ref{wantNum, wantGen}}
		s.length = r.streamLength(d, s.offset)
		return s, 0, true, nil
	}
	lex.pos = save
	return val, lex.pos, false, nil
}

// streamLength returns the stream's byte length: the resolved /Length if
// it is present and validated by an "endstream" at the boundary, else a
// bounded forward scan for "endstream".
func (r *Reader) streamLength(d dict, dataStart int64) int64 {
	if lv, ok := d["Length"]; ok {
		n := r.asInt(lv)
		if n > 0 && dataStart+n <= r.size && r.endstreamAt(dataStart+n) {
			return n
		}
	}
	return r.scanEndstream(dataStart)
}

// endstreamAt reports whether "endstream" appears at off, allowing one
// leading EOL and a little whitespace (a 32-byte window).
func (r *Reader) endstreamAt(off int64) bool {
	w := r.slice(off, 32)
	w = bytes.TrimLeft(w, "\r\n \t")
	return bytes.HasPrefix(w, []byte("endstream"))
}

// scanEndstream finds the first EOL-preceded "endstream" after dataStart,
// bounded by the size cap, trimming one trailing EOL.
func (r *Reader) scanEndstream(dataStart int64) int64 {
	limit := min(r.size-dataStart, int64(safeio.MaxStreamBytes))
	buf := r.slice(dataStart, limit)
	idx := bytes.Index(buf, []byte("endstream"))
	if idx < 0 {
		return int64(len(buf))
	}
	end := idx
	if end > 0 && buf[end-1] == '\n' {
		end--
	}
	if end > 0 && buf[end-1] == '\r' {
		end--
	}
	return int64(end)
}

// asInt resolves a value that may be a direct integer or an indirect
// reference to one (as /Length often is).
func (r *Reader) asInt(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case ref:
		obj, err := r.object(x.num, x.gen)
		if err != nil {
			return 0
		}
		if n, ok := obj.(int64); ok {
			return n
		}
	}
	return 0
}

// decryptStrings walks a parsed value and decrypts every string leaf with
// the owning object's key.
func (r *Reader) decryptStrings(v any, num, gen int) any {
	switch x := v.(type) {
	case string:
		return string(r.dec.DecryptString(num, gen, []byte(x)))
	case dict:
		for k, e := range x {
			x[k] = r.decryptStrings(e, num, gen)
		}
		return x
	case []Value:
		for i := range x {
			x[i].data = r.decryptStrings(x[i].data, num, gen)
		}
		return x
	default:
		return v
	}
}

// streamBody returns the stream's raw section, decrypted.
func (r *Reader) streamBody(s *stream) io.Reader {
	length := s.length
	if length < 0 {
		length = r.scanEndstream(s.offset)
	}
	var body io.Reader = io.NewSectionReader(r.ra, s.offset, length)
	if r.dec != nil && !r.streamExempt(s) {
		body = r.dec.DecryptStream(s.owner.num, s.owner.gen, body)
	}
	return body
}

// streamReader returns the stream's decoded bytes: raw section →
// decryption → filter chain.
func (r *Reader) streamReader(s *stream) (io.ReadCloser, error) {
	body := r.streamBody(s)
	rd, release, _, err := r.applyFilters(body, s.d, false)
	if err != nil {
		return nil, err
	}
	return closingReader(rd, release), nil
}

// imageReader is streamReader stopping at an image codec, which is
// returned instead of applied.
func (r *Reader) imageReader(s *stream) (io.ReadCloser, ImageFilter, error) {
	body := r.streamBody(s)
	rd, release, codec, err := r.applyFilters(body, s.d, true)
	if err != nil {
		return nil, ImageFilter{}, err
	}
	return closingReader(rd, release), codec, nil
}

func closingReader(rd io.Reader, release func()) io.ReadCloser {
	if release == nil {
		return io.NopCloser(rd)
	}
	return &releasingReader{Reader: rd, release: release}
}

// releasingReader returns pooled decoder state on Close. Close is
// idempotent; reads after it fail.
type releasingReader struct {
	io.Reader
	release func()
}

func (c *releasingReader) Close() error {
	if c.release != nil {
		c.release()
		c.release = nil
	}
	return nil
}

// streamExempt reports streams that are never encrypted: cross-reference
// streams and the Identity crypt filter.
func (r *Reader) streamExempt(s *stream) bool {
	if n, ok := s.d["Type"].(name); ok && n == "XRef" {
		return true
	}
	return false
}

// NumPages returns the /Count of the document's page tree.
func (r *Reader) NumPages() int {
	return int(r.Trailer().Key("Root").Key("Pages").Key("Count").intOr(0))
}

func (v Value) intOr(d int64) int64 {
	if n, ok := v.Int64(); ok {
		return n
	}
	return d
}
