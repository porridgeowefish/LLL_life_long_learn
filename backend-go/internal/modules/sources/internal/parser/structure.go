package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const ParserVersion = "structured-1"

type ByteRange struct {
	Start int `json:"startByte"`
	End   int `json:"endByte"`
}
type Block struct {
	ID                 string       `json:"id"`
	Type               string       `json:"type"`
	Range              ByteRange    `json:"range"`
	Representation     string       `json:"representation"`
	Provenance         string       `json:"provenance"`
	BBox               []float64    `json:"bbox,omitempty"`
	Confidence         string       `json:"confidence,omitempty"`
	UncertaintyReasons []string     `json:"uncertaintyReasons,omitempty"`
	TableRows          []ByteRange  `json:"tableRows,omitempty"`
	TableMerges        []TableMerge `json:"tableMerges,omitempty"`
}
type TableMerge struct {
	Row      int    `json:"row"`
	Column   int    `json:"column"`
	Span     int    `json:"gridSpan,omitempty"`
	Vertical string `json:"vMerge,omitempty"`
}
type PageGeometry struct {
	MediaBox        []float64 `json:"mediaBox"`
	CropBox         []float64 `json:"cropBox"`
	Rotation        int       `json:"rotation"`
	CoordinateSpace string    `json:"coordinateSpace"`
}
type GlyphGeometry struct {
	Text       string       `json:"text"`
	Quad       [][2]float64 `json:"quad"`
	Baseline   [][2]float64 `json:"baseline"`
	Size       float64      `json:"size"`
	Provenance string       `json:"provenance"`
}
type Unit struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Number   int             `json:"number,omitempty"`
	Name     string          `json:"name,omitempty"`
	Geometry *PageGeometry   `json:"pageGeometry,omitempty"`
	Blocks   []Block         `json:"blocks"`
	Glyphs   []GlyphGeometry `json:"glyphs,omitempty"`
}
type StructureCoverage struct {
	ExtractedUnits      []int    `json:"extractedUnits,omitempty"`
	UnreadUnits         []int    `json:"unreadUnits,omitempty"`
	OmittedBlocks       []string `json:"omittedBlocks,omitempty"`
	CompleteWithinScope bool     `json:"completeWithinScope"`
	Reasons             []string `json:"reasons,omitempty"`
}
type Structure struct {
	SchemaVersion    int               `json:"schemaVersion"`
	SourceRevisionID string            `json:"sourceRevisionId"`
	DerivedVersionID string            `json:"derivedVersionId"`
	OriginalHash     string            `json:"originalHash"`
	CanonicalHash    string            `json:"canonicalHash"`
	ParserVersion    string            `json:"parserVersion"`
	Units            []Unit            `json:"units"`
	Coverage         StructureCoverage `json:"coverage"`
}

// StructureFromMarkdown records byte ranges without duplicating canonical text.
// Display math and tables are indivisible, except explicitly recorded table rows.
func StructureFromMarkdown(content, provenance string) *Structure {
	s := &Structure{SchemaVersion: 1, ParserVersion: ParserVersion, Coverage: StructureCoverage{CompleteWithinScope: true}}
	u := Unit{ID: "lines_1", Kind: "lines"}
	offset := 0
	mathStart := -1
	tableStart := -1
	var rows []ByteRange
	add := func(kind string, start, end int) {
		if end <= start {
			return
		}
		if len(u.Blocks) >= 20000 {
			u.Blocks[len(u.Blocks)-1].Range.End = end
			u.Blocks[len(u.Blocks)-1].Type = "text"
			if len(s.Coverage.OmittedBlocks) == 0 {
				s.Coverage.OmittedBlocks = []string{"structure-block-limit"}
				s.Coverage.Reasons = append(s.Coverage.Reasons, "structure-block-limit")
			}
			s.Coverage.CompleteWithinScope = false
			return
		}
		rep := "markdown"
		if kind == "math" {
			rep = "latex"
		}
		b := Block{ID: fmt.Sprintf("block_%d", len(u.Blocks)+1), Type: kind, Range: ByteRange{start, end}, Representation: rep, Provenance: provenance}
		if kind == "unknown" {
			b.UncertaintyReasons = []string{"unsupported-math-structure"}
			s.Coverage.Reasons = append(s.Coverage.Reasons, "unsupported-math-structure")
			s.Coverage.CompleteWithinScope = false
		}
		if kind == "table" {
			b.TableRows = rows
		}
		u.Blocks = append(u.Blocks, b)
	}
	for _, line := range strings.SplitAfter(content, "\n") {
		trim := strings.TrimSpace(line)
		end := offset + len(line)
		if mathStart >= 0 {
			if strings.Contains(trim, "$$") {
				add("math", mathStart, end)
				mathStart = -1
			}
			offset = end
			continue
		}
		if tableStart >= 0 && !strings.HasPrefix(trim, "|") {
			add("table", tableStart, offset)
			tableStart = -1
			rows = nil
		}
		if strings.HasPrefix(trim, "$$") {
			if strings.Count(trim, "$$") >= 2 {
				add("math", offset, end)
			} else {
				mathStart = offset
			}
		} else if strings.HasPrefix(trim, "|") {
			if tableStart < 0 {
				tableStart = offset
			}
			rows = append(rows, ByteRange{offset, end})
		} else {
			kind := "text"
			if strings.HasPrefix(trim, "#") {
				kind = "heading"
			}
			if strings.Contains(trim, "[数学结构不确定") {
				kind = "unknown"
			}
			add(kind, offset, end)
		}
		offset = end
	}
	if mathStart >= 0 {
		add("unknown", mathStart, offset)
		s.Coverage.Reasons = append(s.Coverage.Reasons, "unclosed-math")
	}
	if tableStart >= 0 {
		add("table", tableStart, offset)
	}
	s.Units = splitCanonicalUnits(content, u)
	return s
}

var canonicalUnit = regexp.MustCompile(`^#{1,6}\s+第\s+(\d+)\s+(页|张幻灯片)\s*$`)

func splitCanonicalUnits(content string, u Unit) []Unit {
	var units []Unit
	current := Unit{ID: u.ID, Kind: u.Kind}
	for _, b := range u.Blocks {
		text := strings.TrimSpace(content[b.Range.Start:b.Range.End])
		match := canonicalUnit.FindStringSubmatch(text)
		if b.Type == "heading" && (match != nil || strings.HasPrefix(text, "## 工作表：")) {
			if len(current.Blocks) > 0 {
				units = append(units, current)
			}
			if match != nil {
				n, _ := strconv.Atoi(match[1])
				kind := "page"
				if match[2] == "张幻灯片" {
					kind = "slide"
				}
				current = Unit{ID: fmt.Sprintf("%s_%d", kind, n), Kind: kind, Number: n}
			} else {
				current = Unit{ID: fmt.Sprintf("sheet_%d", len(units)+1), Kind: "sheet", Name: strings.TrimPrefix(text, "## 工作表：")}
			}
		}
		current.Blocks = append(current.Blocks, b)
	}
	if len(current.Blocks) > 0 {
		units = append(units, current)
	}
	if len(units) == 0 {
		return []Unit{u}
	}
	return units
}
