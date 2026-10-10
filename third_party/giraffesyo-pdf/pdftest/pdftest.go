// Package pdftest builds minimal synthetic PDF files for tests, in the
// spirit of net/http/httptest. The helpers reproduce structural patterns
// seen in real-world PDFs — Form XObjects, composite fonts, ToUnicode
// CMaps, segmented content streams (this file); Flate streams, PDF 1.5
// cross-reference streams, object streams, and hybrid-reference files
// (xref.go); and standard-handler-encrypted files (crypt.go) — from
// entirely synthetic content.
package pdftest

import (
	"fmt"
	"strings"
)

// Build assembles a PDF from numbered objects: objs[i] becomes object i+1.
// rootID is the /Root (catalog) object number.
func Build(rootID int, objs ...string) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, obj := range objs {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objs)+1, rootID, xref)
	return []byte(b.String())
}

// Stream renders a stream object; /Length is computed from content.
func Stream(dict, content string) string {
	return fmt.Sprintf("<< %s /Length %d >>\nstream\n%s\nendstream", dict, len(content), content)
}

// Catalog renders the document catalog pointing at the page tree root.
func Catalog(pagesID int) string {
	return fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID)
}

// Pages renders the page tree root listing the given page objects.
func Pages(kidIDs ...int) string {
	var kids []string
	for _, id := range kidIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}
	return fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(kidIDs))
}

// Page renders a single page with the given resources and content stream.
func Page(parentID, contentsID int, resources string) string {
	return fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 612 792] /Resources %s /Contents %d 0 R >>",
		parentID, resources, contentsID)
}

// PageContentsArray renders a page whose /Contents is an array of streams
// forming one logical content stream (tokens may straddle boundaries).
func PageContentsArray(parentID int, resources string, contentsIDs ...int) string {
	var refs []string
	for _, id := range contentsIDs {
		refs = append(refs, fmt.Sprintf("%d 0 R", id))
	}
	return fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 612 792] /Resources %s /Contents [%s] >>",
		parentID, resources, strings.Join(refs, " "))
}

// Helvetica renders a standard-14 simple font object (WinAnsi).
func Helvetica() string {
	return "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"
}

// ToUnicodeCMap renders a ToUnicode CMap stream object. Entries go inside
// the bfchar/bfrange sections, e.g. "<0001> <0048>".
func ToUnicodeCMap(body string) string {
	content := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
` + body + `
endcmap
CMapName currentdict /CMap defineresource pop
end
end`
	return Stream("", content)
}

// Type0Font renders a composite Identity-H font referencing a descendant
// CIDFont and a ToUnicode stream.
func Type0Font(descendantID, toUnicodeID int) string {
	return fmt.Sprintf("<< /Type /Font /Subtype /Type0 /BaseFont /FAKEFT+Synthetic /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>",
		descendantID, toUnicodeID)
}

// CIDFont renders the descendant font with a /W widths array.
func CIDFont(w string) string {
	return fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /FAKEFT+Synthetic /DW 1000 %s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> >>", w)
}

// Hex2 renders 2-byte character codes as a PDF hex string, e.g. Hex2(1, 2)
// → "<00010002>".
func Hex2(codes ...int) string {
	var b strings.Builder
	b.WriteByte('<')
	for _, c := range codes {
		fmt.Fprintf(&b, "%04X", c)
	}
	b.WriteByte('>')
	return b.String()
}
