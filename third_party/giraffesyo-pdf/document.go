package pdf

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/giraffesyo/pdf/internal/object"
)

func extractMetadata(r *object.Reader, streamLimit int) (Metadata, error) {
	info := r.Trailer().Key("Info")
	metadata := Metadata{}
	known := map[string]*string{
		"Title":        &metadata.Title,
		"Author":       &metadata.Author,
		"Subject":      &metadata.Subject,
		"Keywords":     &metadata.Keywords,
		"Creator":      &metadata.Creator,
		"Producer":     &metadata.Producer,
		"CreationDate": &metadata.CreationDate,
		"ModDate":      &metadata.ModifiedDate,
		"Trapped":      &metadata.Trapped,
	}
	if info.Kind() == object.Dict {
		for _, key := range info.Keys() {
			text := objectText(info.Key(key))
			if target := known[key]; target != nil {
				*target = text
				continue
			}
			if text != "" {
				if metadata.Custom == nil {
					metadata.Custom = map[string]string{}
				}
				metadata.Custom[key] = text
			}
		}
	}
	xmp := r.Trailer().Key("Root").Key("Metadata")
	if xmp.Kind() == object.Stream {
		data, err := readStreamBoundedLimitError(xmp, streamLimit)
		metadata.XMP = data
		if err != nil {
			return metadata, fmt.Errorf("read XMP metadata: %w", err)
		}
	}
	return metadata, nil
}

func extractOutlines(r *object.Reader, pages map[int]int) ([]Outline, error) {
	root := r.Trailer().Key("Root").Key("Outlines").Key("First")
	if root.IsNull() {
		return nil, nil
	}
	seen := map[int]bool{}
	nodes := 0
	var walkList func(object.Value, int) ([]Outline, error)
	walkList = func(item object.Value, depth int) ([]Outline, error) {
		if depth > 64 {
			return nil, errors.New("outline tree too deep")
		}
		var out []Outline
		for !item.IsNull() {
			nodes++
			if nodes > 50000 {
				return out, errors.New("outline tree exceeds node limit")
			}
			if number, ok := item.ObjectNumber(); ok {
				if seen[number] {
					return out, errors.New("cyclic outline tree")
				}
				seen[number] = true
			}
			outline := Outline{Title: objectText(item.Key("Title"))}
			destination := item.Key("Dest")
			if destination.IsNull() {
				destination = item.Key("A").Key("D")
			}
			outline.Destination = parseDestination(destination, pages)
			children, err := walkList(item.Key("First"), depth+1)
			if err != nil {
				return out, err
			}
			outline.Children = children
			out = append(out, outline)
			item = item.Key("Next")
		}
		return out, nil
	}
	return walkList(root, 0)
}

func extractAnnotations(page object.Value, pages map[int]int) ([]Annotation, error) {
	array := page.Key("Annots")
	if array.IsNull() {
		return nil, nil
	}
	if array.Kind() != object.Array {
		return nil, errors.New("page Annots is not an array")
	}
	out := make([]Annotation, 0, array.Len())
	for i := range array.Len() {
		value := array.Index(i)
		if value.Kind() != object.Dict {
			continue
		}
		annotation := Annotation{
			Subtype:  value.Key("Subtype").Name(),
			Rect:     rectFromValue(value.Key("Rect")),
			Contents: objectText(value.Key("Contents")),
			Title:    objectText(value.Key("T")),
		}
		action := value.Key("A")
		if action.Key("S").Name() == "URI" {
			annotation.URL = objectText(action.Key("URI"))
		}
		destination := value.Key("Dest")
		if destination.IsNull() {
			destination = action.Key("D")
		}
		annotation.Destination = parseDestination(destination, pages)
		out = append(out, annotation)
	}
	return out, nil
}

func extractFormFields(r *object.Reader, pages map[int]int) ([]FormField, error) {
	fields := r.Trailer().Key("Root").Key("AcroForm").Key("Fields")
	if fields.IsNull() {
		return nil, nil
	}
	if fields.Kind() != object.Array {
		return nil, errors.New("AcroForm Fields is not an array")
	}
	var out []FormField
	seen := map[int]bool{}
	nodes := 0
	var walk func(object.Value, string, string, string, int) error
	walk = func(field object.Value, parentName, inheritedType, inheritedValue string, inheritedPage int) error {
		nodes++
		if nodes > 50000 {
			return errors.New("AcroForm field tree exceeds node limit")
		}
		if number, ok := field.ObjectNumber(); ok {
			if seen[number] {
				return errors.New("cyclic AcroForm field tree")
			}
			seen[number] = true
		}
		name := objectText(field.Key("T"))
		if parentName != "" && name != "" {
			name = parentName + "." + name
		} else if name == "" {
			name = parentName
		}
		fieldType := field.Key("FT").Name()
		if fieldType == "" {
			fieldType = inheritedType
		}
		value := objectText(field.Key("V"))
		if value == "" {
			value = inheritedValue
		}
		page := inheritedPage
		if number, ok := field.Key("P").ObjectNumber(); ok {
			page = pages[number]
		}

		kids := field.Key("Kids")
		hasFieldKids := false
		if kids.Kind() == object.Array {
			for i := range kids.Len() {
				kid := kids.Index(i)
				if kid.Key("Subtype").Name() == "Widget" && kid.Key("FT").IsNull() && kid.Key("T").IsNull() {
					if number, ok := kid.Key("P").ObjectNumber(); ok && page == 0 {
						page = pages[number]
					}
					continue
				}
				hasFieldKids = true
				if err := walk(kid, name, fieldType, value, page); err != nil {
					return err
				}
			}
		}
		if !hasFieldKids && (name != "" || fieldType != "" || value != "") {
			out = append(out, FormField{
				Name:          name,
				AlternateName: objectText(field.Key("TU")),
				Type:          fieldType,
				Value:         value,
				Page:          page,
			})
		}
		return nil
	}
	for i := range fields.Len() {
		if err := walk(fields.Index(i), "", "", "", 0); err != nil {
			return out, err
		}
	}
	return out, nil
}

func parseDestination(value object.Value, pages map[int]int) Destination {
	switch value.Kind() {
	case object.Name:
		return Destination{Name: value.Name()}
	case object.String:
		return Destination{Name: objectText(value)}
	case object.Array:
		destination := Destination{}
		if number, ok := value.Index(0).ObjectNumber(); ok {
			destination.Page = pages[number]
		}
		kind := value.Index(1).Name()
		switch kind {
		case "XYZ":
			destination.X, _ = value.Index(2).Float64()
			destination.Y, _ = value.Index(3).Float64()
			destination.Zoom, _ = value.Index(4).Float64()
		case "FitH", "FitBH":
			destination.Y, _ = value.Index(2).Float64()
		case "FitV", "FitBV":
			destination.X, _ = value.Index(2).Float64()
		case "FitR":
			destination.X, _ = value.Index(2).Float64()
			destination.Y, _ = value.Index(3).Float64()
		}
		return destination
	default:
		return Destination{}
	}
}

func objectText(value object.Value) string {
	switch value.Kind() {
	case object.String:
		return decodeTextString([]byte(value.RawString()))
	case object.Name:
		return value.Name()
	case object.Integer:
		number, _ := value.Int64()
		return strconv.FormatInt(number, 10)
	case object.Real:
		number, _ := value.Float64()
		return strconv.FormatFloat(number, 'g', -1, 64)
	case object.Array:
		parts := make([]string, 0, value.Len())
		for i := range value.Len() {
			if text := objectText(value.Index(i)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}
