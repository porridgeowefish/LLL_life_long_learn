package teacherservice

import "unicode/utf8"

func truncateUTF8(text string, bytes int) string {
	if bytes <= 0 {
		return ""
	}
	if len(text) <= bytes {
		return text
	}
	for bytes > 0 && !utf8.RuneStart(text[bytes]) {
		bytes--
	}
	return text[:bytes]
}
