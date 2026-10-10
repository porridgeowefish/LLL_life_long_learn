// Package encoding implements the simple-font character encodings of
// ISO 32000-1 Annex D and the Adobe Glyph List glyph-name algorithm.
//
// A simple font addresses glyphs with single-byte codes; an Encoding maps
// those codes to text. Codes with no mapping decode to "" so callers drop
// them instead of emitting garbage. The package knows nothing about the
// PDF object model: callers traverse /Encoding dictionaries and pass
// plain values.
package encoding

//go:generate go run gen.go

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// An Encoding maps single-byte character codes to text: /Differences
// overrides first, then the base table.
type Encoding struct {
	table *[256]rune
	diff  map[byte]string
}

// New returns the encoding named by base — StandardEncoding,
// WinAnsiEncoding, MacRomanEncoding or PDFDocEncoding — with diffs (a
// /Differences array with glyph names already resolved to text, e.g. via
// GlyphToText) layered on top. Any other base name yields printable-ASCII
// passthrough: high bytes unmapped, which is the honest fallback for
// symbolic fonts whose encoding lives in the font program.
func New(base string, diffs map[byte]string) *Encoding {
	t := &asciiTable
	switch base {
	case "StandardEncoding":
		t = &standardTable
	case "WinAnsiEncoding":
		t = &winAnsiTable
	case "MacRomanEncoding":
		t = &macRomanTable
	case "PDFDocEncoding":
		t = &pdfDocTable
	case "SymbolEncoding":
		table := symbolEncodingTable()
		t = &table
	case "ZapfDingbatsEncoding":
		table := zapfDingbatsTable()
		t = &table
	}
	return &Encoding{table: t, diff: diffs}
}

// Decode maps one character code to text; "" means the code has no
// mapping. A code remapped by /Differences to an unresolvable glyph name
// decodes to "" (the diff entry is present but empty), not to the base
// table's glyph — the file explicitly said the slot means something else.
func (e *Encoding) Decode(code byte) string {
	if s, ok := e.diff[code]; ok {
		return s
	}
	if r := e.table[code]; r != 0 {
		return string(r)
	}
	return ""
}

// GlyphToText resolves a glyph name (from a /Differences array) to text
// per Adobe's "Unicode values from glyph names" algorithm: strip any
// period suffix (a.sc → a), split ligature components on "_" (f_i → fi),
// then resolve each component through the Adobe Glyph List, the uni+hex4*
// (UTF-16BE) form, or the u+hex4..6 (scalar) form. Names with an
// unresolvable component return "".
func GlyphToText(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		name = name[:i]
	}
	if name == "" {
		return ""
	}
	if !strings.ContainsRune(name, '_') {
		return componentText(name)
	}
	var b strings.Builder
	for part := range strings.SplitSeq(name, "_") {
		s := componentText(part)
		if s == "" {
			return ""
		}
		b.WriteString(s)
	}
	return b.String()
}

func componentText(name string) string {
	if s, ok := aglNames[name]; ok {
		return s
	}
	if hex, ok := strings.CutPrefix(name, "uni"); ok && len(hex) >= 4 && len(hex)%4 == 0 {
		units := make([]uint16, 0, len(hex)/4)
		for i := 0; i+4 <= len(hex); i += 4 {
			v, ok := hexUint(hex[i : i+4])
			if !ok {
				return ""
			}
			units = append(units, uint16(v&0xFFFF)) // 4 hex digits
		}
		s := string(utf16.Decode(units))
		if strings.ContainsRune(s, utf8.RuneError) {
			return "" // lone surrogate: unresolvable, not U+FFFD soup
		}
		return s
	}
	if hex, ok := strings.CutPrefix(name, "u"); ok && len(hex) >= 4 && len(hex) <= 6 {
		v, ok := hexUint(hex)
		r := rune(v & 0xFFFFFF) // ≤ 6 hex digits
		if !ok || !utf8.ValidRune(r) {
			return ""
		}
		return string(r)
	}
	return ""
}

// hexUint parses uppercase hex, the only form the AGL algorithm accepts.
func hexUint(s string) (uint32, bool) {
	var v uint32
	for i := range len(s) {
		var d uint32
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			d = uint32(c - '0')
		case c >= 'A' && c <= 'F':
			d = uint32(c-'A') + 10
		default:
			return 0, false
		}
		v = v<<4 | d
	}
	return v, true
}
