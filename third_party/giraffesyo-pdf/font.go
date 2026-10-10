package pdf

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/giraffesyo/pdf/internal/encoding"
	"github.com/giraffesyo/pdf/internal/object"
)

// fontInfo wraps one font dictionary with decode and width lookups.
type fontInfo struct {
	composite     bool
	toUni         *cmapData
	cidUni        *cmapData
	encoding      *cmapData
	fallback      *encoding.Encoding // simple-font encoding (WinAnsi/MacRoman/Differences/...)
	fallbackFirst bool               // explicit/standard PDF encoding precedes font-program hints
	differences   map[byte]string    // explicit /Differences always override font-program hints
	warnings      []error

	// The embedded font program's code→text hints are consulted only when
	// ToUnicode and the encoding do not map a code, so the program —
	// often the largest stream a font owns — is inflated and parsed on
	// first need rather than when the font is loaded. loadEmbedded is nil
	// once that has happened (or when the font has no program).
	// The embedded font program's code→text hints are consulted only when
	// ToUnicode and the encoding do not map a code, so the program —
	// often the largest stream a font owns — is inflated and parsed on
	// first need rather than when the font is loaded.
	embeddedOnce sync.Once
	loadEmbedded func() (map[uint32]string, error)
	embedded     map[uint32]string
	embeddedErr  error

	// Simple fonts have at most 256 codes; each code's text is resolved
	// once and then served from here instead of through the ToUnicode,
	// Differences, and encoding lookups on every glyph. The memo is
	// atomic rather than locked: pages extracted concurrently share the
	// font, and resolution is pure, so a code resolved twice stores the
	// same text.
	simpleText [256]atomic.Pointer[string]

	firstChar int
	widths    []float64          // simple fonts: indexed by code-firstChar
	cidWidths map[uint32]float64 // composite fonts
	defWidth  float64

	vertical    bool
	cidVertical map[uint32]verticalMetric
	defVertical verticalMetric
}

type verticalMetric struct {
	w1 float64
	vx float64
	vy float64
}

// fontCache is a document's fonts, keyed by the object number of their
// dictionary. Pages extracted concurrently share it, so it carries a
// mutex; a fontInfo itself is safe to use from several pages at once.
type fontCache struct {
	mu sync.Mutex
	m  map[int]*fontInfo
}

func newFontCache() *fontCache { return &fontCache{m: map[int]*fontInfo{}} }

func (c *fontCache) get(num int) (*fontInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.m[num]
	return f, ok
}

func (c *fontCache) put(num int, f *fontInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[num] = f
}

// loadFont returns the font resource called name. Lookups go through two
// caches: names is scoped to one content stream (a resource name means
// one thing within a stream), and shared spans the whole document, keyed
// by the font dictionary's object number. The document cache is what
// keeps a font that every page references — the common case — from being
// decoded once per page: building a fontInfo can mean inflating and
// parsing an embedded font program and its CMaps, which dominated
// extraction time on multi-page documents. Fonts given as direct
// dictionaries have no object number and are cached per stream only.
func loadFont(
	names map[string]*fontInfo,
	shared *fontCache,
	resources object.Value,
	name string,
	resolver CMapResolver,
	streamLimit int,
) *fontInfo {
	if f, ok := names[name]; ok {
		return f
	}
	fv := resources.Key("Font").Key(name)
	num, indirect := fv.ObjectNumber()
	if indirect && shared != nil {
		if f, ok := shared.get(num); ok {
			names[name] = f
			return f
		}
	}
	var f *fontInfo
	if fv.Kind() == object.Dict {
		f = newFontInfo(fv, resolver, streamLimit)
	}
	names[name] = f
	if indirect && shared != nil {
		shared.put(num, f)
	}
	return f
}

