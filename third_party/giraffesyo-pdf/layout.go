package pdf

import (
	"math"
	"slices"
	"strings"
	"unicode"
)

// layoutGlyph refers to a page glyph by pointer rather than copying it: a
// Glyph is 160 bytes, and copying every one into per-line slices doubled
// the allocation volume of text reconstruction. The pointer targets the
// caller's slice, which is not resized while layout runs.
type layoutGlyph struct {
	*Glyph
	index int
}

type layoutLine struct {
	glyphs []layoutGlyph
	dir    Point
	normal Point
	offset float64
	first  int
}

func reconstructPositionText(page Page, layout LayoutOptions) string {
	if len(page.Glyphs) == 0 {
		return ""
	}
	lines := buildLayoutLines(page.Glyphs, layout.KeepDuplicateGlyphs)
	if layout.Mode == LayoutColumns {
		lines = orderColumns(lines, page.CropBox)
	} else {
		slices.SortStableFunc(lines, compareLayoutLines)
	}

	var b strings.Builder
	for _, line := range lines {
		if !lineHasText(line) {
			continue // space glyphs alone: nothing a reader would see
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		writeLayoutLine(&b, line)
	}
	return b.String()
}

func lineHasText(line layoutLine) bool {
	for _, g := range line.glyphs {
		if strings.TrimSpace(g.Text) != "" {
			return true
		}
	}
	return false
}

func buildLayoutLines(glyphs []Glyph, keepDuplicates bool) []layoutLine {
	const (
		directionBuckets = 72 // five-degree buckets; exact matching remains below
		offsetCell       = 4.0
		maxLineTolerance = 64.0
	)
	var lines []layoutLine
	lineIndex := make([]map[int][]int, directionBuckets)
	prev := -1 // the line the previous glyph joined
	for i, glyph := range glyphs {
		dir := glyphDirection(glyph)
		normal := Point{X: -dir.Y, Y: dir.X}
		start, _ := glyphBaseline(glyph, dir)
		offset := dotPoint(start, normal)
		tol := min(0.55*glyph.Size, maxLineTolerance)
		if tol <= 0 {
			tol = 5
		}
		best := -1
		bestDistance := math.MaxFloat64
		bucket := layoutDirectionBucket(dir, directionBuckets)
		// Consecutive glyphs of a run share a baseline exactly, so the
		// previous glyph's line is at distance zero — nothing can beat it
		// — and the neighborhood search below is skipped.
		if prev >= 0 && offset == lines[prev].offset && dotPoint(dir, lines[prev].dir) >= 0.985 {
			best, bestDistance = prev, 0
		}
		offsetBucket := int(math.Floor(offset / offsetCell))
		offsetRadius := int(math.Ceil(tol/offsetCell)) + 1
		for directionDelta := -3; directionDelta <= 3 && best < 0; directionDelta++ {
			directionBucket := (bucket + directionDelta + directionBuckets) % directionBuckets
			cells := lineIndex[directionBucket]
			if cells == nil {
				continue // most pages run in one direction; skip the empty neighbors
			}
			for offsetDelta := -offsetRadius; offsetDelta <= offsetRadius; offsetDelta++ {
				for _, j := range cells[offsetBucket+offsetDelta] {
					if dotPoint(dir, lines[j].dir) < 0.985 {
						continue
					}
					distance := math.Abs(offset - lines[j].offset)
					if distance <= tol && distance < bestDistance {
						best, bestDistance = j, distance
					}
				}
			}
		}
		item := layoutGlyph{Glyph: &glyphs[i], index: i}
		if best < 0 {
			lines = append(lines, layoutLine{
				glyphs: []layoutGlyph{item},
				dir:    dir,
				normal: normal,
				offset: offset,
				first:  i,
			})
			if lineIndex[bucket] == nil {
				lineIndex[bucket] = map[int][]int{}
			}
			key := int(math.Floor(offset / offsetCell))
			lineIndex[bucket][key] = append(lineIndex[bucket][key], len(lines)-1)
			prev = len(lines) - 1
			continue
		}
		lines[best].glyphs = append(lines[best].glyphs, item)
		prev = best
	}
	for i, n := 0, len(lines); i < n; i++ {
		if layoutLineSorted(&lines[i]) {
			// Text drawn in reading order, the common case.
			if !keepDuplicates {
				lines[i].glyphs = dropDuplicateGlyphs(lines[i].glyphs, lines[i].dir)
			}
			continue
		}
		sortLineGlyphs(&lines[i])
		if !keepDuplicates {
			lines[i].glyphs = dropDuplicateGlyphs(lines[i].glyphs, lines[i].dir)
		}
		lines = append(lines, splitOverlaidRuns(&lines[i])...)
	}
	return lines
}

// sortLineGlyphs orders a line's glyphs along its direction, keeping
// content order among glyphs at the same position.
func sortLineGlyphs(line *layoutLine) {
	slices.SortStableFunc(line.glyphs, func(a, b layoutGlyph) int {
		ap := glyphProjection(*a.Glyph, line.dir, false)
		bp := glyphProjection(*b.Glyph, line.dir, false)
		switch {
		case ap < bp:
			return -1
		case ap > bp:
			return 1
		default:
			return a.index - b.index
		}
	})
}

// Duplicate tolerances, as fractions of the font size along and across the
// baseline: poppler's dupMaxPriDelta and dupMaxSecDelta.
const (
	duplicateAlong  = 0.1
	duplicateAcross = 0.2
)

// dropDuplicateGlyphs removes glyphs that repeat another on the line —
// the same text at the same place and size — which is how fake bold,
// drop shadows, and fill-then-stroke headings are painted. The glyphs are
// in order along dir, so a duplicate is among the few before it. Of each
// pair it keeps the one drawn first: the copies may sit a hundredth of a
// point apart in either direction, and keeping one copy whole keeps its
// run intact for splitOverlaidRuns. It filters in place.
func dropDuplicateGlyphs(glyphs []layoutGlyph, dir Point) []layoutGlyph {
	kept := glyphs[:0]
	for _, g := range glyphs {
		along := glyphProjection(*g.Glyph, dir, false)
		tol := duplicateAlong * max(g.Size, 1)
		duplicate := false
		for j := len(kept) - 1; j >= 0; j-- {
			k := kept[j]
			if along-glyphProjection(*k.Glyph, dir, false) > tol {
				break
			}
			if isDuplicateGlyph(k.Glyph, g.Glyph, dir) {
				if g.index < k.index {
					kept[j] = g
				}
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, g)
		}
	}
	return kept
}

func isDuplicateGlyph(a, b *Glyph, dir Point) bool {
	if a.Text != b.Text || math.Abs(a.Size-b.Size) > duplicateAlong*max(a.Size, 1) {
		return false
	}
	d := Point{X: b.X - a.X, Y: b.Y - a.Y}
	normal := Point{X: -dir.Y, Y: dir.X}
	size := max(a.Size, 1)
	return math.Abs(dotPoint(d, dir)) <= duplicateAlong*size &&
		math.Abs(dotPoint(d, normal)) <= duplicateAcross*size
}

// splitOverlaidRuns separates text painted over other text on the same
// baseline — a stamp or overlay across a line, or two strings placed at
// the same origin — which sorting glyph by glyph would interleave letter
// by letter. The line's glyphs, sorted along its direction, are divided
// into runs as the content stream drew them; a run that lays two or more
// glyphs over text already placed moves to a line of its own, returned
// for the caller to append. Single overlapping glyphs, such as accents
// positioned over their letter, stay in place.
func splitOverlaidRuns(line *layoutLine) []layoutLine {
	if len(line.glyphs) < 4 {
		return nil
	}
	byIndex := slices.Clone(line.glyphs)
	slices.SortFunc(byIndex, func(a, b layoutGlyph) int { return a.index - b.index })

	type run struct {
		glyphs     []layoutGlyph
		start, end float64
	}
	var runs []run
	for i, g := range byIndex {
		start := glyphProjection(*g.Glyph, line.dir, false)
		end := glyphProjection(*g.Glyph, line.dir, true)
		if i > 0 {
			r := &runs[len(runs)-1]
			// A run continues while the pen moves forward; kerning may
			// step back a little.
			if start >= r.end-0.3*max(g.Size, 1) {
				r.glyphs = append(r.glyphs, g)
				r.end = max(r.end, end)
				continue
			}
		}
		runs = append(runs, run{glyphs: []layoutGlyph{g}, start: start, end: end})
	}
	if len(runs) < 2 {
		return nil
	}
	slices.SortStableFunc(runs, func(a, b run) int {
		switch {
		case a.start < b.start:
			return -1
		case a.start > b.start:
			return 1
		}
		return 0
	})

	// Place each run on the first track whose placed runs it does not
	// cover with two or more glyphs.
	type track struct {
		glyphs []layoutGlyph
		ends   []float64 // per placed run: [start, end] pairs
	}
	overlaps := func(t *track, r run) bool {
		for k := 0; k < len(t.ends); k += 2 {
			covered := 0
			for _, g := range r.glyphs {
				start := glyphProjection(*g.Glyph, line.dir, false)
				tol := 0.3 * max(g.Size, 1)
				if start > t.ends[k]-tol && start < t.ends[k+1]-tol {
					covered++
				}
			}
			if covered >= 2 {
				return true
			}
		}
		return false
	}
	var tracks []track
	for _, r := range runs {
		placed := false
		for t := range tracks {
			if !overlaps(&tracks[t], r) {
				tracks[t].glyphs = append(tracks[t].glyphs, r.glyphs...)
				tracks[t].ends = append(tracks[t].ends, r.start, r.end)
				placed = true
				break
			}
		}
		if !placed {
			tracks = append(tracks, track{glyphs: r.glyphs, ends: []float64{r.start, r.end}})
		}
	}
	if len(tracks) < 2 {
		return nil
	}
	extra := make([]layoutLine, 0, len(tracks)-1)
	for t := range tracks {
		split := *line
		split.glyphs = tracks[t].glyphs
		split.first = slices.MinFunc(split.glyphs, func(a, b layoutGlyph) int { return a.index - b.index }).index
		sortLineGlyphs(&split)
		if t == 0 {
			*line = split
			continue
		}
		extra = append(extra, split)
	}
	return extra
}

// layoutLineSorted reports whether the line's glyphs, which are in index
// order, are already in non-decreasing baseline order — the order the
// stable sort below would produce.
func layoutLineSorted(line *layoutLine) bool {
	prev := math.Inf(-1)
	for _, g := range line.glyphs {
		p := glyphProjection(*g.Glyph, line.dir, false)
		if p < prev {
			return false
		}
		prev = p
	}
	return true
}

func layoutDirectionBucket(dir Point, buckets int) int {
	angle := math.Atan2(dir.Y, dir.X)
	bucket := int(math.Floor((angle + math.Pi) / (2 * math.Pi) * float64(buckets)))
	if bucket >= buckets {
		return 0
	}
	return bucket
}

func compareLayoutLines(a, b layoutLine) int {
	if dotPoint(a.dir, b.dir) >= 0.985 {
		switch {
		case a.offset > b.offset:
			return -1
		case a.offset < b.offset:
			return 1
		}
	}
	return a.first - b.first
}

func writeLayoutLine(b *strings.Builder, line layoutLine) {
	if lineHasRightToLeft(line) {
		writeBidiLine(b, line)
		return
	}
	walkLayoutLine(line, func(text string) { b.WriteString(text) })
}

// walkLayoutLine calls emit with the line's text in order: each glyph's
// text, and a space wherever a gap between glyphs implies one. Whitespace
// at either end of the line is dropped, and so is a space glyph lying
// within a neighbouring glyph that it was not drawn next to — a stray from
// a smaller line whose baseline falls within this one's tolerance, which
// would otherwise split a word. A space drawn between its neighbours stays
// even when kerning pulls them over it, as right-aligned page numbers do.
func walkLayoutLine(line layoutLine, emit func(string)) {
	pending := ""     // whitespace glyphs held until text follows them
	pendingMid := 0.0 // midpoint of the last of them
	pendingIndex := 0 // content index of the last of them
	textEnd := 0.0    // prevEnd before them
	prevIndex := 0    // content index of the last glyph written
	wrote := false
	prevEnd := 0.0
	for _, glyph := range line.glyphs {
		start := glyphProjection(*glyph.Glyph, line.dir, false)
		end := glyphProjection(*glyph.Glyph, line.dir, true)
		if strings.TrimSpace(glyph.Text) == "" {
			mid := (start + end) / 2
			if wrote && (prevEnd <= mid || glyph.index == prevIndex+1) {
				if pending == "" {
					textEnd = prevEnd
				}
				pending += glyph.Text
				pendingMid, pendingIndex = mid, glyph.index
				prevEnd = end
			}
			continue
		}
		if pending != "" && start < pendingMid && glyph.index != pendingIndex+1 {
			pending, prevEnd = "", textEnd // the space lies within this glyph
		}
		if wrote {
			switch {
			case pending != "":
				emit(pending)
			case !strings.HasPrefix(glyph.Text, " "):
				threshold := 0.17 * glyph.Size
				if threshold <= 0 {
					threshold = 1
				}
				if start-prevEnd > threshold {
					emit(" ")
				}
			}
		}
		text := strings.TrimRightFunc(glyph.Text, unicode.IsSpace)
		emit(text)
		pending = glyph.Text[len(text):]
		pendingMid, pendingIndex = end, glyph.index
		wrote = true
		prevEnd = end
		prevIndex = glyph.index
	}
}

func glyphDirection(g Glyph) Point { return g.direction() }

// glyphBaseline is Baseline for a direction already computed by the
// caller, which every layout pass holds for the line being built.
func glyphBaseline(g Glyph, dir Point) (Point, Point) {
	length := math.Abs(g.Advance)
	start := Point{X: g.X, Y: g.Y}
	return start, Point{X: start.X + dir.X*length, Y: start.Y + dir.Y*length}
}

func glyphProjection(g Glyph, dir Point, end bool) float64 {
	if !end {
		// The origin's projection needs no baseline: this is what
		// glyphBaseline's start would give, without computing its end.
		return g.X*dir.X + g.Y*dir.Y
	}
	_, finish := glyphBaseline(g, dir)
	return dotPoint(finish, dir)
}

func dotPoint(a, b Point) float64 { return a.X*b.X + a.Y*b.Y }

func orderColumns(lines []layoutLine, box Rect) []layoutLine {
	var fragments []layoutLine
	for _, line := range lines {
		if math.Abs(line.dir.Y) > 0.25 || len(line.glyphs) < 2 {
			fragments = append(fragments, line)
			continue
		}
		start := 0
		for i := 1; i < len(line.glyphs); i++ {
			prev := line.glyphs[i-1]
			current := line.glyphs[i]
			gap := glyphProjection(*current.Glyph, line.dir, false) -
				glyphProjection(*prev.Glyph, line.dir, true)
			size := max(prev.Size, current.Size)
			threshold := max(4*max(size, 1), 0.04*(box.MaxX-box.MinX))
			if gap > threshold {
				fragments = append(fragments, lineFragment(line, start, i))
				start = i
			}
		}
		fragments = append(fragments, lineFragment(line, start, len(line.glyphs)))
	}

	type column struct {
		lines []layoutLine
		x     float64
	}
	type horizontalLine struct {
		line layoutLine
		x    float64
		tol  float64
	}
	var horizontal []horizontalLine
	var other []layoutLine
	for _, line := range fragments {
		if math.Abs(line.dir.Y) > 0.25 || len(line.glyphs) == 0 {
			other = append(other, line)
			continue
		}
		x := line.glyphs[0].X
		tol := 2 * max(line.glyphs[0].Size, 1)
		horizontal = append(horizontal, horizontalLine{line: line, x: x, tol: tol})
	}
	slices.SortStableFunc(horizontal, func(a, b horizontalLine) int {
		switch {
		case a.x < b.x:
			return -1
		case a.x > b.x:
			return 1
		default:
			return a.line.first - b.line.first
		}
	})
	var columns []column
	for _, item := range horizontal {
		if len(columns) == 0 || math.Abs(columns[len(columns)-1].x-item.x) > item.tol {
			columns = append(columns, column{x: item.x, lines: []layoutLine{item.line}})
		} else {
			last := len(columns) - 1
			columns[last].lines = append(columns[last].lines, item.line)
		}
	}
	ordered := make([]layoutLine, 0, len(fragments))
	for _, column := range columns {
		slices.SortStableFunc(column.lines, compareLayoutLines)
		ordered = append(ordered, column.lines...)
	}
	slices.SortStableFunc(other, compareLayoutLines)
	return append(ordered, other...)
}

func lineFragment(line layoutLine, start, end int) layoutLine {
	line.glyphs = line.glyphs[start:end]
	line.first = line.glyphs[0].index
	return line
}

// GlyphsIn returns glyphs whose quadrilateral intersects region.
func (p Page) GlyphsIn(region Rect) []Glyph {
	out := make([]Glyph, 0, len(p.Glyphs))
	for _, glyph := range p.Glyphs {
		if rectsIntersect(glyphRect(glyph), region) {
			out = append(out, glyph)
		}
	}
	return out
}

// TextIn reconstructs text from glyphs intersecting region.
func (p Page) TextIn(region Rect, layout LayoutOptions) string {
	p.Glyphs = p.GlyphsIn(region)
	return p.TextWithOptions(layout)
}

func glyphRect(g Glyph) Rect {
	rect := Rect{
		MinX: math.MaxFloat64,
		MinY: math.MaxFloat64,
		MaxX: -math.MaxFloat64,
		MaxY: -math.MaxFloat64,
	}
	for _, point := range g.Quad() {
		rect.MinX = min(rect.MinX, point.X)
		rect.MinY = min(rect.MinY, point.Y)
		rect.MaxX = max(rect.MaxX, point.X)
		rect.MaxY = max(rect.MaxY, point.Y)
	}
	return rect
}

func rectsIntersect(a, b Rect) bool {
	return a.MaxX >= b.MinX && b.MaxX >= a.MinX && a.MaxY >= b.MinY && b.MaxY >= a.MinY
}
