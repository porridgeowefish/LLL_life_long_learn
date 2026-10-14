package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	pdf "github.com/giraffesyo/pdf"
)

func extractPDF(ctx context.Context, f io.ReaderAt, size int64, out *boundedText) error {
	number := 0
	structure := &Structure{SchemaVersion: 1, ParserVersion: ParserVersion, Coverage: StructureCoverage{CompleteWithinScope: true}}
	out.structure = structure
	geometryGlyphs := 0
	document, err := pdf.ExtractPages(ctx, contextReaderAt{ctx, f}, size, pdf.Options{
		Strict: true, Concurrency: 1,
		Limits: pdf.Limits{MaxStreamBytes: 16 << 20, MaxOperatorsPerPage: 1000000, MaxGlyphsPerPage: 200000, MaxFormDepth: 32, MaxImagesPerPage: 10000, MaxImageBytesPerPage: 16 << 20},
	}, func(page pdf.Page) error {
		number++
		if page.Number > 0 {
			number = page.Number
		}
		unit := Unit{ID: fmt.Sprintf("page_%d", number), Kind: "page", Number: number, Geometry: &PageGeometry{MediaBox: []float64{page.MediaBox.MinX, page.MediaBox.MinY, page.MediaBox.MaxX, page.MediaBox.MaxY}, CropBox: []float64{page.CropBox.MinX, page.CropBox.MinY, page.CropBox.MaxX, page.CropBox.MaxY}, Rotation: page.Rotation, CoordinateSpace: "unrotated-page"}}
		text, simpleMath := pdfText(page)
		if strings.TrimSpace(text) == "" {
			structure.Coverage.UnreadUnits = append(structure.Coverage.UnreadUnits, number)
			structure.Coverage.Reasons = append(structure.Coverage.Reasons, fmt.Sprintf("page-%d-no-text", number))
			structure.Units = append(structure.Units, unit)
			return nil
		}
		start := out.b.Len()
		partial := func(e error) error {
			if errors.Is(e, errOutputFull) {
				unit.Blocks = append(unit.Blocks, Block{ID: fmt.Sprintf("page_%d_partial", number), Type: "unknown", Range: ByteRange{start, out.b.Len()}, Representation: "markdown", Provenance: "local", UncertaintyReasons: []string{"page-text-truncated"}})
				structure.Units = append(structure.Units, unit)
				structure.Coverage.UnreadUnits = append(structure.Coverage.UnreadUnits, number)
				structure.Coverage.Reasons = append(structure.Coverage.Reasons, "page-text-truncated")
			}
			return e
		}
		if err := out.text(fmt.Sprintf("## 第 %d 页\n\n", number)); err != nil {
			return partial(err)
		}
		if err := out.text(text); err != nil {
			return partial(err)
		}
		if e := out.text("\n\n"); e != nil {
			return partial(e)
		}
		block := Block{ID: fmt.Sprintf("page_%d_text", number), Type: "text", Range: ByteRange{start, out.b.Len()}, Representation: "markdown", Provenance: "local", BBox: []float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}}
		if simpleMath {
			block.Type = "math"
			block.Representation = "latex"
			block.Confidence = "high"
		}
		for _, g := range page.Glyphs {
			quad := g.Quad()
			baseline := g.Baseline()
			provenance := "local"
			if g.ActualText {
				provenance = "actual-text"
				block.Provenance = "actual-text"
			}
			gg := GlyphGeometry{Text: g.Text, Size: g.Size, Provenance: provenance, Baseline: [][2]float64{{baseline.Start.X, baseline.Start.Y}, {baseline.End.X, baseline.End.Y}}}
			for _, p := range quad {
				gg.Quad = append(gg.Quad, [2]float64{p.X, p.Y})
				block.BBox[0] = math.Min(block.BBox[0], p.X)
				block.BBox[1] = math.Min(block.BBox[1], p.Y)
				block.BBox[2] = math.Max(block.BBox[2], p.X)
				block.BBox[3] = math.Max(block.BBox[3], p.Y)
			}
			if geometryGlyphs < 20000 {
				unit.Glyphs = append(unit.Glyphs, gg)
				geometryGlyphs++
			} else if len(structure.Coverage.OmittedBlocks) == 0 {
				structure.Coverage.OmittedBlocks = append(structure.Coverage.OmittedBlocks, "glyph-geometry-limit")
				structure.Coverage.Reasons = append(structure.Coverage.Reasons, "glyph-geometry-limit")
			}
		}
		if len(page.Glyphs) == 0 {
			block.BBox = nil
		}
		unit.Blocks = append(unit.Blocks, block)
		structure.Units = append(structure.Units, unit)
		structure.Coverage.ExtractedUnits = append(structure.Coverage.ExtractedUnits, number)
		return nil
	})
	if document != nil && number < document.PageCount {
		for p := number + 1; p <= document.PageCount; p++ {
			structure.Coverage.UnreadUnits = append(structure.Coverage.UnreadUnits, p)
		}
	}
	if err == nil || errors.Is(err, errOutputFull) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, pdf.ErrPasswordRequired) {
		return failure("encrypted_document", err)
	}
	var strict *pdf.StrictError
	if errors.As(err, &strict) {
		switch strict.Warning.Code {
		case pdf.WarningStreamLimit, pdf.WarningWorkLimit:
			return failure("resource_limit", err)
		case pdf.WarningUnsupported:
			// A recoverable page feature must not discard earlier readable pages.
			// Remaining pages already appear in UnreadUnits and can be rendered
			// by PDFium for OCR, even when their font cannot be decoded here.
			if document != nil && document.PageCount > 0 {
				if page := strict.Warning.Page; page > 0 {
					found := false
					for _, n := range structure.Coverage.UnreadUnits {
						if n == page {
							found = true
						}
					}
					if !found {
						structure.Coverage.UnreadUnits = append([]int{page}, structure.Coverage.UnreadUnits...)
					}
					extracted := structure.Coverage.ExtractedUnits[:0]
					for _, n := range structure.Coverage.ExtractedUnits {
						if n != page {
							extracted = append(extracted, n)
						}
					}
					structure.Coverage.ExtractedUnits = extracted
				}
				structure.Coverage.CompleteWithinScope = false
				structure.Coverage.Reasons = append(structure.Coverage.Reasons, "unsupported-page-feature-requires-ocr")
				return nil
			}
			return failure("unsupported_format", err)
		}
	}
	return failure("invalid_document", err)
}