func newFontInfo(fv object.Value, resolver CMapResolver, streamLimit int) *fontInfo {
	f := &fontInfo{defWidth: 500, defVertical: verticalMetric{w1: -1000, vy: 880}}
	var err error
	f.toUni, err = parseToUnicode(fv.Key("ToUnicode"), resolver, streamLimit)
	if err != nil {
		f.warnings = append(f.warnings, err)
	}

	if fv.Key("Subtype").Name() == "Type0" {
		f.composite = true
		f.defWidth = 1000
		f.encoding, err = parseEncodingCMap(fv.Key("Encoding"), resolver, streamLimit)
		if err != nil {
			f.warnings = append(f.warnings, err)
		}
		if f.encoding == nil {
			if f.toUni != nil && len(f.toUni.spaces) > 0 {
				f.encoding = &cmapData{
					spaces:   append([]codeSpace(nil), f.toUni.spaces...),
					identity: true,
				}
			} else {
				f.encoding = identityCMap(false)
			}
		}
		f.vertical = f.encoding.wmode == 1
		desc := fv.Key("DescendantFonts").Index(0)
		if desc.Kind() == object.Dict {
			info := desc.Key("CIDSystemInfo")
			f.cidUni = adobeCIDUnicode(info.Key("Registry").RawString(), info.Key("Ordering").RawString())
			if dw, ok := desc.Key("DW").Float64(); ok {
				f.defWidth = dw
			}
			f.cidWidths = parseCIDWidths(desc.Key("W"))
			f.defVertical = parseDefaultVertical(desc.Key("DW2"))
			f.cidVertical = parseCIDVertical(desc.Key("W2"))
			f.loadEmbedded = func() (map[uint32]string, error) {
				return embeddedCompositeFallback(desc, streamLimit)
			}
		}
		return f
	}

	// Simple font: /FirstChar + /Widths, standard encoding as fallback.
	// Exception: a Type3 font without ToUnicode draws glyph procedures
	// addressed by arbitrary codes; no encoding fallback can recover text,
	// and passing raw codes through produces convincing-looking garbage.
	// Dropping the glyphs lets truly image-like documents fail honestly.
	if fv.Key("Subtype").Name() == "Type3" && f.toUni == nil {
		return f
	}
	f.differences = parseDifferences(fv.Key("Encoding").Key("Differences"))
	f.fallback, f.fallbackFirst = fallbackEncoding(fv, f.differences)
	f.loadEmbedded = func() (map[uint32]string, error) {
		return embeddedSimpleFallback(fv, streamLimit)
	}
	f.firstChar = int(intOr(fv.Key("FirstChar"), 0))
	if wArr := fv.Key("Widths"); wArr.Kind() == object.Array {
		f.widths = make([]float64, wArr.Len())
		for i := range f.widths {
			f.widths[i], _ = wArr.Index(i).Float64()
		}
	}
	if mw, ok := fv.Key("FontDescriptor").Key("MissingWidth").Float64(); ok {
		f.defWidth = mw
	}
	return f
}

// intOr returns the value's integer, or d for non-integers.
func intOr(v object.Value, d int64) int64 {
	if n, ok := v.Int64(); ok {
		return n
	}
	return d
}

// fallbackEncoding builds a simple font's code→text fallback from its
// /Encoding entry per ISO 32000-1 §9.6.5: a base encoding name, or a
// dictionary carrying /BaseEncoding and /Differences, or — when absent —
// StandardEncoding for nonsymbolic fonts. Symbolic fonts (descriptor flag
// 3 set, flag 6 clear) keep their built-in encoding, which lives inside
// the font program; without parsing it only printable ASCII passes
// through, and high bytes drop honestly.
func fallbackEncoding(fv object.Value, differences map[byte]string) (*encoding.Encoding, bool) {
	builtIn := builtInFontEncoding(fv.Key("BaseFont").Name())
	enc := fv.Key("Encoding")
	if enc.Kind() == object.Dict {
		base := enc.Key("BaseEncoding").Name()
		switch base {
		case "WinAnsiEncoding", "MacRomanEncoding", "SymbolEncoding", "ZapfDingbatsEncoding":
		default:
			base = builtIn
			if base == "" {
				if symbolicFont(fv) {
					return encoding.New("", differences), false
				}
				base = "StandardEncoding"
			}
		}
		return encoding.New(base, differences), true
	}
	switch name := enc.Name(); name {
	case "WinAnsiEncoding", "MacRomanEncoding", "SymbolEncoding", "ZapfDingbatsEncoding":
		return encoding.New(name, nil), true
	case "":
		if builtIn != "" {
			return encoding.New(builtIn, nil), true
		}
		if symbolicFont(fv) {
			return encoding.New("", nil), false // font-program hint first
		}
		return encoding.New("StandardEncoding", nil), true
	default:
		return encoding.New("", nil), false // unrecognized named encoding
	}
}

