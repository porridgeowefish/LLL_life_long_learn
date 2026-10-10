package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"math"

	"github.com/giraffesyo/pdf/internal/codec/ccitt"
	"github.com/giraffesyo/pdf/internal/codec/jbig2"
	"github.com/giraffesyo/pdf/internal/filter"
	"github.com/giraffesyo/pdf/internal/object"
	"github.com/giraffesyo/pdf/internal/safeio"
)

// ErrImageTooLarge is returned by Image.Decode for an image whose pixel
// count exceeds Limits.MaxImagePixels.
var ErrImageTooLarge = errors.New("pdf: image exceeds the pixel limit")

// Image is one image painted on a page — an image XObject or an inline
// image — with the geometry it was painted at and its data in the most
// directly usable form the package can supply without decoding it.
//
// Data holds the image after decryption and the general-purpose stream
// filters. When Filter is empty, Data is unpacked samples: Height rows of
// Width × Components samples of BitsPerComponent bits each, rows padded to
// a byte boundary. Otherwise Filter names the image codec Data is still
// encoded with: DCTDecode (Data is a complete JPEG file), JPXDecode (a
// JPEG 2000 codestream or JP2 file), CCITTFaxDecode or JBIG2Decode. An
// OCR engine or image library that accepts that format can take Data as
// is; Decode turns the formats the package understands into an
// image.Image.
//
// Soft masks, colour-key masking and rendering intent are not reported.
type Image struct {
	Width, Height    int
	BitsPerComponent int
	// Components is the number of samples per pixel in Data: 1 for image
	// masks and Indexed images, 3 for RGB, 4 for CMYK, and so on. It is
	// zero when the codec carries its own colour information (JPXDecode
	// without a /ColorSpace).
	Components int
	// ColorSpace is the colour space family: DeviceGray, DeviceRGB,
	// DeviceCMYK, Indexed, ICCBased, CalGray, CalRGB, Lab, Separation,
	// DeviceN or Pattern. It is empty for an image mask, and for a
	// JPXDecode image that carries its own colour space.
	ColorSpace string
	// ImageMask reports a stencil mask: one bit per pixel, where sample
	// 0 (or 1 under a /Decode [1 0]) marks the pixels painted in the
	// current fill colour. Scanned bilevel pages are often masks.
	ImageMask bool
	// Inline reports an inline image (BI … ID … EI) rather than an XObject.
	Inline bool
	Filter string
	Data   []byte

	ctm       matrix
	decode    []float64 // the /Decode array, nil for the default
	palette   *indexedPalette
	ccitt     ccitt.Options
	globals   []byte // JBIG2Globals
	maxPixels int
}

// indexedPalette is an Indexed colour space's lookup table.
type indexedPalette struct {
	base   colorSpace
	hival  int
	lookup []byte
}

// ToPage maps a pixel position to unrotated page space: x runs right and
// y runs down from the image's top-left sample, so (0, 0) is the top-left
// corner and (Width, Height) the bottom-right. Coordinates an OCR engine
// reports for the image land on the page through this mapping.
func (im Image) ToPage(x, y float64) Point {
	w, h := float64(max(im.Width, 1)), float64(max(im.Height, 1))
	return im.unitToPage(x/w, 1-y/h)
}

// unitToPage maps the image's unit square, (0, 0) at the bottom-left
// corner as PDF defines image space, through the CTM it was painted with.
func (im Image) unitToPage(u, v float64) Point {
	m := im.ctm
	return Point{X: u*m[0] + v*m[2] + m[4], Y: u*m[1] + v*m[3] + m[5]}
}

// Quad returns the region the image was painted over: its bottom-left,
// bottom-right, top-right and top-left corners in page space, in that
// order, so the first edge follows the image's top-to-bottom rows'
// direction and the shape reflects any rotation or skew.
func (im Image) Quad() Quad {
	return Quad{im.unitToPage(0, 0), im.unitToPage(1, 0), im.unitToPage(1, 1), im.unitToPage(0, 1)}
}

// Bounds returns the page-space bounding box of Quad.
func (im Image) Bounds() Rect {
	q := im.Quad()
	r := Rect{MinX: q[0].X, MinY: q[0].Y, MaxX: q[0].X, MaxY: q[0].Y}
	for _, p := range q[1:] {
		r.MinX, r.MaxX = min(r.MinX, p.X), max(r.MaxX, p.X)
		r.MinY, r.MaxY = min(r.MinY, p.Y), max(r.MaxY, p.Y)
	}
	return r
}

