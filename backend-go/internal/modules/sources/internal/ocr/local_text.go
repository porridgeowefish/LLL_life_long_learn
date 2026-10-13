package ocr

import (
	"strings"
	"unicode"
)

// Windows returns spaces between individual CJK glyph words. Remove only
// those artificial spaces, keeping paragraph breaks and Latin word spacing.
func normalizeLocalText(text string) string {
	chars := []rune(text)
	cjk := func(r rune) bool {
		return unicode.Is(unicode.Han, r) || strings.ContainsRune("，。！？；：、（）《》“”‘’", r)
	}
	var out strings.Builder
	for i := 0; i < len(chars); i++ {
		if chars[i] == ' ' || chars[i] == '\t' {
			end := i
			for end < len(chars) && (chars[end] == ' ' || chars[end] == '\t') {
				end++
			}
			if i > 0 && end < len(chars) && cjk(chars[i-1]) && cjk(chars[end]) {
				i = end - 1
				continue
			}
		}
		out.WriteRune(chars[i])
	}
	return strings.TrimSpace(out.String())
}
