package pdf

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A PDF paints right-to-left scripts in visual order: an Arabic or Hebrew
// word's glyphs run left to right across the page, last letter first.
// Layout orders a line's glyphs by position, so a line holding such text
// is converted back to logical (reading) order before it is written.
//
// The conversion applies the Unicode Bidirectional Algorithm's reordering
// rule (UAX #9, L2) to the line's pieces — glyph texts and the spaces
// between them — at embedding levels resolved from their directions: a
// right-to-left piece is level 1 in a left-to-right line, a left-to-right
// piece level 2 in a right-to-left one, and numbers are left-to-right
// within right-to-left text. Reversing every run at each level, from the
// highest down, is its own inverse, so applied to visual order it yields
// logical order. Explicit embeddings and isolates have no representation
// in painted glyphs, so the levels are resolved from the text alone.

// bidiClass is the part of a piece's bidirectional type that reordering
// needs.
type bidiClass uint8

const (
	bidiNeutral bidiClass = iota
	bidiLeft
	bidiRight
	bidiNumber    // European or Arabic digits
	bidiSeparator // a common or European separator, which joins digits
)

// isRightToLeft reports whether r is a strong right-to-left character:
// Hebrew, Arabic, Syriac, Thaana, NKo, and related blocks and their
// presentation forms, excluding the digits and marks among them.
func isRightToLeft(r rune) bool {
	switch {
	case r < 0x0590:
		return false
	case r >= 0x0660 && r <= 0x0669, r >= 0x06F0 && r <= 0x06F9:
		return false // Arabic-Indic digits: numbers, not letters
	case r <= 0x08FF:
		return !unicode.Is(unicode.Mn, r)
	case r >= 0xFB1D && r <= 0xFDFF, r >= 0xFE70 && r <= 0xFEFF:
		return !unicode.Is(unicode.Mn, r)
	case r >= 0x10800 && r <= 0x10FFF, r >= 0x1E800 && r <= 0x1EFFF:
		return !unicode.Is(unicode.Mn, r)
	}
	return false
}

// lineHasRightToLeft reports whether any glyph on the line is strongly
// right-to-left. Most text is below U+0590, so the scan is cheap.
func lineHasRightToLeft(line layoutLine) bool {
	for _, g := range line.glyphs {
		for _, r := range g.Text {
			if r >= 0x0590 && isRightToLeft(r) {
				return true
			}
		}
	}
	return false
}

func classifyBidi(text string) bidiClass {
	digits := false
	for _, r := range text {
		switch {
		case isRightToLeft(r):
			return bidiRight
		case unicode.IsDigit(r):
			digits = true
		case unicode.IsLetter(r):
			return bidiLeft
		}
	}
	if digits {
		return bidiNumber
	}
	if r, size := utf8.DecodeRuneInString(text); size == len(text) && strings.ContainsRune(",.:/+-", r) {
		return bidiSeparator
	}
	return bidiNeutral
}

// writeBidiLine writes a line holding right-to-left text in logical order.
func writeBidiLine(b *strings.Builder, line layoutLine) {
	var pieces []string
	walkLayoutLine(line, func(text string) { pieces = append(pieces, text) })
	for _, piece := range logicalOrder(pieces) {
		b.WriteString(piece)
	}
}

// logicalOrder reorders pieces given in visual order into logical order.
func logicalOrder(pieces []string) []string {
	classes := make([]bidiClass, len(pieces))
	right, left := 0, 0
	for i, p := range pieces {
		classes[i] = classifyBidi(p)
		switch classes[i] {
		case bidiRight:
			right++
		case bidiLeft:
			left++
		}
	}
	// A lone separator between digits joins the number (UAX #9, W4).
	for i := 1; i+1 < len(classes); i++ {
		if classes[i] == bidiSeparator && classes[i-1] == bidiNumber && classes[i+1] == bidiNumber {
			classes[i] = bidiNumber
		}
	}
	rtlBase := right > left
	base := uint8(0)
	if rtlBase {
		base = 1
	}

	// strongAt gives the direction a piece lends to the neutrals beside it:
	// numbers count as right-to-left (UAX #9, N1), except that in a
	// left-to-right line a number led by left-to-right text is left-to-right
	// (W7).
	lastStrong := make([]bidiClass, len(pieces)) // nearest strong class before i, in visual order
	prev := bidiLeft                             // the line's start takes the base direction (sos)
	if rtlBase {
		prev = bidiRight
	}
	for i, c := range classes {
		lastStrong[i] = prev
		if c == bidiLeft || c == bidiRight {
			prev = c
		}
	}
	direction := func(i int) bidiClass {
		switch classes[i] {
		case bidiLeft, bidiRight:
			return classes[i]
		case bidiNumber:
			if !rtlBase && lastStrong[i] == bidiLeft {
				return bidiLeft
			}
			return bidiRight
		}
		return bidiNeutral
	}

	levels := make([]uint8, len(pieces))
	for i := range pieces {
		switch classes[i] {
		case bidiRight:
			levels[i] = 1
		case bidiLeft:
			levels[i] = 2 * base // 0 in a left-to-right line, 2 in a right-to-left one
		case bidiNumber:
			if direction(i) == bidiLeft {
				levels[i] = 0
			} else {
				levels[i] = 2
			}
		default:
			// A neutral takes the direction of the text on both sides when
			// they agree, and the line's otherwise (N1, N2).
			before, after := bidiNeutral, bidiNeutral
			for j := i - 1; j >= 0 && before == bidiNeutral; j-- {
				before = direction(j)
			}
			for j := i + 1; j < len(pieces) && after == bidiNeutral; j++ {
				after = direction(j)
			}
			switch {
			case before == bidiRight && after == bidiRight:
				levels[i] = 1
			case before == bidiLeft && after == bidiLeft:
				levels[i] = 2 * base
			default:
				levels[i] = base
			}
		}
	}

	out := slices.Clone(pieces)
	highest := slices.Max(levels)
	for level := highest; level >= 1; level-- {
		for i := 0; i < len(out); {
			if levels[i] < level {
				i++
				continue
			}
			j := i
			for j < len(out) && levels[j] >= level {
				j++
			}
			slices.Reverse(out[i:j])
			slices.Reverse(levels[i:j])
			i = j
		}
	}
	return out
}