func pdfText(page pdf.Page) (string, bool) {
	if len(page.Glyphs) == 2 {
		base, script := page.Glyphs[0], page.Glyphs[1]
		horizontal := func(g pdf.Glyph) bool {
			b := g.Baseline()
			return math.Abs(b.End.Y-b.Start.Y) < 0.1 && b.End.X > b.Start.X
		}
		simple := func(s string) bool {
			return len(s) == 1 && s[0] >= '0' && s[0] <= '9' || len(s) == 1 && s[0] >= 'a' && s[0] <= 'z' || len(s) == 1 && s[0] >= 'A' && s[0] <= 'Z'
		}
		dx := script.X - base.Baseline().End.X
		dy := script.Y - base.Y
		if !base.ActualText && !script.ActualText && simple(base.Text) && simple(script.Text) && horizontal(base) && horizontal(script) && base.Size > 0 && script.Size <= base.Size*0.75 && script.Size >= base.Size*0.4 && dx >= -base.Size*0.15 && dx <= base.Size*0.35 {
			if dy >= base.Size*0.25 && dy <= base.Size*0.85 {
				return "$$" + base.Text + "^{" + script.Text + "}$$", true
			}
			if dy <= -base.Size*0.2 && dy >= -base.Size*0.65 {
				return "$$" + base.Text + "_{" + script.Text + "}$$", true
			}
		}
	}
	return page.Text(), false
}