// Decode returns the image's pixels. Unpacked samples in the Device, Cal,
// ICCBased, Indexed and single-colorant Separation and DeviceN spaces, and
// image masks, decode at 1, 2, 4, 8 or 16 bits per component; DCTDecode
// decodes through image/jpeg; CCITTFaxDecode and JBIG2Decode (generic,
// symbol and text regions with arithmetic coding) decode natively to
// one-bit grey. Other colour spaces, JPXDecode, and JBIG2 features outside
// that subset return an error wrapping errors.ErrUnsupported. An image
// with more than Limits.MaxImagePixels pixels returns ErrImageTooLarge
// rather than allocating.
//
// Results are *image.Gray, *image.Gray16, *image.Paletted (Indexed),
// *image.RGBA, *image.CMYK, or whatever image/jpeg returns. A stencil
// mask decodes to *image.Gray with painted pixels black. CCITT data that
// ends early yields the rows decoded, the rest white.
func (im Image) Decode() (image.Image, error) {
	if im.Width <= 0 || im.Height <= 0 {
		return nil, errors.New("pdf: image has no dimensions")
	}
	if limit := im.maxPixels; limit > 0 && im.Width > limit/im.Height {
		return nil, fmt.Errorf("%w: %d×%d", ErrImageTooLarge, im.Width, im.Height)
	}
	switch im.Filter {
	case "":
		return im.decodeSamples(im.Data)
	case "DCTDecode":
		return im.decodeJPEG()
	case "CCITTFaxDecode":
		opts := im.ccitt
		opts.Columns, opts.Rows = im.Width, im.Height
		bits, rows, err := ccitt.Decode(im.Data, opts)
		if rows == 0 {
			if err == nil {
				err = errors.New("no rows")
			}
			return nil, fmt.Errorf("pdf: decode CCITTFaxDecode image: %w", err)
		}
		return im.decodeBilevel(bits, rows)
	case "JBIG2Decode":
		bits, err := jbig2.Decode(im.Data, im.globals, im.Width, im.Height)
		if err != nil {
			return nil, fmt.Errorf("pdf: decode JBIG2Decode image: %w", err)
		}
		// JBIG2 is 1 = black; the filter delivers DeviceGray, 0 = black.
		for i := range bits {
			bits[i] = ^bits[i]
		}
		return im.decodeBilevel(bits, im.Height)
	default:
		return nil, fmt.Errorf("pdf: decode %s image: %w", im.Filter, errors.ErrUnsupported)
	}
}

// decodeBilevel decodes one-bit-per-pixel rows from a bilevel codec,
// padding missing rows white.
func (im Image) decodeBilevel(bits []byte, rows int) (image.Image, error) {
	bi := im
	bi.BitsPerComponent, bi.Components = 1, 1
	if !bi.ImageMask && bi.palette == nil {
		bi.ColorSpace = "DeviceGray"
	}
	stride := (im.Width + 7) / 8
	if rows < im.Height {
		white := byte(0xFF)
		if bi.inverted(0) {
			white = 0
		}
		full := make([]byte, stride*im.Height)
		copy(full, bits[:min(len(bits), stride*rows)])
		for i := stride * rows; i < len(full); i++ {
			full[i] = white
		}
		bits = full
	}
	return bi.decodeSamples(bits)
}

func (im Image) decodeJPEG() (image.Image, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(im.Data))
	if err != nil {
		return nil, fmt.Errorf("pdf: decode DCTDecode image: %w", err)
	}
	if limit := im.maxPixels; limit > 0 && cfg.Height > 0 && cfg.Width > limit/cfg.Height {
		return nil, fmt.Errorf("%w: %d×%d", ErrImageTooLarge, cfg.Width, cfg.Height)
	}
	img, err := jpeg.Decode(bytes.NewReader(im.Data))
	if err != nil {
		return nil, fmt.Errorf("pdf: decode DCTDecode image: %w", err)
	}
	if im.inverted(0) {
		switch pix := img.(type) {
		case *image.Gray:
			invertBytes(pix.Pix)
		case *image.CMYK:
			invertBytes(pix.Pix)
		}
	}
	return img, nil
}

