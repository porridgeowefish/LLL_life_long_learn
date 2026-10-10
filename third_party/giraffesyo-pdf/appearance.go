package pdf

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/giraffesyo/pdf/internal/object"
)

// Annotation flags (ISO 32000-1 §12.5.3) that keep an annotation off the
// page a viewer shows.
const (
	annotHidden = 1 << 1
	annotNoView = 1 << 5
)

// walkAppearances extracts the text an annotation's normal appearance
// paints — a filled form field's value, a free-text comment, a stamp —
// after the page's own content, the order a viewer paints them in.
// Annotations a viewer does not show (Hidden, NoView) are skipped.
//
// A text or choice field's value is drawn from the field itself when the
// form asks viewers to regenerate appearances (/NeedAppearances) or the
// field has none, as viewers and poppler do; any other annotation without
// a normal appearance stream paints nothing.
func (w *walker) walkAppearances(pageNode object.Value, pageResources object.Value) error {
	annots := pageNode.Key("Annots")
	if annots.Kind() != object.Array {
		return nil
	}
	needAppearances := -1 // unknown until a field asks
	for i := range annots.Len() {
		annot := annots.Index(i)
		appearance := normalAppearance(annot) // most links have none
		isField := annot.Key("Subtype").Name() == "Widget"
		if appearance.Kind() != object.Stream && !isField {
			continue
		}
		if flags, _ := annot.Key("F").Int64(); flags&(annotHidden|annotNoView) != 0 {
			continue
		}
		if isField {
			if needAppearances < 0 {
				needAppearances = 0
				if b, _ := pageNode.Trailer().Key("Root").Key("AcroForm").Key("NeedAppearances").Bool(); b {
					needAppearances = 1
				}
			}
			if (needAppearances == 1 || appearance.Kind() != object.Stream) && w.drawFieldValue(annot) {
				continue
			}
		}
		if appearance.Kind() != object.Stream {
			continue
		}
		ctm, ok := appearanceMatrix(appearance, rectFromValue(annot.Key("Rect")))
		if !ok {
			continue
		}
		// A form field's appearance may rely on the form's default
		// resources for its font, as viewers allow.
		resources := appearance.Key("Resources")
		if resources.Kind() != object.Dict {
			resources = pageNode.Trailer().Key("Root").Key("AcroForm").Key("DR")
			if resources.Kind() != object.Dict {
				resources = pageResources
			}
		}
		if w.depth >= w.limits.MaxFormDepth {
			return w.warning(WarningWorkLimit, errors.New("form XObject nesting exceeds limit"))
		}
		w.depth++
		err := w.walkStream(appearance, resources, gstate{ctm: ctm, hscale: 1})
		w.depth--
		if err != nil {
			return err
		}
	}
	return nil
}

// normalAppearance returns the annotation's normal appearance stream: /AP
// /N itself, or the state /AS selects when /N holds one per state, as a
// checkbox's does.
func normalAppearance(annot object.Value) object.Value {
	normal := annot.Key("AP").Key("N")
	if normal.Kind() == object.Dict {
		return normal.Key(annot.Key("AS").Name())
	}
	return normal
}

