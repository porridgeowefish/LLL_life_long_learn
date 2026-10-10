package pdf

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/giraffesyo/pdf/internal/crypt"
	"github.com/giraffesyo/pdf/internal/safeio"
)

// ErrPasswordRequired reports that an encrypted document could not be
// unlocked with the supplied password.
var ErrPasswordRequired = crypt.ErrPasswordRequired

// WarningCode identifies a recoverable extraction problem.
type WarningCode string

const (
	// WarningMalformedDocument reports malformed document-level structures
	// such as outlines or form-field trees.
	WarningMalformedDocument WarningCode = "malformed_document"
	// WarningMalformedPage reports malformed page content or page extras.
	WarningMalformedPage WarningCode = "malformed_page"
	// WarningStream reports a stream decoding or filter failure.
	WarningStream WarningCode = "stream_error"
	// WarningStreamLimit reports decoded data truncated at a configured limit.
	WarningStreamLimit WarningCode = "stream_limit"
	// WarningWorkLimit reports an operator, glyph, or nesting budget.
	WarningWorkLimit WarningCode = "work_limit"
	// WarningUnsupported reports a recoverable unsupported PDF feature.
	WarningUnsupported WarningCode = "unsupported"
	// WarningOCR reports an external OCR implementation failure.
	WarningOCR WarningCode = "ocr"
)

// Warning describes a condition that may have made an extraction incomplete.
// Page is one-based, or zero for a document-level warning.
type Warning struct {
	Page int
	Code WarningCode
	Err  error
}

func (w Warning) Error() string {
	if w.Page > 0 {
		return fmt.Sprintf("pdf: page %d: %s: %v", w.Page, w.Code, w.Err)
	}
	return fmt.Sprintf("pdf: %s: %v", w.Code, w.Err)
}

func (w Warning) Unwrap() error { return w.Err }

// StrictError is returned when strict extraction encounters a condition that
// permissive extraction would record as a Warning.
type StrictError struct {
	Warning Warning
}

func (e *StrictError) Error() string { return e.Warning.Error() }
func (e *StrictError) Unwrap() error { return e.Warning.Err }

// PageRange selects an inclusive, one-based page range. A Last value of zero
// means the final page.
type PageRange struct {
	First int
	Last  int
}

// Limits bounds work performed on untrusted documents. Zero fields use the
// package defaults.
type Limits struct {
	MaxStreamBytes      int
	MaxOperatorsPerPage int
	MaxGlyphsPerPage    int
	MaxFormDepth        int

	// MaxImagesPerPage bounds how many image paintings a page is read
	// for, whether or not their data is collected. Default 10,000.
	MaxImagesPerPage int
	// MaxImageBytesPerPage bounds the image data read for one page, over
	// all its distinct images, each of which MaxStreamBytes bounds on its
	// own. Images past the budget are left out with a WarningStreamLimit.
	// Default 256 MiB.
	MaxImageBytesPerPage int
	// MaxImagePixels bounds the images Image.Decode is willing to
	// allocate, in pixels. Default 64 Mi (a 600 dpi letter page is
	// 34 Mi). Larger images are still reported with their encoded Data.
	MaxImagePixels int
}

func (l Limits) normalized() Limits {
	if l.MaxStreamBytes == 0 {
		l.MaxStreamBytes = safeio.MaxStreamBytes
	}
	if l.MaxOperatorsPerPage == 0 {
		l.MaxOperatorsPerPage = maxOpsPerPage
	}
	if l.MaxGlyphsPerPage == 0 {
		l.MaxGlyphsPerPage = maxGlyphsPerPage
	}
	if l.MaxFormDepth == 0 {
		l.MaxFormDepth = maxFormDepth
	}
	if l.MaxImagesPerPage == 0 {
		l.MaxImagesPerPage = maxImagesPerPage
	}
	if l.MaxImageBytesPerPage == 0 {
		l.MaxImageBytesPerPage = maxImageBytesPerPage
	}
	if l.MaxImagePixels == 0 {
		l.MaxImagePixels = maxImagePixels
	}
	return l
}

func (l Limits) validate() error {
	switch {
	case l.MaxStreamBytes < 0:
		return errors.New("pdf: MaxStreamBytes must not be negative")
	case l.MaxOperatorsPerPage < 0:
		return errors.New("pdf: MaxOperatorsPerPage must not be negative")
	case l.MaxGlyphsPerPage < 0:
		return errors.New("pdf: MaxGlyphsPerPage must not be negative")
	case l.MaxFormDepth < 0:
		return errors.New("pdf: MaxFormDepth must not be negative")
	case l.MaxImagesPerPage < 0:
		return errors.New("pdf: MaxImagesPerPage must not be negative")
	case l.MaxImageBytesPerPage < 0:
		return errors.New("pdf: MaxImageBytesPerPage must not be negative")
	case l.MaxImagePixels < 0:
		return errors.New("pdf: MaxImagePixels must not be negative")
	default:
		return nil
	}
}

