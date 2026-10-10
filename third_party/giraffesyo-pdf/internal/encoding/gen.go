//go:build ignore

// gen.go generates glyphlist.go and tables.go.
//
// Usage:
//
//	go run gen.go [path/to/glyphlist.txt]
//
// Without an argument it downloads the Adobe Glyph List (table version
// 2.0, unchanged since 2002) from the adobe-type-tools/agl-aglfn
// repository.
//
// The four simple-font encoding tables are generated from tableD2 below,
// a transcription of ISO 32000-1:2008 Table D.2 (Latin character set and
// encodings): one row per glyph name with its code in each encoding
// column, octal as printed in the spec, -1 where the spec leaves the code
// unassigned. Glyph names resolve to Unicode through the AGL, so the
// tables and the /Differences path share one source of truth.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

const aglURL = "https://raw.githubusercontent.com/adobe-type-tools/agl-aglfn/master/glyphlist.txt"

// row is one line of Table D.2: a glyph name and its code in the
// Standard, MacRoman, WinAnsi and PDFDoc encodings (-1 = unassigned).
type row struct {
	name               string
	std, mac, win, pdf int
}

// tableD2 lists every non-alphanumeric row of Table D.2. Letters A-Z,
// a-z and digits 0-9 are identity in all four encodings and are appended
// programmatically in main.
//
// Spec deviations from the Mac OS character set (deliberate, per Annex D):
// MacRomanEncoding has currency at 0333 (not the Euro of Mac OS 8.5+) and
// no Apple-logo glyph at 0360.
var tableD2 = []row{
	{"AE", 0341, 0256, 0306, 0306},
	{"Aacute", -1, 0347, 0301, 0301},
	{"Acircumflex", -1, 0345, 0302, 0302},
	{"Adieresis", -1, 0200, 0304, 0304},
	{"Agrave", -1, 0313, 0300, 0300},
	{"Aring", -1, 0201, 0305, 0305},
	{"Atilde", -1, 0314, 0303, 0303},
	{"Ccedilla", -1, 0202, 0307, 0307},
	{"Delta", -1, 0306, -1, -1},
	{"Eacute", -1, 0203, 0311, 0311},
	{"Ecircumflex", -1, 0346, 0312, 0312},
	{"Edieresis", -1, 0350, 0313, 0313},
	{"Egrave", -1, 0351, 0310, 0310},
	{"Eth", -1, -1, 0320, 0320},
	{"Euro", -1, -1, 0200, 0240},
	{"Iacute", -1, 0352, 0315, 0315},
	{"Icircumflex", -1, 0353, 0316, 0316},
	{"Idieresis", -1, 0354, 0317, 0317},
	{"Igrave", -1, 0355, 0314, 0314},
	{"Lslash", 0350, -1, -1, 0225},
	{"Ntilde", -1, 0204, 0321, 0321},
	{"OE", 0352, 0316, 0214, 0226},
	{"Oacute", -1, 0356, 0323, 0323},
	{"Ocircumflex", -1, 0357, 0324, 0324},
	{"Odieresis", -1, 0205, 0326, 0326},
	{"Ograve", -1, 0361, 0322, 0322},
	{"Omega", -1, 0275, -1, -1},
	{"Oslash", 0351, 0257, 0330, 0330},
	{"Otilde", -1, 0315, 0325, 0325},
	{"Scaron", -1, -1, 0212, 0227},
	{"Thorn", -1, -1, 0336, 0336},
	{"Uacute", -1, 0362, 0332, 0332},
	{"Ucircumflex", -1, 0363, 0333, 0333},
	{"Udieresis", -1, 0206, 0334, 0334},
	{"Ugrave", -1, 0364, 0331, 0331},
	{"Yacute", -1, -1, 0335, 0335},
	{"Ydieresis", -1, 0331, 0237, 0230},
	{"Zcaron", -1, -1, 0216, 0231},
	{"aacute", -1, 0207, 0341, 0341},
	{"acircumflex", -1, 0211, 0342, 0342},
	{"acute", 0302, 0253, 0264, 0264},
	{"adieresis", -1, 0212, 0344, 0344},
	{"ae", 0361, 0276, 0346, 0346},
	{"agrave", -1, 0210, 0340, 0340},
	{"ampersand", 0046, 0046, 0046, 0046},
	{"approxequal", -1, 0305, -1, -1},
	{"aring", -1, 0214, 0345, 0345},
	{"asciicircum", 0136, 0136, 0136, 0136},
	{"asciitilde", 0176, 0176, 0176, 0176},
	{"asterisk", 0052, 0052, 0052, 0052},
	{"at", 0100, 0100, 0100, 0100},
	{"atilde", -1, 0213, 0343, 0343},
	{"backslash", 0134, 0134, 0134, 0134},
	{"bar", 0174, 0174, 0174, 0174},
	{"braceleft", 0173, 0173, 0173, 0173},
	{"braceright", 0175, 0175, 0175, 0175},
	{"bracketleft", 0133, 0133, 0133, 0133},
	{"bracketright", 0135, 0135, 0135, 0135},
	{"breve", 0306, 0371, -1, 0030},
	{"brokenbar", -1, -1, 0246, 0246},
	{"bullet", 0267, 0245, 0225, 0200},
	{"caron", 0317, 0377, -1, 0031},
	{"ccedilla", -1, 0215, 0347, 0347},
	{"cedilla", 0313, 0374, 0270, 0270},
	{"cent", 0242, 0242, 0242, 0242},
	{"circumflex", 0303, 0366, 0210, 0032},
	{"colon", 0072, 0072, 0072, 0072},
	{"comma", 0054, 0054, 0054, 0054},
	{"copyright", -1, 0251, 0251, 0251},
	{"currency", 0250, 0333, 0244, 0244},
	{"dagger", 0262, 0240, 0206, 0201},
	{"daggerdbl", 0263, 0340, 0207, 0202},
	{"degree", -1, 0241, 0260, 0260},
	{"dieresis", 0310, 0254, 0250, 0250},
	{"divide", -1, 0326, 0367, 0367},
	{"dollar", 0044, 0044, 0044, 0044},
	{"dotaccent", 0307, 0372, -1, 0033},
	{"dotlessi", 0365, 0365, -1, 0232},
	{"eacute", -1, 0216, 0351, 0351},
	{"ecircumflex", -1, 0220, 0352, 0352},
	{"edieresis", -1, 0221, 0353, 0353},
	{"egrave", -1, 0217, 0350, 0350},
	{"eight", 0070, 0070, 0070, 0070},
	{"ellipsis", 0274, 0311, 0205, 0203},
	{"emdash", 0320, 0321, 0227, 0204},
	{"endash", 0261, 0320, 0226, 0205},
	{"equal", 0075, 0075, 0075, 0075},
	{"eth", -1, -1, 0360, 0360},
	{"exclam", 0041, 0041, 0041, 0041},
	{"exclamdown", 0241, 0301, 0241, 0241},
	{"fi", 0256, 0336, -1, 0223},
	{"five", 0065, 0065, 0065, 0065},
	{"fl", 0257, 0337, -1, 0224},
	{"florin", 0246, 0304, 0203, 0206},
	{"four", 0064, 0064, 0064, 0064},
	{"fraction", 0244, 0332, -1, 0207},
	{"germandbls", 0373, 0247, 0337, 0337},
	{"grave", 0301, 0140, 0140, 0140},
	{"greater", 0076, 0076, 0076, 0076},
	{"greaterequal", -1, 0263, -1, -1},
	{"guillemotleft", 0253, 0307, 0253, 0253},
	{"guillemotright", 0273, 0310, 0273, 0273},
	{"guilsinglleft", 0254, 0334, 0213, 0210},
	{"guilsinglright", 0255, 0335, 0233, 0211},
	{"hungarumlaut", 0315, 0375, -1, 0034},
	{"hyphen", 0055, 0055, 0055, 0055},
	{"iacute", -1, 0222, 0355, 0355},
	{"icircumflex", -1, 0224, 0356, 0356},
	{"idieresis", -1, 0225, 0357, 0357},
	{"igrave", -1, 0223, 0354, 0354},
	{"infinity", -1, 0260, -1, -1},
	{"integral", -1, 0272, -1, -1},
	{"lessequal", -1, 0262, -1, -1},
	{"less", 0074, 0074, 0074, 0074},
	{"logicalnot", -1, 0302, 0254, 0254},
	{"lozenge", -1, 0327, -1, -1},
	{"lslash", 0370, -1, -1, 0233},
	{"macron", 0305, 0370, 0257, 0257},
	{"minus", -1, -1, -1, 0212},
	{"mu", -1, 0265, 0265, 0265},
	{"multiply", -1, -1, 0327, 0327},
	{"nine", 0071, 0071, 0071, 0071},
	{"notequal", -1, 0255, -1, -1},
	{"ntilde", -1, 0226, 0361, 0361},
	{"numbersign", 0043, 0043, 0043, 0043},
	{"oacute", -1, 0227, 0363, 0363},
	{"ocircumflex", -1, 0231, 0364, 0364},
	{"odieresis", -1, 0232, 0366, 0366},
	{"oe", 0372, 0317, 0234, 0234},
	{"ogonek", 0316, 0376, -1, 0035},
	{"ograve", -1, 0230, 0362, 0362},
	{"one", 0061, 0061, 0061, 0061},
	{"onehalf", -1, -1, 0275, 0275},
	{"onequarter", -1, -1, 0274, 0274},
	{"onesuperior", -1, -1, 0271, 0271},
	{"ordfeminine", 0343, 0273, 0252, 0252},
	{"ordmasculine", 0353, 0274, 0272, 0272},
	{"oslash", 0371, 0277, 0370, 0370},
	{"otilde", -1, 0233, 0365, 0365},
	{"paragraph", 0266, 0246, 0266, 0266},
	{"parenleft", 0050, 0050, 0050, 0050},
	{"parenright", 0051, 0051, 0051, 0051},
	{"partialdiff", -1, 0266, -1, -1},
	{"percent", 0045, 0045, 0045, 0045},
	{"period", 0056, 0056, 0056, 0056},
	{"periodcentered", 0264, 0341, 0267, 0267},
	{"perthousand", 0275, 0344, 0211, 0213},
	{"pi", -1, 0271, -1, -1},
	{"plus", 0053, 0053, 0053, 0053},
	{"plusminus", -1, 0261, 0261, 0261},
	{"product", -1, 0270, -1, -1},
	{"question", 0077, 0077, 0077, 0077},
	{"questiondown", 0277, 0300, 0277, 0277},
	{"quotedbl", 0042, 0042, 0042, 0042},
	{"quotedblbase", 0271, 0343, 0204, 0214},
	{"quotedblleft", 0252, 0322, 0223, 0215},
	{"quotedblright", 0272, 0323, 0224, 0216},
	{"quoteleft", 0140, 0324, 0221, 0217},
	{"quoteright", 0047, 0325, 0222, 0220},
	{"quotesinglbase", 0270, 0342, 0202, 0221},
	{"quotesingle", 0251, 0047, 0047, 0047},
	{"radical", -1, 0303, -1, -1},
	{"registered", -1, 0250, 0256, 0256},
	{"ring", 0312, 0373, -1, 0036},
	{"scaron", -1, -1, 0232, 0235},
	{"section", 0247, 0244, 0247, 0247},
	{"semicolon", 0073, 0073, 0073, 0073},
	{"seven", 0067, 0067, 0067, 0067},
	{"six", 0066, 0066, 0066, 0066},
	{"slash", 0057, 0057, 0057, 0057},
	{"space", 0040, 0040, 0040, 0040},
	{"sterling", 0243, 0243, 0243, 0243},
	{"summation", -1, 0267, -1, -1},
	{"thorn", -1, -1, 0376, 0376},
	{"three", 0063, 0063, 0063, 0063},
	{"threequarters", -1, -1, 0276, 0276},
	{"threesuperior", -1, -1, 0263, 0263},
	{"tilde", 0304, 0367, 0230, 0037},
	{"trademark", -1, 0252, 0231, 0222},
	{"two", 0062, 0062, 0062, 0062},
	{"twosuperior", -1, -1, 0262, 0262},
	{"uacute", -1, 0234, 0372, 0372},
	{"ucircumflex", -1, 0236, 0373, 0373},
	{"udieresis", -1, 0237, 0374, 0374},
	{"ugrave", -1, 0235, 0371, 0371},
	{"underscore", 0137, 0137, 0137, 0137},
	{"yacute", -1, -1, 0375, 0375},
	{"ydieresis", -1, 0330, 0377, 0377},
	{"yen", 0245, 0264, 0245, 0245},
	{"zcaron", -1, -1, 0236, 0236},
	{"zero", 0060, 0060, 0060, 0060},
}

