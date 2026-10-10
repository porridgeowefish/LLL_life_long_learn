package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	maxZIPEntries       = 10000
	maxXMLBytes         = 16 << 20
	maxSelectedXMLBytes = 64 << 20
)

type officeArchive struct {
	ctx       context.Context
	entries   map[string]*zip.File
	readBytes int64
}

func selectedXML(name, ext string) bool {
	switch ext {
	case ".docx":
		return name == "word/document.xml" || name == "word/footnotes.xml" || name == "word/endnotes.xml" || strings.HasPrefix(name, "word/header") && strings.HasSuffix(name, ".xml") || strings.HasPrefix(name, "word/footer") && strings.HasSuffix(name, ".xml")
	case ".pptx":
		return name == "ppt/presentation.xml" || name == "ppt/_rels/presentation.xml.rels" || strings.HasPrefix(name, "ppt/slides/") && (strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels")) || strings.HasPrefix(name, "ppt/notesSlides/") && strings.HasSuffix(name, ".xml")
	case ".xlsx":
		return name == "xl/workbook.xml" || name == "xl/_rels/workbook.xml.rels" || name == "xl/sharedStrings.xml" || strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml")
	}
	return false
}

func extractOffice(ctx context.Context, f io.ReaderAt, size int64, ext string, out *boundedText) error {
	var signature [8]byte
	_, _ = f.ReadAt(signature[:], 0)
	if bytes.Equal(signature[:], []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
		return failure("encrypted_document", errors.New("encrypted Office containers require an unencrypted OOXML copy"))
	}
	z, err := zip.NewReader(contextReaderAt{ctx, f}, size)
	if err != nil {
		return failure("invalid_document", err)
	}
	if len(z.File) > maxZIPEntries {
		return failure("resource_limit", errors.New("Office container exceeds 10000 ZIP entries"))
	}
	a := &officeArchive{ctx: ctx, entries: map[string]*zip.File{}}
	var total uint64
	for _, entry := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := entry.Name
		if strings.Contains(name, "\\") || path.Clean(name) != strings.TrimSuffix(name, "/") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") {
			return failure("invalid_document", errors.New("invalid ZIP part name"))
		}
		if _, duplicate := a.entries[name]; duplicate {
			return failure("invalid_document", errors.New("duplicate ZIP part"))
		}
		a.entries[name] = entry
		if !selectedXML(name, ext) {
			continue
		}
		if entry.UncompressedSize64 > maxXMLBytes {
			return failure("resource_limit", errors.New("Office XML part exceeds 16 MiB"))
		}
		total += entry.UncompressedSize64
		if total > maxSelectedXMLBytes {
			return failure("resource_limit", errors.New("Office selected XML exceeds 64 MiB"))
		}
	}
	switch ext {
	case ".docx":
		return a.word(out)
	case ".pptx":
		return a.presentation(out)
	case ".xlsx":
		return a.workbook(out)
	}
	return failure("unsupported_format", errors.New("unknown Office format"))
}
func (a *officeArchive) load(name string) ([]byte, error) {
	if err := a.ctx.Err(); err != nil {
		return nil, err
	}
	entry := a.entries[name]
	if entry == nil {
		return nil, failure("invalid_document", fmt.Errorf("missing Office part %s", name))
	}
	if entry.UncompressedSize64 > maxXMLBytes {
		return nil, failure("resource_limit", errors.New("Office XML part exceeds 16 MiB"))
	}
	r, err := entry.Open()
	if err != nil {
		return nil, failure("invalid_document", err)
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(contextReader{a.ctx, r}, maxXMLBytes+1))
	a.readBytes += int64(len(data))
	if len(data) > maxXMLBytes || a.readBytes > maxSelectedXMLBytes {
		return nil, failure("resource_limit", errors.New("Office expanded XML budget exceeded"))
	}
	if err != nil {
		return nil, failure("invalid_document", err)
	}
	return data, nil
}
func (a *officeArchive) walk(name string, visit func(xml.Token) error) error {
	data, err := a.load(name)
	if err != nil {
		return err
	}
	d := xml.NewDecoder(contextReader{a.ctx, bytes.NewReader(data)})
	depth := 0
	for {
		token, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return failure("invalid_document", err)
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
			if depth > 128 {
				return failure("resource_limit", errors.New("Office XML nesting exceeds 128"))
			}
		case xml.EndElement:
			depth--
		}
		// No custom entities, DTD processing or external resource resolution.
		if _, directive := token.(xml.Directive); directive {
			return failure("invalid_document", errors.New("Office XML directives are not supported"))
		}
		if err := visit(token); err != nil {
			return err
		}
	}
}

