package workspace

import (
	"regexp"
	"strings"
	"unicode"
)

// collapseDashes collapses 2+ consecutive dashes into one.
var collapseDashes = regexp.MustCompile(`-{2,}`)

// Slugify converts a free-form title into a filesystem-safe kebab-case slug.
// Allows Unicode letters and digits (so CJK / accented Latin pass through),
// replaces separators (space/_/slash/dot) with a single dash, drops everything else.
// Returns an empty string if the input has no usable characters.
func Slugify(title string) string {
	if title == "" {
		return ""
	}
	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevDash := false
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		default:
			// Treat any other punctuation/whitespace as separator.
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	s = collapseDashes.ReplaceAllString(s, "-")
	return s
}