// extraCodes are the "also encoded as" duplicates from the notes to
// Table D.2: space and hyphen have second codes in some encodings.
var extraCodes = []struct {
	table string
	code  int
	name  string
}{
	{"win", 0240, "space"},  // note 5
	{"win", 0255, "hyphen"}, // note 5
	{"mac", 0312, "space"},  // note 4
}

func main() {
	agl, err := loadAGL()
	check(err)

	names, err := parseAGL(agl)
	check(err)

	check(writeGlyphlist(names))
	check(writeTables(names))
}

func loadAGL() ([]byte, error) {
	if len(os.Args) > 1 {
		return os.ReadFile(os.Args[1])
	}
	resp, err := http.Get(aglURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", aglURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// parseAGL reads "name;XXXX[ XXXX...]" lines into name → text.
func parseAGL(data []byte) (map[string]string, error) {
	names := make(map[string]string)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, codes, ok := strings.Cut(line, ";")
		if !ok {
			return nil, fmt.Errorf("malformed AGL line: %q", line)
		}
		var text []rune
		for field := range strings.FieldsSeq(codes) {
			var v uint32
			if _, err := fmt.Sscanf(field, "%04X", &v); err != nil {
				return nil, fmt.Errorf("bad codepoint in AGL line %q: %v", line, err)
			}
			text = append(text, rune(v))
		}
		if len(text) == 0 {
			return nil, fmt.Errorf("empty AGL mapping: %q", line)
		}
		names[name] = string(text)
	}
	if len(names) < 4000 {
		return nil, fmt.Errorf("suspiciously small AGL: %d entries", len(names))
	}
	return names, nil
}

const header = `// Code generated by gen.go; DO NOT EDIT.

`

func writeGlyphlist(names map[string]string) error {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString(`// This file embeds the Adobe Glyph List (glyphlist.txt, table version
// 2.0), https://github.com/adobe-type-tools/agl-aglfn:
//
// Copyright 2002-2019 Adobe (http://www.adobe.com/). Redistribution and
// use in source and binary forms, with or without modification, are
// permitted provided that redistributions of source code retain the
// above copyright notice and this list of conditions, and that neither
// the name of Adobe nor the names of its contributors be used to endorse
// or promote products derived from this software without specific prior
// written permission. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS
// AND CONTRIBUTORS "AS IS" WITHOUT WARRANTIES OF ANY KIND.

package encoding

// aglNames maps every Adobe Glyph List name to its text. Values may be
// multi-rune (ligature and composed-sequence entries).
var aglNames = map[string]string{
`)
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "\t%q: %q,\n", k, names[k])
	}
	b.WriteString("}\n")
	return writeFormatted("glyphlist.go", b.Bytes())
}

