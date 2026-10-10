package pdf

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/giraffesyo/pdf/internal/encoding"
)

// decodeTextString decodes a PDF text string in UTF-16BE, UTF-16LE,
// UTF-8-with-BOM, or PDFDocEncoding.
func decodeTextString(raw []byte) string {
	switch {
	case len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff:
		return decodeUTF16(raw[2:], false)
	case len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe:
		return decodeUTF16(raw[2:], true)
	case len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf:
		if utf8.Valid(raw[3:]) {
			return string(raw[3:])
		}
	}
	enc := encoding.New("PDFDocEncoding", nil)
	var b strings.Builder
	for _, code := range raw {
		b.WriteString(enc.Decode(code))
	}
	return b.String()
}

func decodeUTF16(raw []byte, little bool) string {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		if little {
			units = append(units, uint16(raw[i+1])<<8|uint16(raw[i]))
		} else {
			units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
		}
	}
	return string(utf16.Decode(units))
}
