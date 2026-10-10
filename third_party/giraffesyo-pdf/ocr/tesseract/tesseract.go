// Package tesseract implements pdf.OCR with the Tesseract command-line
// program, as a reference for plugging an OCR engine into
// github.com/giraffesyo/pdf: it OCRs the images a page paints and places
// the recognized words on the page through pdf.Image.ToPage, so they
// flow into Page.Text and the layout modes like typeset text.
//
// The package shells out to the tesseract executable (https://github.com/tesseract-ocr/tesseract),
// which must be installed separately; it has no Go dependencies beyond
// the standard library and the root pdf package. JPEG images are handed
// to Tesseract as they are; everything else goes through pdf.Image.Decode
// and is sent as PNG. Pages without images (text converted to outlines)
// yield nothing: rendering a page is out of scope.
//
//	doc, err := pdf.ExtractWithOptions(ctx, r, size, pdf.Options{
//		OCR: &tesseract.Engine{Languages: []string{"eng"}},
//	})
package tesseract

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/giraffesyo/pdf"
)

// Engine runs Tesseract. The zero value runs "tesseract" from PATH with
// its default language and page segmentation. An Engine is safe for
// concurrent use; every call runs its own process.
type Engine struct {
	// Command is the executable to run; "tesseract" when empty.
	Command string
	// Languages are passed as -l, joined with "+"; Tesseract's default
	// (eng) when empty.
	Languages []string
	// PageSegmentation is Tesseract's --psm; its default (3, automatic)
	// when zero.
	PageSegmentation int
	// MinConfidence drops words Tesseract is less sure of, on its 0–100
	// scale. Zero keeps every word.
	MinConfidence float64
	// MinImageSize skips images whose shorter side is below it, in
	// pixels: icons and rules are not worth a process each. 16 when zero;
	// negative disables the check.
	MinImageSize int
	// Args are appended to the command line, after the options above.
	Args []string
}

// ExtractPage OCRs each image on the page and returns the words found,
// positioned on the page. Images the package cannot decode and Tesseract
// failures are reported together with the words of the images that
// succeeded, so a page is never lost to one bad image.
func (e *Engine) ExtractPage(ctx context.Context, req pdf.OCRRequest) ([]pdf.Glyph, error) {
	var (
		glyphs []pdf.Glyph
		errs   []error
	)
	for i, im := range req.Page.Images {
		if e.skip(im) {
			continue
		}
		words, err := e.run(ctx, im)
		if err != nil {
			errs = append(errs, fmt.Errorf("image %d: %w", i+1, err))
			if ctx.Err() != nil {
				break
			}
			continue
		}
		glyphs = append(glyphs, words...)
	}
	return glyphs, errors.Join(errs...)
}

func (e *Engine) skip(im pdf.Image) bool {
	limit := e.MinImageSize
	if limit == 0 {
		limit = 16
	}
	return limit > 0 && min(im.Width, im.Height) < limit
}

// run OCRs one image and maps Tesseract's word boxes onto the page.
func (e *Engine) run(ctx context.Context, im pdf.Image) ([]pdf.Glyph, error) {
	input, err := e.encode(im)
	if err != nil {
		return nil, err
	}
	args := []string{"stdin", "stdout"}
	if len(e.Languages) > 0 {
		args = append(args, "-l", strings.Join(e.Languages, "+"))
	}
	if e.PageSegmentation > 0 {
		args = append(args, "--psm", strconv.Itoa(e.PageSegmentation))
	}
	if dpi := e.dpi(im); dpi > 0 {
		args = append(args, "--dpi", strconv.Itoa(dpi))
	}
	args = append(args, e.Args...)
	args = append(args, "tsv")

	command := e.Command
	if command == "" {
		command = "tesseract"
	}
	cmd := exec.CommandContext(ctx, command, args...) // #nosec G204 -- the command and arguments are the caller's configuration
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("tesseract: %w: %s", err, lastLine(msg))
		}
		return nil, fmt.Errorf("tesseract: %w", err)
	}
	return e.glyphs(im, stdout.Bytes())
}

// encode returns the image in a format Tesseract reads: its own JPEG
// bytes, or a PNG of the decoded pixels.
func (e *Engine) encode(im pdf.Image) ([]byte, error) {
	if im.Filter == "DCTDecode" {
		return im.Data, nil
	}
	img, err := im.Decode()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dpi estimates the image's resolution from the page area it covers, so
// Tesseract sizes its layout analysis to the text. Zero leaves it to
// Tesseract's own estimate.
func (e *Engine) dpi(im pdf.Image) int {
	b := im.Bounds()
	width := b.MaxX - b.MinX
	if width <= 0 || im.Width <= 0 {
		return 0
	}
	dpi := float64(im.Width) / (width / 72)
	if dpi < 70 || dpi > 2400 || math.IsNaN(dpi) {
		return 0
	}
	return int(math.Round(dpi))
}

// glyphs turns Tesseract's TSV output into page-positioned glyphs, one
// per word. The TSV columns are level, page_num, block_num, par_num,
// line_num, word_num, left, top, width, height, conf, text; words are
// level 5. Boxes are in image pixels from the top-left corner, which is
// exactly what Image.ToPage takes.
func (e *Engine) glyphs(im pdf.Image, tsv []byte) ([]pdf.Glyph, error) {
	var out []pdf.Glyph
	sc := bufio.NewScanner(bytes.NewReader(tsv))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	header := true
	for sc.Scan() {
		if header {
			header = false
			continue
		}
		fields := strings.Split(sc.Text(), "\t")
		if len(fields) < 12 || fields[0] != "5" {
			continue
		}
		text := strings.TrimSpace(fields[11])
		if text == "" {
			continue
		}
		conf, _ := strconv.ParseFloat(fields[10], 64)
		if conf < e.MinConfidence {
			continue
		}
		box, ok := parseBox(fields[6:10])
		if !ok {
			continue
		}
		out = append(out, wordGlyph(im, text, box))
	}
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("tesseract output: %w", err)
	}
	return out, nil
}

type box struct{ left, top, width, height float64 }

func parseBox(fields []string) (box, bool) {
	var vals [4]float64
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil || v < 0 {
			return box{}, false
		}
		vals[i] = v
	}
	b := box{left: vals[0], top: vals[1], width: vals[2], height: vals[3]}
	return b, b.width > 0 && b.height > 0
}

// wordGlyph places a word box on the page: the baseline runs along the
// box's bottom edge, and the ascent reaches its top edge, so rotated
// images yield rotated text.
func wordGlyph(im pdf.Image, text string, b box) pdf.Glyph {
	origin := im.ToPage(b.left, b.top+b.height)
	end := im.ToPage(b.left+b.width, b.top+b.height)
	top := im.ToPage(b.left, b.top)
	dx, dy := end.X-origin.X, end.Y-origin.Y
	advance := math.Hypot(dx, dy)
	g := pdf.Glyph{
		Text:    text,
		X:       origin.X,
		Y:       origin.Y,
		Advance: advance,
		Ascent:  pdf.Point{X: top.X - origin.X, Y: top.Y - origin.Y},
	}
	g.Size = math.Hypot(g.Ascent.X, g.Ascent.Y)
	if advance > 0 {
		g.Direction = pdf.Point{X: dx / advance, Y: dy / advance}
	}
	return g
}

func lastLine(s string) string {
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}