func builtInFontEncoding(name string) string {
	if plus := strings.IndexByte(name, '+'); plus >= 0 {
		name = name[plus+1:]
	}
	switch name {
	case "Symbol":
		return "SymbolEncoding"
	case "ZapfDingbats":
		return "ZapfDingbatsEncoding"
	default:
		return ""
	}
}

// parseDifferences reads a /Differences array — integers set the current
// code, names assign consecutive codes — resolving glyph names to text.
func parseDifferences(arr object.Value) map[byte]string {
	if arr.Kind() != object.Array {
		return nil
	}
	var m map[byte]string
	code := -1
	for i := range arr.Len() {
		el := arr.Index(i)
		switch el.Kind() {
		case object.Integer:
			n, _ := el.Int64()
			code = int(n)
		case object.Name:
			if code >= 0 && code <= 0xFF {
				if m == nil {
					m = map[byte]string{}
				}
				m[byte(code)] = encoding.GlyphToText(el.Name())
			}
			code++
		}
	}
	return m
}

func symbolicFont(fv object.Value) bool {
	flags := intOr(fv.Key("FontDescriptor").Key("Flags"), 0)
	return flags&4 != 0 && flags&32 == 0
}

// parseCIDWidths reads a CIDFont /W array: sequences of either
// "c [w1 w2 ...]" or "cFirst cLast w".
func parseCIDWidths(w object.Value) map[uint32]float64 {
	if w.Kind() != object.Array {
		return nil
	}
	out := map[uint32]float64{}
	for i := 0; i < w.Len(); {
		c := w.Index(i)
		if i+1 >= w.Len() {
			break
		}
		next := w.Index(i + 1)
		switch next.Kind() {
		case object.Array:
			start := clampCID(intOr(c, 0))
			for j := 0; j < next.Len() && j < 65536; j++ {
				out[start+uint32(j)], _ = next.Index(j).Float64()
			}
			i += 2
		case object.Integer, object.Real:
			if i+2 >= w.Len() {
				return out
			}
			lo, hi := clampCID(intOr(c, 0)), clampCID(intOr(next, 0))
			width, _ := w.Index(i + 2).Float64()
			if hi-lo < 65536 {
				for code := lo; code <= hi; code++ {
					out[code] = width
				}
			}
			i += 3
		default:
			return out
		}
	}
	return out
}

// clampCID bounds a CID read from a (possibly hostile) file to the valid
// two-byte range.
func clampCID(v int64) uint32 {
	if v < 0 {
		return 0
	}
	if v > 0xFFFF {
		return 0xFFFF
	}
	return uint32(v)
}

type decoded struct {
	text     string
	width    float64 // glyph-space units (1/1000 em)
	space    bool    // single-byte code 32: word spacing applies
	vertical bool
	vm       verticalMetric
}

// appendDecoded splits raw string bytes into per-code decoded glyphs and
// appends them to dst.
func (f *fontInfo) appendDecoded(dst []decoded, raw []byte) []decoded {
	if f == nil {
		return dst
	}
	n := len(raw)
	dst = slices.Grow(dst, n)

	if f.composite {
		for len(raw) > 0 {
			key, consumed := f.encoding.nextCode(raw)
			if consumed == 0 {
				break
			}
			cid := f.encoding.cid(key)
			width := f.cidWidth(cid)
			vm := f.verticalMetric(cid, width)
			dst = append(dst, decoded{
				text:     f.mapComposite(key, cid),
				width:    width,
				space:    key.bytes == 1 && key.value == 32,
				vertical: f.vertical,
				vm:       vm,
			})
			raw = raw[consumed:]
		}
		return dst
	}
	for i := range raw {
		code := uint32(raw[i])
		dst = append(dst, decoded{
			text:  f.mapSimple(code),
			width: f.simpleWidth(int(code)),
			space: raw[i] == ' ',
		})
	}
	return dst
}

// mapSimple decodes a simple-font code using ToUnicode first, then the font
// encoding fallback.
func (f *fontInfo) mapSimple(code uint32) string {
	if code > 0xff {
		return f.resolveSimple(code)
	}
	if s := f.simpleText[code].Load(); s != nil {
		return *s
	}
	s := f.resolveSimple(code)
	f.simpleText[code].Store(&s)
	return s
}