func (a *officeArchive) text(name string, out *boundedText) error {
	inText := 0
	return a.walk(name, func(token xml.Token) error {
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText++
			case "tab":
				return out.text("\t")
			case "br", "cr":
				return out.text("\n")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText--
			case "p", "tr":
				return out.text("\n")
			case "tc":
				return out.text("\t")
			}
		case xml.CharData:
			if inText > 0 {
				return out.text(string(t))
			}
		}
		return nil
	})
}
func (a *officeArchive) word(out *boundedText) error {
	if err := a.text("word/document.xml", out); err != nil {
		return err
	}
	var extras []string
	for name := range a.entries {
		if selectedXML(name, ".docx") && name != "word/document.xml" {
			extras = append(extras, name)
		}
	}
	sort.Strings(extras)
	for _, name := range extras {
		if err := out.text("\n"); err != nil {
			return err
		}
		if err := a.text(name, out); err != nil {
			return err
		}
	}
	return nil
}

type relationship struct{ target, kind string }

func attr(t xml.StartElement, name string) string {
	for _, at := range t.Attr {
		if at.Name.Local == name {
			return at.Value
		}
	}
	return ""
}
func (a *officeArchive) relationships(owner string) (map[string]relationship, error) {
	relName := path.Join(path.Dir(owner), "_rels", path.Base(owner)+".rels")
	result := map[string]relationship{}
	if a.entries[relName] == nil {
		return result, nil
	}
	err := a.walk(relName, func(token xml.Token) error {
		t, ok := token.(xml.StartElement)
		if !ok || t.Name.Local != "Relationship" {
			return nil
		}
		if strings.EqualFold(attr(t, "TargetMode"), "External") {
			return nil
		}
		target := attr(t, "Target")
		if strings.Contains(target, "\\") || strings.Contains(target, ":") {
			return failure("invalid_document", errors.New("invalid Office relationship target"))
		}
		if strings.HasPrefix(target, "/") {
			target = strings.TrimPrefix(target, "/")
		} else {
			target = path.Join(path.Dir(owner), target)
		}
		target = path.Clean(target)
		prefix := strings.SplitN(owner, "/", 2)[0] + "/"
		if !strings.HasPrefix(target, prefix) {
			return failure("invalid_document", errors.New("Office relationship escapes its package"))
		}
		result[attr(t, "Id")] = relationship{target, attr(t, "Type")}
		return nil
	})
	return result, err
}
func (a *officeArchive) presentation(out *boundedText) error {
	relations, err := a.relationships("ppt/presentation.xml")
	if err != nil {
		return err
	}
	var slideNames []string
	err = a.walk("ppt/presentation.xml", func(token xml.Token) error {
		t, ok := token.(xml.StartElement)
		if !ok || t.Name.Local != "sldId" {
			return nil
		}
		for _, at := range t.Attr {
			if at.Name.Local == "id" && strings.Contains(at.Name.Space, "relationships") {
				r, ok := relations[at.Value]
				if !ok || !strings.HasSuffix(r.kind, "/slide") {
					return failure("invalid_document", errors.New("missing slide relationship"))
				}
				if !strings.HasPrefix(r.target, "ppt/slides/") || !strings.HasSuffix(r.target, ".xml") {
					return failure("invalid_document", errors.New("invalid slide part"))
				}
				slideNames = append(slideNames, r.target)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for index, name := range slideNames {
		content := &boundedText{ctx: a.ctx}
		if err := a.text(name, content); err != nil && !errors.Is(err, errOutputFull) {
			return err
		}
		if strings.TrimSpace(content.b.String()) != "" {
			if err := out.text(fmt.Sprintf("## 第 %d 张幻灯片\n\n", index+1)); err != nil {
				return err
			}
			if err := out.text(content.String() + "\n"); err != nil {
				return err
			}
		}
		rels, err := a.relationships(name)
		if err != nil {
			return err
		}
		// Sort to keep corrupted files with multiple notes relationships deterministic.
		var notes []string
		for _, r := range rels {
			if strings.HasSuffix(r.kind, "/notesSlide") {
				if !strings.HasPrefix(r.target, "ppt/notesSlides/") {
					return failure("invalid_document", errors.New("invalid notes part"))
				}
				notes = append(notes, r.target)
			}
		}
		sort.Strings(notes)
		for _, note := range notes {
			content := &boundedText{ctx: a.ctx}
			if err := a.text(note, content); err != nil && !errors.Is(err, errOutputFull) {
				return err
			}
			if strings.TrimSpace(content.b.String()) != "" {
				if err := out.text("\n备注：\n\n" + content.String() + "\n"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *officeArchive) sharedStrings(name string) ([]string, error) {
	if a.entries[name] == nil {
		return nil, nil
	}
	var result []string
	var current strings.Builder
	inString, inText := false, false
	err := a.walk(name, func(token xml.Token) error {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				inString = true
				current.Reset()
			}
			if inString && t.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
			if t.Name.Local == "si" {
				result = append(result, current.String())
				inString = false
			}
		case xml.CharData:
			if inString && inText {
				current.Write(t)
			}
		}
		return nil
	})
	return result, err
}
func (a *officeArchive) workbook(out *boundedText) error {
	relations, err := a.relationships("xl/workbook.xml")
	if err != nil {
		return err
	}
	type sheet struct{ name, target string }
	var sheets []sheet
	err = a.walk("xl/workbook.xml", func(token xml.Token) error {
		t, ok := token.(xml.StartElement)
		if !ok || t.Name.Local != "sheet" {
			return nil
		}
		r, ok := relations[attr(t, "id")]
		if !ok || !strings.HasSuffix(r.kind, "/worksheet") || !strings.HasPrefix(r.target, "xl/worksheets/") {
			return failure("invalid_document", errors.New("missing worksheet relationship"))
		}
		sheets = append(sheets, sheet{attr(t, "name"), r.target})
		return nil
	})
	if err != nil {
		return err
	}
	sharedName := "xl/sharedStrings.xml"
	for _, r := range relations {
		if strings.HasSuffix(r.kind, "/sharedStrings") {
			sharedName = r.target
		}
	}
	shared, err := a.sharedStrings(sharedName)
	if err != nil {
		return err
	}
	for _, sheet := range sheets {
		if err := a.sheet(sheet.name, sheet.target, shared, out); err != nil {
			return err
		}
	}
	return nil
}

var cellReference = regexp.MustCompile(`^[A-Z]{1,3}[1-9][0-9]{0,6}$`)

func (a *officeArchive) sheet(name, part string, shared []string, out *boundedText) error {
	var value, formula, inline strings.Builder
	var ref, kind, field string
	inCell, header := false, false
	emit := func() error {
		text := value.String()
		switch kind {
		case "s":
			index, err := strconv.Atoi(text)
			if err != nil || index < 0 || index >= len(shared) {
				return failure("invalid_document", errors.New("invalid shared string index"))
			}
			text = shared[index]
		case "inlineStr":
			text = inline.String()
		case "b":
			if text == "1" {
				text = "TRUE"
			} else if text == "0" {
				text = "FALSE"
			}
		}
		if formula.Len() > 0 {
			if text == "" {
				text = "=" + formula.String()
			} else {
				text = "=" + formula.String() + " (缓存值：" + text + ")"
			}
		}
		if text == "" {
			return nil
		}
		if !header {
			if err := out.text("## 工作表：" + strings.ReplaceAll(name, "\n", " ") + "\n\n| 单元格 | 内容 |\n| --- | --- |\n"); err != nil {
				return err
			}
			header = true
		}
		return tableRow(out, []string{ref, text})
	}
	return a.walk(part, func(token xml.Token) error {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "c" {
				if inCell {
					return failure("invalid_document", errors.New("nested worksheet cell"))
				}
				inCell = true
				ref = attr(t, "r")
				kind = attr(t, "t")
				value.Reset()
				formula.Reset()
				inline.Reset()
				if ref != "" && !cellReference.MatchString(ref) {
					return failure("invalid_document", errors.New("invalid worksheet cell reference"))
				}
			}
			if inCell && (t.Name.Local == "v" || t.Name.Local == "f" || t.Name.Local == "t") {
				field = t.Name.Local
			}
		case xml.EndElement:
			if t.Name.Local == "c" {
				inCell = false
				field = ""
				return emit()
			}
			if t.Name.Local == field {
				field = ""
			}
		case xml.CharData:
			if inCell {
				switch field {
				case "v":
					value.Write(t)
				case "f":
					formula.Write(t)
				case "t":
					inline.Write(t)
				}
			}
		}
		return nil
	})
}