func invertBytes(b []byte) {
	for i := range b {
		b[i] = ^b[i]
	}
}

// inverted reports whether the /Decode array flips component c: its
// maximum comes first. The default array and arrays too short to cover
// the component never invert.
func (im Image) inverted(c int) bool {
	return 2*c+1 < len(im.decode) && im.decode[2*c] > im.decode[2*c+1]
}

// decodeSamples unpacks rows of BitsPerComponent-bit samples. Data that
// is too short reads as zero samples.
func (im Image) decodeSamples(data []byte) (image.Image, error) {
	bpc, ncomp := im.BitsPerComponent, im.Components
	switch bpc {
	case 1, 2, 4, 8, 16:
	default:
		return nil, fmt.Errorf("pdf: decode image with %d bits per component: %w", bpc, errors.ErrUnsupported)
	}
	if ncomp < 1 {
		return nil, errors.New("pdf: image has no colour components")
	}
	w, h := im.Width, im.Height
	stride := (w*ncomp*bpc + 7) / 8
	rows := sampleRows{data: data, stride: stride, bpc: bpc, maxval: 1<<bpc - 1}
	rect := image.Rect(0, 0, w, h)

	family := im.ColorSpace
	if im.ImageMask {
		family = "DeviceGray"
	}
	switch {
	case im.palette != nil && ncomp == 1:
		return im.decodeIndexed(rows, rect)
	case ncomp == 1 && grayFamily(family):
		invert := im.inverted(0)
		if family == "Separation" || family == "DeviceN" {
			invert = !invert // tint 1 is full colorant: dark
		}
		if bpc == 16 {
			img := image.NewGray16(rect)
			for y := range h {
				for x := range w {
					v := rows.sample(y, x)
					if invert {
						v = 0xFFFF - v
					}
					img.Pix[y*img.Stride+2*x] = byte(v >> 8 & 0xFF)
					img.Pix[y*img.Stride+2*x+1] = byte(v & 0xFF)
				}
			}
			return img, nil
		}
		img := image.NewGray(rect)
		if bpc == 8 && !invert {
			for y := range h {
				copy(img.Pix[y*img.Stride:(y+1)*img.Stride], rows.row(y))
			}
			return img, nil
		}
		for y := range h {
			for x := range w {
				v := rows.sample(y, x)
				if invert {
					v = rows.maxval - v
				}
				img.Pix[y*img.Stride+x] = byte(v * 255 / rows.maxval & 0xFF)
			}
		}
		return img, nil
	case ncomp == 3 && (family == "DeviceRGB" || family == "CalRGB" || family == "ICCBased"):
		img := image.NewRGBA(rect)
		for y := range h {
			for x := range w {
				i := y*img.Stride + 4*x
				img.Pix[i] = im.component(rows, y, 3*x, 0)
				img.Pix[i+1] = im.component(rows, y, 3*x+1, 1)
				img.Pix[i+2] = im.component(rows, y, 3*x+2, 2)
				img.Pix[i+3] = 0xFF
			}
		}
		return img, nil
	case ncomp == 4 && (family == "DeviceCMYK" || family == "ICCBased"):
		img := image.NewCMYK(rect)
		for y := range h {
			for x := range w {
				i := y*img.Stride + 4*x
				for c := range 4 {
					img.Pix[i+c] = im.component(rows, y, 4*x+c, c)
				}
			}
		}
		return img, nil
	default:
		return nil, fmt.Errorf("pdf: decode %d-component %s image: %w", ncomp, orUnknown(family), errors.ErrUnsupported)
	}
}

func orUnknown(family string) string {
	if family == "" {
		return "untyped"
	}
	return family
}

func grayFamily(family string) bool {
	switch family {
	case "DeviceGray", "CalGray", "ICCBased", "Separation", "DeviceN":
		return true
	}
	return false
}

// component reads sample i of row y scaled to 8 bits, applying the
// /Decode inversion of component c.
func (im Image) component(rows sampleRows, y, i, c int) byte {
	v := rows.sample(y, i)
	if im.inverted(c) {
		v = rows.maxval - v
	}
	if rows.bpc == 16 {
		return byte(v >> 8 & 0xFF)
	}
	return byte(v * 255 / rows.maxval & 0xFF)
}