func (f *fontInfo) resolveSimple(code uint32) string {
	if f.toUni != nil {
		if s, ok := f.toUni.unicode[codeKey{value: code, bytes: 1}]; ok {
			return s
		}
	}
	if code <= 0xff {
		if s, explicitlyMapped := f.differences[byte(code)]; explicitlyMapped {
			return s
		}
	}
	if f.fallbackFirst && f.fallback != nil && code <= 0xff {
		if s := f.fallback.Decode(byte(code)); s != "" {
			return s
		}
	}
	if s := f.embeddedText(code); sanitizeText(s, false) != "" {
		return s
	}
	if f.fallback != nil && code <= 0xFF {
		return f.fallback.Decode(byte(code))
	}
	return "�"
}

func (f *fontInfo) mapComposite(key codeKey, cid uint32) string {
	if f.toUni != nil {
		if s, ok := f.toUni.unicode[key]; ok {
			return s
		}
	}
	if f.cidUni != nil {
		if s, ok := f.cidUni.unicode[codeKey{value: cid, bytes: 2}]; ok {
			return s
		}
	}
	if s := f.embeddedText(cid); s != "" {
		return s
	}
	return "�"
}

// embeddedText returns the font program's hint for code, parsing the
// program on first use. A parse error is kept in embeddedErr for the
// walker to report once per content stream.
func (f *fontInfo) embeddedText(code uint32) string {
	f.embeddedOnce.Do(func() {
		if f.loadEmbedded != nil {
			f.embedded, f.embeddedErr = f.loadEmbedded()
		}
	})
	return f.embedded[code]
}

// embeddedFailure returns the font program's parse error once it has been
// attempted, for the walker to report at the point of use.
func (f *fontInfo) embeddedFailure() error { return f.embeddedErr }

func (f *fontInfo) verticalMetric(cid uint32, width float64) verticalMetric {
	vm, ok := f.cidVertical[cid]
	if !ok {
		vm = f.defVertical
	}
	if vm.vx == 0 {
		vm.vx = width / 2
	}
	return vm
}

func parseDefaultVertical(v object.Value) verticalMetric {
	vm := verticalMetric{w1: -1000, vy: 880}
	if v.Kind() == object.Array {
		if v.Len() > 0 {
			vm.vy, _ = v.Index(0).Float64()
		}
		if v.Len() > 1 {
			vm.w1, _ = v.Index(1).Float64()
		}
	}
	return vm
}

// parseCIDVertical reads a CIDFont /W2 array. Array runs contain triples
// (w1y, v1x, v1y); range runs repeat one triple.
func parseCIDVertical(v object.Value) map[uint32]verticalMetric {
	if v.Kind() != object.Array {
		return nil
	}
	out := map[uint32]verticalMetric{}
	for i := 0; i < v.Len(); {
		start := clampCID(intOr(v.Index(i), 0))
		if i+1 >= v.Len() {
			break
		}
		next := v.Index(i + 1)
		if next.Kind() == object.Array {
			for j, cid := 0, start; j+2 < next.Len() && cid <= 0xffff; j, cid = j+3, cid+1 {
				w1, _ := next.Index(j).Float64()
				vx, _ := next.Index(j + 1).Float64()
				vy, _ := next.Index(j + 2).Float64()
				out[cid] = verticalMetric{w1: w1, vx: vx, vy: vy}
			}
			i += 2
			continue
		}
		if i+4 >= v.Len() {
			break
		}
		end := clampCID(intOr(next, 0))
		w1, _ := v.Index(i + 2).Float64()
		vx, _ := v.Index(i + 3).Float64()
		vy, _ := v.Index(i + 4).Float64()
		if end >= start && end-start < 65536 {
			for cid := start; cid <= end; cid++ {
				out[cid] = verticalMetric{w1: w1, vx: vx, vy: vy}
			}
		}
		i += 5
	}
	return out
}

func (f *fontInfo) simpleWidth(code int) float64 {
	idx := code - f.firstChar
	if idx >= 0 && idx < len(f.widths) && f.widths[idx] > 0 {
		return f.widths[idx]
	}
	return f.defWidth
}

func (f *fontInfo) cidWidth(code uint32) float64 {
	if w, ok := f.cidWidths[code]; ok {
		return w
	}
	return f.defWidth
}