// LayoutMode selects how Page.Text reconstructs text.
type LayoutMode uint8

const (
	// LayoutPosition groups glyphs using their geometric baselines.
	LayoutPosition LayoutMode = iota
	// LayoutContentOrder preserves the order in which glyphs occur in content
	// streams, inserting line breaks only when the baseline changes.
	LayoutContentOrder
	// LayoutColumns separates wide horizontal gaps into column regions and
	// reads each column from top to bottom.
	LayoutColumns
)

// LayoutOptions controls plain-text reconstruction.
type LayoutOptions struct {
	Mode LayoutMode

	// KeepDuplicateGlyphs keeps glyphs that repeat another on the same
	// line: the same text painted at the same place and size, as fake
	// bold, drop shadows, and fill-then-stroke headings are drawn. By
	// default text reconstruction reads each such glyph once, as poppler
	// and PDFBox do; Page.Glyphs reports every painted glyph either way.
	KeepDuplicateGlyphs bool
}

// Point is a coordinate in unrotated PDF page space.
type Point struct {
	X float64
	Y float64
}

// Rect is an axis-aligned rectangle in unrotated PDF page space.
type Rect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// Line describes a directed line segment.
type Line struct {
	Start Point
	End   Point
}

// Quad describes a glyph-region perimeter. The first edge follows the
// baseline direction and the opposite edge returns toward its origin.
type Quad [4]Point

// Metadata contains common document information dictionary and XMP fields.
// Dates retain their original PDF string representation.
type Metadata struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreationDate string
	ModifiedDate string
	Trapped      string
	XMP          []byte
	Custom       map[string]string
}

// Destination identifies a named destination or a destination on a page.
type Destination struct {
	Name string
	Page int
	X    float64
	Y    float64
	Zoom float64
}

// Outline is one bookmark in the document outline tree.
type Outline struct {
	Title       string
	Destination Destination
	Children    []Outline
}

// Annotation contains text-bearing annotation and link information.
type Annotation struct {
	Subtype     string
	Rect        Rect
	Contents    string
	Title       string
	URL         string
	Destination Destination
}

// FormField is one terminal AcroForm field.
type FormField struct {
	Name          string
	AlternateName string
	Type          string
	Value         string
	Page          int
}

// CMapResolver supplies a named predefined CMap that is not built into the
// package. The returned bytes use normal PDF CMap syntax.
type CMapResolver func(name string) ([]byte, error)

// OCRRequest is passed to an external OCR implementation for a page the
// OCRPolicy, or OCRSelect, selects. Page carries whatever the content streams yielded —
// its Glyphs, and its Images with their placement and encoded data — so
// an implementation can OCR the page's images directly, through
// Image.Decode or by handing the still-encoded Data to an engine that
// reads JPEG. Reader and Size identify the original PDF for
// implementations that render the page with an external renderer
// instead, which is what text converted to vector outlines needs.
type OCRRequest struct {
	Reader     io.ReaderAt
	Size       int64
	PageNumber int
	Page       Page
}

// OCR supplies text for pages the content streams cannot. Implementations
// invoke an external renderer or OCR engine; the core package remains
// dependency-free. The glyphs returned are positioned in unrotated page
// space (Image.ToPage maps an engine's image coordinates there) and are
// appended to the page's glyphs; Page.OCRGlyphs counts them. Glyphs
// returned together with an error are kept, and the error is recorded as
// a WarningOCR.
//
// Pages are OCR'd concurrently when Options.Concurrency allows: an
// implementation must be safe for concurrent use, or the caller sets
// Concurrency to one.
type OCR interface {
	ExtractPage(context.Context, OCRRequest) ([]Glyph, error)
}

// OCRPolicy selects the pages an OCR implementation is asked about.
type OCRPolicy uint8

const (
	// OCRTextlessPages selects pages whose content streams produced no
	// glyphs: scanned pages, and pages whose text was converted to
	// outlines. This is the default.
	OCRTextlessPages OCRPolicy = iota
	// OCRImagePages selects pages that paint at least one image, with or
	// without text of their own, for documents that mix typeset text and
	// scanned figures or stamps.
	OCRImagePages
	// OCRAllPages selects every extracted page.
	OCRAllPages
)

// OCRFunc adapts a function to OCR.
type OCRFunc func(context.Context, OCRRequest) ([]Glyph, error)