func (im Image) decodeIndexed(rows sampleRows, rect image.Rectangle) (image.Image, error) {
	pal := im.palette
	n := pal.base.ncomp
	if n < 1 {
		return nil, fmt.Errorf("pdf: decode Indexed image over %s: %w", orUnknown(pal.base.family), errors.ErrUnsupported)
	}
	entries := min(pal.hival+1, 256)
	palette := make(color.Palette, entries)
	for i := range entries {
		var comps [4]byte
		for c := range n {
			if j := i*n + c; j < len(pal.lookup) && c < 4 {
				comps[c] = pal.lookup[j]
			}
		}
		switch {
		case n == 1 && grayFamily(pal.base.family):
			g := comps[0]
			if pal.base.family == "Separation" || pal.base.family == "DeviceN" {
				g = 255 - g
			}
			palette[i] = color.Gray{Y: g}
		case n == 3 && (pal.base.family == "DeviceRGB" || pal.base.family == "CalRGB" || pal.base.family == "ICCBased"):
			palette[i] = color.RGBA{R: comps[0], G: comps[1], B: comps[2], A: 0xFF}
		case n == 4 && (pal.base.family == "DeviceCMYK" || pal.base.family == "ICCBased"):
			palette[i] = color.CMYK{C: comps[0], M: comps[1], Y: comps[2], K: comps[3]}
		default:
			return nil, fmt.Errorf("pdf: decode Indexed image over %d-component %s: %w", n, orUnknown(pal.base.family), errors.ErrUnsupported)
		}
	}
	img := image.NewPaletted(rect, palette)
	last := uint32(entries - 1) // #nosec G115 -- entries is at most 256
	for y := range im.Height {
		for x := range im.Width {
			img.Pix[y*img.Stride+x] = byte(min(rows.sample(y, x), last) & 0xFF)
		}
	}
	return img, nil
}

// sampleRows reads packed samples from row-padded image data. Samples
// beyond the data read as zero.
type sampleRows struct {
	data   []byte
	stride int
	bpc    int
	maxval uint32
}

func (r sampleRows) row(y int) []byte {
	start := y * r.stride
	if start >= len(r.data) {
		return nil
	}
	return r.data[start:min(start+r.stride, len(r.data))]
}

// sample returns the i-th sample of row y.
func (r sampleRows) sample(y, i int) uint32 {
	row := r.row(y)
	switch r.bpc {
	case 8:
		if i < len(row) {
			return uint32(row[i])
		}
		return 0
	case 16:
		if 2*i+1 < len(row) {
			return uint32(row[2*i])<<8 | uint32(row[2*i+1])
		}
		return 0
	default:
		bit := i * r.bpc
		if bit/8 >= len(row) {
			return 0
		}
		shift := 8 - r.bpc - bit%8
		return uint32(row[bit/8]>>shift) & r.maxval
	}
}

// colorSpace is what Decode needs to know about a colour space: its
// family and component count, and the lookup table of an Indexed space.
type colorSpace struct {
	family  string
	ncomp   int
	palette *indexedPalette
}

// resolveColorSpace interprets a /ColorSpace value. Named resources are
// looked up in the resources' /ColorSpace dictionary.
func resolveColorSpace(v, resources object.Value, depth int) colorSpace {
	if depth > 4 {
		return colorSpace{}
	}
	switch v.Kind() {
	case object.Name:
		if cs, ok := deviceColorSpace(v.Name()); ok {
			return cs
		}
		if named := resources.Key("ColorSpace").Key(v.Name()); !named.IsNull() {
			return resolveColorSpace(named, object.Value{}, depth+1)
		}
		if v.Name() == "Pattern" {
			return colorSpace{family: "Pattern", ncomp: 1}
		}
		return colorSpace{}
	case object.Array:
		if v.Len() == 0 {
			return colorSpace{}
		}
		family := v.Index(0).Name()
		switch family {
		case "ICCBased":
			n := int(intOr(v.Index(1).Key("N"), 0))
			if n != 1 && n != 3 && n != 4 {
				alt := resolveColorSpace(v.Index(1).Key("Alternate"), resources, depth+1)
				if alt.ncomp == 0 {
					alt.ncomp = 3
				}
				n = alt.ncomp
			}
			return colorSpace{family: "ICCBased", ncomp: n}
		case "CalRGB", "Lab":
			return colorSpace{family: family, ncomp: 3}
		case "CalGray":
			return colorSpace{family: family, ncomp: 1}
		case "Separation":
			return colorSpace{family: family, ncomp: 1}
		case "DeviceN":
			return colorSpace{family: family, ncomp: max(v.Index(1).Len(), 1)}
		case "Indexed", "I":
			return indexedColorSpace(resolveColorSpace(v.Index(1), resources, depth+1), v.Index(2), v.Index(3))
		case "Pattern":
			return colorSpace{family: family, ncomp: 1}
		case "DeviceGray", "DeviceRGB", "DeviceCMYK", "G", "RGB", "CMYK":
			cs, _ := deviceColorSpace(family)
			return cs
		default:
			return colorSpace{}
		}
	default:
		return colorSpace{}
	}
}