// appearanceMatrix maps an appearance stream's form space onto the
// annotation rectangle (ISO 32000-1 §12.5.5): the form's bounding box,
// transformed by its matrix, is scaled and translated to fill rect.
func appearanceMatrix(appearance object.Value, rect Rect) (matrix, bool) {
	bbox := rectFromValue(appearance.Key("BBox"))
	form := identity
	if m := appearance.Key("Matrix"); m.Kind() == object.Array && m.Len() == 6 {
		for i := range form {
			form[i], _ = m.Index(i).Float64()
		}
	}
	box := Rect{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
	for _, p := range [4]Point{{bbox.MinX, bbox.MinY}, {bbox.MaxX, bbox.MinY}, {bbox.MinX, bbox.MaxY}, {bbox.MaxX, bbox.MaxY}} {
		x := p.X*form[0] + p.Y*form[2] + form[4]
		y := p.X*form[1] + p.Y*form[3] + form[5]
		box.MinX, box.MaxX = min(box.MinX, x), max(box.MaxX, x)
		box.MinY, box.MaxY = min(box.MinY, y), max(box.MaxY, y)
	}
	width, height := box.MaxX-box.MinX, box.MaxY-box.MinY
	if !(width > 0 && height > 0) || rect.MaxX <= rect.MinX || rect.MaxY <= rect.MinY {
		return matrix{}, false
	}
	fit := matrix{
		(rect.MaxX - rect.MinX) / width, 0,
		0, (rect.MaxY - rect.MinY) / height,
		rect.MinX - box.MinX*(rect.MaxX-rect.MinX)/width,
		rect.MinY - box.MinY*(rect.MaxY-rect.MinY)/height,
	}
	return mul(form, fit), true
}

// Field flags (ISO 32000-1 §12.7.4) that change how a value is drawn.
const (
	fieldMultiline = 1 << 12
	fieldCombo     = 1 << 17
)

// maxFieldRunes bounds the text drawn for one field.
const maxFieldRunes = 4096

// drawFieldValue adds the glyphs a viewer draws for a text or choice
// field's value — a text field's value, a combo box's selection, a list
// box's options — laid out in the widget rectangle at the size its /DA
// names, and reports whether the field was one it draws. Advances are
// estimated at half the font size: the value is text for extraction, not
// for rendering, and its string need not be encodable in the field font.
func (w *walker) drawFieldValue(annot object.Value) bool {
	var lines []string
	flags, _ := object.Inherited(annot, "Ff").Int64()
	switch object.Inherited(annot, "FT").Name() {
	case "Tx":
		value := objectText(object.Inherited(annot, "V"))
		if flags&fieldMultiline != 0 {
			lines = strings.Split(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(value), "\n")
		} else {
			lines = []string{value}
		}
	case "Ch":
		if flags&fieldCombo != 0 {
			value := object.Inherited(annot, "V")
			if value.Kind() == object.Array && value.Len() > 0 {
				value = value.Index(0)
			}
			lines = []string{objectText(value)}
			break
		}
		options := object.Inherited(annot, "Opt")
		for i := range options.Len() {
			option := options.Index(i)
			if option.Kind() == object.Array { // [export display]
				option = option.Index(1)
			}
			lines = append(lines, objectText(option))
		}
	default:
		return false
	}
	rect := rectFromValue(annot.Key("Rect"))
	if rect.MaxX <= rect.MinX || rect.MaxY <= rect.MinY {
		return true
	}
	size := fieldFontSize(objectText(object.Inherited(annot, "DA")))
	height := rect.MaxY - rect.MinY
	if size <= 0 { // auto-sized
		size = min(12, max(1, (height-4)*0.8))
		if len(lines) > 1 {
			size = min(size, 12)
		}
	}
	quadding, _ := object.Inherited(annot, "Q").Int64()
	advance := size / 2
	runes := 0
	for i, line := range lines {
		// A single line sits centred in the box; several run down from the
		// top, as a list box or multiline field shows them.
		baseline := rect.MinY + (height-size)/2 + 0.22*size
		if len(lines) > 1 {
			baseline = rect.MaxY - 2 - float64(i+1)*size*1.15 + 0.22*size
			if baseline < rect.MinY {
				break
			}
		}
		width := float64(len([]rune(line))) * advance
		x := rect.MinX + 2
		switch quadding {
		case 1:
			x = rect.MinX + (rect.MaxX-rect.MinX-width)/2
		case 2:
			x = rect.MaxX - 2 - width
		}
		for _, r := range line {
			if runes++; runes > maxFieldRunes {
				return true
			}
			if r != ' ' {
				w.glyphs.add(Glyph{
					Text: string(r), X: x, Y: baseline, Advance: advance, Size: size,
					Direction: Point{X: 1}, Ascent: Point{Y: size},
				})
			}
			x += advance
		}
	}
	return true
}

// fieldFontSize returns the font size a field's default appearance string
// sets with Tf ("/Helv 12 Tf 0 g"), or 0 for auto-size or none.
func fieldFontSize(da string) float64 {
	fields := strings.Fields(da)
	for i := len(fields) - 1; i >= 1; i-- {
		if fields[i] == "Tf" {
			size, err := strconv.ParseFloat(fields[i-1], 64)
			if err == nil && size > 0 && size < 1000 {
				return size
			}
			return 0
		}
	}
	return 0
}