func writeTables(names map[string]string) error {
	glyphRune := func(name string) rune {
		s, ok := names[name]
		if !ok {
			check(fmt.Errorf("Table D.2 name %q not in AGL", name))
		}
		r := []rune(s)
		if len(r) != 1 {
			check(fmt.Errorf("Table D.2 name %q maps to %d runes", name, len(r)))
		}
		return r[0]
	}

	tables := map[string]*[256]rune{
		"std": new([256]rune), "mac": new([256]rune),
		"win": new([256]rune), "pdf": new([256]rune),
	}
	assign := func(table string, code int, name string) {
		if code < 0 {
			return
		}
		if code > 0xFF {
			check(fmt.Errorf("%s %q: code %o out of range", table, name, code))
		}
		if prev := tables[table][code]; prev != 0 {
			check(fmt.Errorf("%s code %o assigned twice (%q)", table, code, name))
		}
		tables[table][code] = glyphRune(name)
	}

	rows := tableD2
	for _, letters := range []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "abcdefghijklmnopqrstuvwxyz"} {
		for _, c := range letters {
			rows = append(rows, row{string(c), int(c), int(c), int(c), int(c)})
		}
	}

	for _, r := range rows {
		assign("std", r.std, r.name)
		assign("mac", r.mac, r.name)
		assign("win", r.win, r.name)
		assign("pdf", r.pdf, r.name)
	}
	for _, e := range extraCodes {
		if tables[e.table][e.code] != 0 {
			check(fmt.Errorf("extra %s code %o already assigned", e.table, e.code))
		}
		tables[e.table][e.code] = glyphRune(e.name)
	}

	// AGL unifies the glyph name "Omega" with U+2126 OHM SIGN; Apple's
	// mapping table (ROMAN.TXT) and Unicode NFC both prefer the
	// canonically equivalent U+03A9 GREEK CAPITAL LETTER OMEGA.
	tables["mac"][0275] = 0x03A9

	ascii := new([256]rune)
	for c := 0x20; c <= 0x7E; c++ {
		ascii[c] = rune(c)
	}

	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString(`// The simple-font encoding tables of ISO 32000-1:2008 Annex D,
// Table D.2, generated from glyph names resolved through the Adobe Glyph
// List. Slot value 0 means the code is unassigned in that encoding.

package encoding

`)
	emit := func(goName, comment string, t *[256]rune) {
		fmt.Fprintf(&b, "// %s\nvar %s = [256]rune{\n", comment, goName)
		for base := 0; base < 256; base += 8 {
			b.WriteString("\t")
			for i := base; i < base+8; i++ {
				fmt.Fprintf(&b, "0x%04x, ", t[i])
			}
			b.WriteString("\n")
		}
		b.WriteString("}\n\n")
	}
	emit("standardTable", "standardTable is Adobe StandardEncoding.", tables["std"])
	emit("macRomanTable", "macRomanTable is MacRomanEncoding (Annex D form: currency at 0333, no Apple-logo glyph).", tables["mac"])
	emit("winAnsiTable", "winAnsiTable is WinAnsiEncoding (CP1252; space also at 0240, hyphen also at 0255 per note 5).", tables["win"])
	emit("pdfDocTable", "pdfDocTable is PDFDocEncoding (text strings, not font glyphs).", tables["pdf"])
	emit("asciiTable", "asciiTable is printable-ASCII passthrough for symbolic and unrecognized encodings.", ascii)
	return writeFormatted("tables.go", b.Bytes())
}

func writeFormatted(path string, src []byte) error {
	out, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	return os.WriteFile(path, out, 0o644)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}