func deviceColorSpace(name string) (colorSpace, bool) {
	switch name {
	case "DeviceGray", "G":
		return colorSpace{family: "DeviceGray", ncomp: 1}, true
	case "DeviceRGB", "RGB":
		return colorSpace{family: "DeviceRGB", ncomp: 3}, true
	case "DeviceCMYK", "CMYK":
		return colorSpace{family: "DeviceCMYK", ncomp: 4}, true
	}
	return colorSpace{}, false
}

// maxPaletteBytes bounds an Indexed lookup table: 256 entries of at most
// 32 components, far beyond anything real.
const maxPaletteBytes = 256 * 32

func indexedColorSpace(base colorSpace, hival, lookup object.Value) colorSpace {
	n, _ := hival.Int64()
	pal := &indexedPalette{base: base, hival: int(min(max(n, 0), 255))}
	switch lookup.Kind() {
	case object.String:
		pal.lookup = []byte(lookup.RawString())
	case object.Stream:
		pal.lookup, _ = readStreamBoundedLimitError(lookup, maxPaletteBytes)
	}
	return colorSpace{family: "Indexed", ncomp: 1, palette: pal}
}

// colorSpaceFromOperand interprets an inline image's /CS operand: a
// device space name or abbreviation, [/Indexed base hival lookup], or a
// name from the resources' /ColorSpace dictionary.
func colorSpaceFromOperand(op operand, resources object.Value) colorSpace {
	switch op.kind {
	case opName:
		if cs, ok := deviceColorSpace(op.name); ok {
			return cs
		}
		return resolveColorSpace(resources.Key("ColorSpace").Key(op.name), object.Value{}, 1)
	case opArr:
		if len(op.arr) == 4 && op.arr[0].kind == opName && (op.arr[0].name == "I" || op.arr[0].name == "Indexed") {
			base := colorSpaceFromOperand(op.arr[1], resources)
			n := 0
			if op.arr[2].kind == opNum {
				n = int(op.arr[2].num)
			}
			pal := &indexedPalette{base: base, hival: min(max(n, 0), 255)}
			if op.arr[3].kind == opStr {
				pal.lookup = op.arr[3].str
			}
			return colorSpace{family: "Indexed", ncomp: 1, palette: pal}
		}
		if len(op.arr) == 1 && op.arr[0].kind == opName {
			return colorSpaceFromOperand(op.arr[0], resources)
		}
		return colorSpace{}
	default:
		return colorSpace{}
	}
}

// pageImage is an image the page walk painted: where, and from what. The
// data is read only once the walk is over and the page's policy says the
// images are wanted, so a document whose images nobody asked for never
// pays to decode them.
type pageImage struct {
	ctm       matrix
	xobject   object.Value // the image XObject, or null for an inline image
	inline    inlineImage
	resources object.Value // for an inline image's named colour space
}

// inlineImage is the dictionary and still-encoded data of a BI … ID … EI
// sequence. The data is copied out of the content buffer, which the
// next page reuses.
type inlineImage struct {
	dict map[string]operand
	data []byte
}

// imageLoader turns a page's recorded images into Images, reading each
// XObject once per page however often it was painted, within the page's
// image byte budget.
type imageLoader struct {
	limits  Limits
	budget  int            // MaxImageBytesPerPage left
	cache   map[int]*Image // by object number
	globals map[int][]byte // JBIG2Globals streams by object number
}