// ExtractPage calls f.
func (f OCRFunc) ExtractPage(ctx context.Context, req OCRRequest) ([]Glyph, error) {
	return f(ctx, req)
}

// Options controls extraction. Its zero value preserves Extract's permissive,
// all-pages behavior.
type Options struct {
	Password string
	Pages    []PageRange
	Strict   bool
	Limits   Limits
	Layout   LayoutOptions

	IgnoreArtifacts bool
	// IgnoreAnnotationAppearances leaves out the text that annotations
	// paint over the page: filled form fields, free-text comments, stamps.
	// By default a page's glyphs include it, after the page's own content,
	// as a viewer shows it and as poppler and MuPDF extract it; hidden
	// annotations are skipped either way. Text and choice fields are drawn
	// from their values when the form asks viewers to regenerate
	// appearances (/NeedAppearances) or a field has none. IncludeAnnotations
	// and IncludeFormValues report annotations and field values as data,
	// independently of this.
	IgnoreAnnotationAppearances bool
	IncludeMetadata             bool
	IncludeOutlines             bool
	IncludeAnnotations          bool
	IncludeFormValues           bool
	// IncludeImages reports the images each page paints in Page.Images,
	// with their placement and their data as described by Image. Without
	// it image data is never read, except for the pages an OCR
	// implementation is asked about, and those pages do not retain it.
	IncludeImages bool

	// PreserveLigatures keeps the Unicode presentation forms U+FB00–U+FB06
	// (ﬀ ﬁ ﬂ ﬃ ﬄ ﬅ ﬆ) in glyph text. By default they fold to their letter
	// sequences (ff, fi, fl, ffi, ffl, st) — the form searchers, tokenizers
	// and other extractors expect — whether the glyph reached the ligature
	// codepoint through a glyph name (Type1 /Differences, the standard
	// encodings, an embedded font program) or through a ToUnicode CMap.
	PreserveLigatures bool

	CMapResolver CMapResolver

	// OCR is asked for text on the pages OCRPolicy selects. It runs after
	// the page's content streams are extracted, with the page's glyphs
	// and images in hand.
	OCR       OCR
	OCRPolicy OCRPolicy

	// OCRSelect chooses the pages OCR is asked about, and overrides
	// OCRPolicy where it is set. The three policies are the answers
	// worth naming, not the only reasonable ones: a form with a typed
	// header over a scanned body has glyphs, so OCRTextlessPages passes
	// it by, while OCRImagePages reads every figure in a document that
	// is mostly typeset. A predicate can say what the document actually
	// needs — a floor on glyphs per page, a page range, a size the
	// images have to reach.
	//
	// The page it receives is the one the content streams produced:
	// Glyphs and ImageCount are set, and Images is still empty, because
	// image data is read only once something asks for it. Like OCR
	// itself, it is called from the page workers and must be safe for
	// concurrent use; see Concurrency.
	OCRSelect func(page Page) bool

	// Concurrency bounds how many pages are extracted at once. Zero picks
	// a worker per available processor; one extracts sequentially.
	// Extraction stays deterministic either way: pages, their glyphs, and
	// their warnings come out in the same order.
	//
	// Pages run concurrently only when nothing observable depends on the
	// order they execute in. Strict stops at the first warning, and
	// CMapResolver is caller code that need not be safe for concurrent
	// use, so those extract sequentially whatever this is set to. OCR
	// implementations are called concurrently, since OCR dominates the
	// cost of a scanned document; set Concurrency to one for an engine
	// that cannot share.
	Concurrency int
}

func (o Options) validate() error {
	if err := o.Limits.validate(); err != nil {
		return err
	}
	for _, p := range o.Pages {
		if p.First < 1 {
			return errors.New("pdf: page range First must be at least 1")
		}
		if p.Last < 0 || p.Last > 0 && p.Last < p.First {
			return errors.New("pdf: page range Last must be zero or at least First")
		}
	}
	switch o.Layout.Mode {
	case LayoutPosition, LayoutContentOrder, LayoutColumns:
	default:
		return errors.New("pdf: unknown layout mode")
	}
	if o.Concurrency < 0 {
		return errors.New("pdf: Concurrency must not be negative")
	}
	switch o.OCRPolicy {
	case OCRTextlessPages, OCRImagePages, OCRAllPages:
	default:
		return errors.New("pdf: unknown OCR policy")
	}
	return nil
}

func selectedPage(n int, ranges []PageRange, total int) bool {
	if len(ranges) == 0 {
		return true
	}
	for _, p := range ranges {
		last := p.Last
		if last == 0 || last > total {
			last = total
		}
		if n >= p.First && n <= last {
			return true
		}
	}
	return false
}
