package retrieval

import (
	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/parser"
	"sort"
	"strings"
)

type protectedBlock struct {
	Start, End int
	ID         string
}

func protectedBlocks(body string) []protectedBlock {
	var out []protectedBlock
	structure := parser.StructureFromMarkdown(body, "local")
	for _, u := range structure.Units {
		for _, b := range u.Blocks {
			if b.Type == "math" || b.Type == "unknown" {
				out = append(out, protectedBlock{b.Range.Start, b.Range.End, b.ID})
			}
			if b.Type == "table" {
				for i, row := range b.TableRows {
					out = append(out, protectedBlock{row.Start, row.End, b.ID + "_row_" + string(rune('A'+i%26))})
				}
			}
		}
	}
	// Inline math is atomic as well. Code fences have already been isolated by
	// the canonical parser; escaped dollars are not delimiters.
	start := -1
	fence := false
	offset := 0
	for _, line := range strings.SplitAfter(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = !fence
		}
		if !fence {
			for i := 0; i < len(line); i++ {
				if line[i] != '$' || i > 0 && line[i-1] == '\\' || i+1 < len(line) && line[i+1] == '$' || i > 0 && line[i-1] == '$' {
					continue
				}
				if start < 0 {
					start = offset + i
				} else {
					out = append(out, protectedBlock{start, offset + i + 1, "inline-math"})
					start = -1
				}
			}
		}
		offset += len(line)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}
func safeEnd(body string, start, end int) int {
	for _, b := range protectedBlocks(body) {
		if start < b.End && end > b.Start && end < b.End {
			return max(start, b.Start)
		}
	}
	return end
}
func alignPreview(body string, start, end, budget int) (int, int, bool) {
	for _, b := range protectedBlocks(body) {
		if start < b.End && end > b.Start {
			start = min(start, b.Start)
			end = max(end, b.End)
		}
	}
	return start, end, EstimateTokens(body[start:end]) <= budget
}