func newImageLoader(limits Limits) *imageLoader {
	return &imageLoader{limits: limits, budget: limits.MaxImageBytesPerPage}
}

// readLimit is how much one more image may read: its own stream limit,
// within what is left of the page's budget.
func (l *imageLoader) readLimit() (int, error) {
	if l.budget <= 0 {
		return 0, fmt.Errorf("page image data: %w", safeio.ErrLimitExceeded)
	}
	return min(l.limits.MaxStreamBytes, l.budget), nil
}

// readAll reads an image's data against the budget.
func (l *imageLoader) readAll(rd io.Reader) ([]byte, error) {
	limit, err := l.readLimit()
	if err != nil {
		return nil, err
	}
	data, err := safeio.ReadAllGuardedLimitError(rd, limit)
	l.budget -= len(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (l *imageLoader) load(src pageImage) (Image, error) {
	if src.xobject.IsNull() {
		im, err := l.loadInline(src.inline, src.resources)
		if err != nil {
			return Image{}, err
		}
		im.ctm = src.ctm
		return im, nil
	}
	num, indirect := src.xobject.ObjectNumber()
	if indirect {
		if cached, ok := l.cache[num]; ok {
			if cached == nil {
				return Image{}, errSkippedImage
			}
			im := *cached
			im.ctm = src.ctm
			return im, nil
		}
	}
	im, err := l.loadXObject(src.xobject)
	if indirect {
		if l.cache == nil {
			l.cache = map[int]*Image{}
		}
		if err != nil {
			l.cache[num] = nil // reported once; later paintings are skipped silently
		} else {
			copied := im
			l.cache[num] = &copied
		}
	}
	if err != nil {
		return Image{}, err
	}
	im.ctm = src.ctm
	return im, nil
}

// errSkippedImage marks a repeated painting of an image whose failure
// was already reported.
var errSkippedImage = errors.New("image skipped")

func (l *imageLoader) loadXObject(v object.Value) (Image, error) {
	im := Image{
		Width:            int(intOr(v.Key("Width"), 0)),
		Height:           int(intOr(v.Key("Height"), 0)),
		BitsPerComponent: int(intOr(v.Key("BitsPerComponent"), 0)),
		maxPixels:        l.limits.MaxImagePixels,
	}
	if mask, _ := v.Key("ImageMask").Bool(); mask {
		im.ImageMask = true
	}
	if err := im.validateDimensions(); err != nil {
		return Image{}, err
	}
	if !im.ImageMask {
		im.setColorSpace(resolveColorSpace(v.Key("ColorSpace"), object.Value{}, 0))
	} else {
		im.BitsPerComponent, im.Components = 1, 1
	}
	im.decode = floatArray(v.Key("Decode"))

	rc, codec, err := v.ImageReader()
	if err != nil {
		return Image{}, err
	}
	defer func() { _ = rc.Close() }() // read-only handle
	im.Data, err = l.readAll(rc)
	if err != nil {
		return Image{}, err
	}
	im.Filter = codec.Name
	switch codec.Name {
	case "CCITTFaxDecode":
		im.ccitt = ccittOptions(
			intOr(codec.Parms.Key("K"), 0),
			intOr(codec.Parms.Key("Columns"), 1728),
			boolOr(codec.Parms.Key("BlackIs1")),
			boolOr(codec.Parms.Key("EncodedByteAlign")),
		)
		im.BitsPerComponent, im.Components = 1, 1
	case "JBIG2Decode":
		im.BitsPerComponent, im.Components = 1, 1
		if globals := codec.Parms.Key("JBIG2Globals"); globals.Kind() == object.Stream {
			im.globals, err = l.loadGlobals(globals)
			if err != nil {
				return Image{}, fmt.Errorf("JBIG2Globals: %w", err)
			}
		}
	case "JPXDecode":
		// The codestream defines what the dictionary leaves out.
	default:
		if err := im.validateSamples(); err != nil {
			return Image{}, err
		}
	}
	return im, nil
}

func (l *imageLoader) loadGlobals(v object.Value) ([]byte, error) {
	num, indirect := v.ObjectNumber()
	if indirect {
		if data, ok := l.globals[num]; ok {
			return data, nil
		}
	}
	rc, err := v.Reader()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }() // read-only handle
	data, err := l.readAll(rc)
	if err != nil {
		return nil, err
	}
	if indirect {
		if l.globals == nil {
			l.globals = map[int][]byte{}
		}
		l.globals[num] = data
	}
	return data, nil
}

// loadInline builds an Image from an inline image's abbreviated (or full)
// dictionary keys and its data, applying any general-purpose filters and
// stopping at an image codec as ImageReader does for XObjects.
func (l *imageLoader) loadInline(src inlineImage, resources object.Value) (Image, error) {
	d := src.dict
	im := Image{
		Inline:           true,
		Width:            int(operandNumber(d, "W", "Width")),
		Height:           int(operandNumber(d, "H", "Height")),
		BitsPerComponent: int(operandNumber(d, "BPC", "BitsPerComponent")),
		maxPixels:        l.limits.MaxImagePixels,
	}
	if im.ImageMask = operandBool(d, "IM", "ImageMask"); im.ImageMask {
		im.BitsPerComponent, im.Components = 1, 1
	} else if cs, ok := operandKey(d, "CS", "ColorSpace"); ok {
		im.setColorSpace(colorSpaceFromOperand(cs, resources))
	}
	if err := im.validateDimensions(); err != nil {
		return Image{}, err
	}
	if dec, ok := operandKey(d, "D", "Decode"); ok && dec.kind == opArr {
		for _, el := range dec.arr {
			if el.kind == opNum {
				im.decode = append(im.decode, el.num)
			}
		}
	}

	filters := operandNames(d, "F", "Filter")
	if len(filters) > maxInlineFilters {
		return Image{}, fmt.Errorf("inline image filter chain of %d exceeds limit", len(filters))
	}
	parms := operandList(d, "DP", "DecodeParms")
	data := src.data
	if _, err := l.readLimit(); err != nil {
		return Image{}, err
	}
	for i, name := range filters {
		var parm operand
		if i < len(parms) {
			parm = parms[i]
		}
		if codec, ok := filter.ImageCodec(name); ok {
			if i != len(filters)-1 {
				return Image{}, fmt.Errorf("image codec /%s is not the last filter", codec)
			}
			im.Filter = codec
			switch codec {
			case "CCITTFaxDecode":
				im.ccitt = ccittOptions(
					int64(operandNumber(parm.dict, "K")),
					int64(operandNumberOr(parm.dict, 1728, "Columns")),
					operandBool(parm.dict, "BlackIs1"),
					operandBool(parm.dict, "EncodedByteAlign"),
				)
				im.BitsPerComponent, im.Components = 1, 1
			case "JBIG2Decode", "JPXDecode":
				return Image{}, fmt.Errorf("inline image with /%s: %w", codec, errors.ErrUnsupported)
			}
			break
		}
		rd, err := filter.Apply(bytes.NewReader(data), name, filterParams(parm.dict))
		if err != nil {
			return Image{}, err
		}
		limit, err := l.readLimit()
		if err != nil {
			return Image{}, err
		}
		data, err = safeio.ReadAllGuardedLimitError(rd, limit)
		if rel, ok := rd.(filter.Releaser); ok {
			rel.Release()
		}
		if err != nil {
			return Image{}, err
		}
	}
	l.budget -= len(data)
	if im.Filter == "" {
		if err := im.validateSamples(); err != nil {
			return Image{}, err
		}
	}
	im.Data = data
	return im, nil
}

// maxInlineFilters matches the object layer's bound on a stream's filter
// chain: each filter is a pass over the data.
const maxInlineFilters = 8

func (im *Image) setColorSpace(cs colorSpace) {
	im.ColorSpace, im.Components, im.palette = cs.family, cs.ncomp, cs.palette
}

func (im Image) validateDimensions() error {
	if im.Width <= 0 || im.Height <= 0 {
		return fmt.Errorf("image has invalid dimensions %d×%d", im.Width, im.Height)
	}
	return nil
}

// validateSamples checks what unpacked samples need to be interpretable.
func (im Image) validateSamples() error {
	switch im.BitsPerComponent {
	case 1, 2, 4, 8, 16:
	default:
		return fmt.Errorf("image has invalid BitsPerComponent %d", im.BitsPerComponent)
	}
	if im.Components == 0 {
		return errors.New("image has no usable colour space")
	}
	return nil
}

func ccittOptions(k, columns int64, blackIs1, byteAlign bool) ccitt.Options {
	return ccitt.Options{
		K:                int(k),
		Columns:          int(columns),
		BlackIs1:         blackIs1,
		EncodedByteAlign: byteAlign,
	}
}

func filterParams(d map[string]operand) filter.Params {
	p := filter.Params{
		Predictor:        int(operandNumber(d, "Predictor")),
		Colors:           int(operandNumber(d, "Colors")),
		BitsPerComponent: int(operandNumber(d, "BitsPerComponent")),
		Columns:          int(operandNumber(d, "Columns")),
	}
	if ec, ok := d["EarlyChange"]; ok && ec.kind == opNum && ec.num == 0 {
		p.NoEarlyChange = true
	}
	return p
}

func boolOr(v object.Value) bool {
	b, _ := v.Bool()
	return b
}

func floatArray(v object.Value) []float64 {
	if v.Kind() != object.Array {
		return nil
	}
	out := make([]float64, 0, v.Len())
	for i := range v.Len() {
		f, _ := v.Index(i).Float64()
		out = append(out, f)
	}
	return out
}

// operandKey looks up the first of the given keys — an inline image's
// abbreviation and its full name — in an operand dictionary.
func operandKey(d map[string]operand, keys ...string) (operand, bool) {
	for _, key := range keys {
		if op, ok := d[key]; ok {
			return op, true
		}
	}
	return operand{}, false
}

func operandNumber(d map[string]operand, keys ...string) float64 {
	return operandNumberOr(d, 0, keys...)
}

func operandNumberOr(d map[string]operand, def float64, keys ...string) float64 {
	if op, ok := operandKey(d, keys...); ok && op.kind == opNum {
		return op.num
	}
	return def
}

func operandBool(d map[string]operand, keys ...string) bool {
	// The lexer records booleans without their value; an inline image
	// sets /IM or /ImageMask only to turn it on, so presence is enough.
	op, ok := operandKey(d, keys...)
	return ok && op.kind == opBool
}

// operandNames returns a name or array-of-names entry as a list.
func operandNames(d map[string]operand, keys ...string) []string {
	op, ok := operandKey(d, keys...)
	if !ok {
		return nil
	}
	switch op.kind {
	case opName:
		return []string{op.name}
	case opArr:
		names := make([]string, 0, len(op.arr))
		for _, el := range op.arr {
			if el.kind == opName {
				names = append(names, el.name)
			}
		}
		return names
	}
	return nil
}

// operandList returns a dictionary or array-of-dictionaries entry as a
// list aligned with operandNames.
func operandList(d map[string]operand, keys ...string) []operand {
	op, ok := operandKey(d, keys...)
	if !ok {
		return nil
	}
	switch op.kind {
	case opDict:
		return []operand{op}
	case opArr:
		return op.arr
	}
	return nil
}

// inlineImageLength returns the byte length of an unfiltered inline
// image's data, or -1 when it cannot be known from the dictionary alone
// (filtered data, or a colour space named in the resources).
func inlineImageLength(d map[string]operand) int {
	if _, filtered := operandKey(d, "F", "Filter"); filtered {
		return -1
	}
	w, h := operandNumber(d, "W", "Width"), operandNumber(d, "H", "Height")
	bpc, ncomp := operandNumber(d, "BPC", "BitsPerComponent"), 0.0
	if operandBool(d, "IM", "ImageMask") {
		bpc, ncomp = 1, 1
	} else if cs, ok := operandKey(d, "CS", "ColorSpace"); ok {
		switch cs.kind {
		case opName:
			if dev, ok := deviceColorSpace(cs.name); ok {
				ncomp = float64(dev.ncomp)
			}
		case opArr:
			if len(cs.arr) == 4 && cs.arr[0].kind == opName && (cs.arr[0].name == "I" || cs.arr[0].name == "Indexed") {
				ncomp = 1
			}
		}
	}
	if w <= 0 || h <= 0 || bpc <= 0 || ncomp <= 0 || w*ncomp*bpc > math.MaxInt32 || h > math.MaxInt32 {
		return -1
	}
	stride := (int(w)*int(ncomp)*int(bpc) + 7) / 8
	if stride > math.MaxInt32/int(h) {
		return -1
	}
	return stride * int(h)
}
