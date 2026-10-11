package parser

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

const mathNS = "http://schemas.openxmlformats.org/officeDocument/2006/math"
const wordNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const compatNS = "http://schemas.openxmlformats.org/markup-compatibility/2006"

type xmlNode struct {
	name     xml.Name
	attrs    []xml.Attr
	text     string
	children []*xmlNode
}

func (n *xmlNode) attribute(key string) string {
	for _, a := range n.attrs {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}
func (n *xmlNode) child(key string) *xmlNode {
	for _, c := range n.children {
		if c.name.Local == key {
			return c
		}
	}
	return &xmlNode{}
}
func readTree(data []byte) (*xmlNode, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	root := &xmlNode{}
	stack := []*xmlNode{root}
	count := 0
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, failure("invalid_document", e)
		}
		switch v := t.(type) {
		case xml.StartElement:
			count++
			if len(stack) > 128 || count > 200000 {
				return nil, failure("resource_limit", errors.New("Office tree budget exceeded"))
			}
			n := &xmlNode{name: v.Name, attrs: v.Attr}
			p := stack[len(stack)-1]
			p.children = append(p.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].text += string(v)
		case xml.Directive:
			return nil, failure("invalid_document", errors.New("XML directives unsupported"))
		}
	}
	return root, nil
}
func mathLatex(n *xmlNode) (string, bool) {
	if n == nil {
		return "", true
	}
	join := func(nodes []*xmlNode) (string, bool) {
		var b strings.Builder
		ok := true
		for _, c := range nodes {
			s, v := mathLatex(c)
			b.WriteString(s)
			ok = ok && v
		}
		return b.String(), ok
	}
	part := func(key string) (string, bool) { return mathLatex(n.child(key)) }
	if n.name.Local == "" {
		return "", true
	}
	if n.name.Space != mathNS {
		return "", false
	}
	switch n.name.Local {
	case "t":
		return n.text, true
	case "oMath", "oMathPara", "r", "e", "num", "den", "sub", "sup", "deg", "mr":
		return join(n.children)
	case "f":
		a, x := part("num")
		b, y := part("den")
		return `\frac{` + a + `}{` + b + `}`, x && y
	case "rad":
		e, x := part("e")
		deg, y := part("deg")
		if deg != "" {
			return `\sqrt[` + deg + `]{` + e + `}`, x && y
		}
		return `\sqrt{` + e + `}`, x
	case "sSub", "sSup", "sSubSup", "sPre":
		e, x := part("e")
		sub, y := part("sub")
		sup, z := part("sup")
		v := e
		if n.name.Local == "sPre" {
			v = ""
		}
		if sub != "" {
			v += `_{` + sub + `}`
		}
		if sup != "" {
			v += `^{` + sup + `}`
		}
		if n.name.Local == "sPre" {
			v += e
		}
		return v, x && y && z
	case "m":
		var rows []string
		ok := true
		for _, r := range n.children {
			if r.name.Local != "mr" {
				continue
			}
			var cells []string
			for _, c := range r.children {
				v, x := mathLatex(c)
				cells = append(cells, v)
				ok = ok && x
			}
			rows = append(rows, strings.Join(cells, " & "))
		}
		return `\begin{matrix}` + strings.Join(rows, ` \\ `) + `\end{matrix}`, ok
	case "eqArr":
		var rows []string
		ok := true
		for _, c := range n.children {
			if c.name.Local == "e" {
				v, x := mathLatex(c)
				rows = append(rows, v)
				ok = ok && x
			}
		}
		return `\begin{aligned}` + strings.Join(rows, ` \\ `) + `\end{aligned}`, ok
	case "d":
		beg, end := "(", ")"
		p := n.child("dPr")
		if v := p.child("begChr").attribute("val"); v != "" {
			beg = v
		}
		if v := p.child("endChr").attribute("val"); v != "" {
			end = v
		}
		var parts []string
		ok := true
		for _, c := range n.children {
			if c.name.Local == "e" {
				v, x := mathLatex(c)
				parts = append(parts, v)
				ok = ok && x
			}
		}
		return `\left` + beg + strings.Join(parts, `\middle|`) + `\right` + end, ok
	case "nary":
		symbol := n.child("naryPr").child("chr").attribute("val")
		op := map[string]string{"∑": `\sum`, "∏": `\prod`, "∫": `\int`, "∮": `\oint`, "∬": `\iint`, "∭": `\iiint`}[symbol]
		if op == "" {
			op = `\int`
			if symbol != "" {
				return "", false
			}
		}
		a, x := part("sub")
		b, y := part("sup")
		e, z := part("e")
		if a != "" {
			op += `_{` + a + `}`
		}
		if b != "" {
			op += `^{` + b + `}`
		}
		return op + " " + e, x && y && z
	case "acc":
		e, x := part("e")
		op := map[string]string{"̂": `\hat`, "̅": `\bar`, "⃗": `\vec`, "̃": `\tilde`}[n.child("accPr").child("chr").attribute("val")]
		if op == "" {
			return "", false
		}
		return op + "{" + e + "}", x
	case "bar":
		e, x := part("e")
		return `\overline{` + e + `}`, x
	case "limLow", "limUpp":
		e, x := part("e")
		v, y := part("lim")
		op := `\underset`
		if n.name.Local == "limUpp" {
			op = `\overset`
		}
		return op + "{" + v + "}{" + e + "}", x && y
	}
	if strings.HasSuffix(n.name.Local, "Pr") {
		return "", true
	}
	return "", false
}
func (a *officeArchive) structuredText(name string, out *boundedText) error {
	data, e := a.load(name)
	if e != nil {
		return e
	}
	tree, e := readTree(data)
	if e != nil {
		return e
	}
	var render func(*xmlNode) string
	type tableRecord struct {
		text   string
		merges []TableMerge
	}
	var tables []tableRecord
	render = func(n *xmlNode) string {
		if a.ctx.Err() != nil {
			return ""
		}
		if n.name.Space == compatNS && n.name.Local == "AlternateContent" {
			for _, c := range n.children {
				if c.name.Local == "Choice" && (c.attribute("Requires") == "w" || c.attribute("Requires") == "m") {
					return render(c)
				}
			}
			return render(n.child("Fallback"))
		}
		if n.name.Space == mathNS && (n.name.Local == "oMath" || n.name.Local == "oMathPara") {
			v, ok := mathLatex(n)
			if !ok {
				return "\n[数学结构不确定：" + n.name.Local + "，需对照原件]\n"
			}
			return "\n$$" + v + "$$\n"
		}
		var b strings.Builder
		if n.name.Space == wordNS && n.name.Local == "tbl" {
			rowIndex := 0
			var merges []TableMerge
			for _, r := range n.children {
				if r.name.Local != "tr" {
					continue
				}
				b.WriteString("| ")
				column := 0
				for _, c := range r.children {
					if c.name.Local == "tc" {
						pr := c.child("tcPr")
						span, _ := strconv.Atoi(pr.child("gridSpan").attribute("val"))
						vertical := pr.child("vMerge").attribute("val")
						if pr.child("vMerge").name.Local != "" && vertical == "" {
							vertical = "continue"
						}
						if span > 1 || vertical != "" {
							merges = append(merges, TableMerge{Row: rowIndex, Column: column, Span: span, Vertical: vertical})
						}
						column++
						b.WriteString(strings.ReplaceAll(strings.TrimSpace(render(c)), "\n", "<br>"))
						b.WriteString(" | ")
					}
				}
				b.WriteString("\n")
				if rowIndex == 0 {
					b.WriteString("|")
					for i := 0; i < column; i++ {
						b.WriteString(" --- |")
					}
					b.WriteString("\n")
				}
				rowIndex++
			}
			text := b.String() + "\n"
			tables = append(tables, tableRecord{text, merges})
			return text
		}
		if n.name.Space == wordNS && n.name.Local == "p" {
			style := n.child("pPr").child("pStyle").attribute("val")
			level := a.wordStyles[style]
			if strings.HasPrefix(strings.ToLower(style), "heading") {
				level, _ = strconv.Atoi(strings.TrimPrefix(strings.ToLower(style), "heading"))
			}
			if level >= 1 && level <= 6 {
				b.WriteString(strings.Repeat("#", level) + " ")
			}
			if n.child("pPr").child("numPr").name.Local != "" {
				id := n.child("pPr").child("numPr").child("numId").attribute("val")
				if a.wordLists[id] == "decimal" {
					b.WriteString("1. ")
				} else {
					b.WriteString("- ")
				}
			}
		}
		switch n.name.Local {
		case "t":
			if n.name.Space == wordNS || n.name.Space == "" || strings.Contains(n.name.Space, "drawingml") {
				b.WriteString(n.text)
			}
		case "tab":
			b.WriteString("\t")
		case "br", "cr":
			b.WriteString("\n")
		}
		for _, c := range n.children {
			b.WriteString(render(c))
		}
		if n.name.Local == "p" {
			b.WriteString("\n")
		}
		return b.String()
	}
	text := render(tree)
	if e := a.ctx.Err(); e != nil {
		return e
	}
	start := out.b.Len()
	e = out.text(text)
	if strings.HasPrefix(name, "word/") && e == nil {
		local := StructureFromMarkdown(text, "local")
		if out.structure == nil {
			out.structure = &Structure{SchemaVersion: 1, ParserVersion: ParserVersion, Units: []Unit{{ID: "lines_1", Kind: "lines"}}, Coverage: StructureCoverage{CompleteWithinScope: true}}
		}
		target := &out.structure.Units[0]
		for _, u := range local.Units {
			for _, b := range u.Blocks {
				if b.Type == "table" {
					for _, record := range tables {
						if text[b.Range.Start:b.Range.End] == strings.TrimSuffix(record.text, "\n") {
							b.TableMerges = record.merges
							break
						}
					}
				}
				b.Range.Start += start
				b.Range.End += start
				for i := range b.TableRows {
					b.TableRows[i].Start += start
					b.TableRows[i].End += start
				}
				b.ID = "block_" + strconv.Itoa(len(target.Blocks)+1)
				target.Blocks = append(target.Blocks, b)
			}
		}
		out.structure.Coverage.Reasons = append(out.structure.Coverage.Reasons, local.Coverage.Reasons...)
	}
	return e
}

func (a *officeArchive) loadWordProperties() error {
	a.wordStyles = map[string]int{}
	a.wordLists = map[string]string{}
	var walk func(*xmlNode, func(*xmlNode))
	walk = func(n *xmlNode, visit func(*xmlNode)) {
		visit(n)
		for _, c := range n.children {
			walk(c, visit)
		}
	}
	if a.entries["word/styles.xml"] != nil {
		data, e := a.load("word/styles.xml")
		if e != nil {
			return e
		}
		root, e := readTree(data)
		if e != nil {
			return e
		}
		parents := map[string]string{}
		walk(root, func(n *xmlNode) {
			if n.name.Space != wordNS || n.name.Local != "style" {
				return
			}
			id := n.attribute("styleId")
			parents[id] = n.child("basedOn").attribute("val")
			level := -1
			if val := n.child("pPr").child("outlineLvl").attribute("val"); val != "" {
				level, _ = strconv.Atoi(val)
			}
			if level >= 0 && level < 6 {
				a.wordStyles[id] = level + 1
			}
			name := strings.ToLower(n.child("name").attribute("val"))
			if strings.HasPrefix(name, "heading ") {
				v, _ := strconv.Atoi(strings.TrimPrefix(name, "heading "))
				if v >= 1 && v <= 6 {
					a.wordStyles[id] = v
				}
			}
		})
		for i := 0; i < 8; i++ {
			for id, parent := range parents {
				if a.wordStyles[id] == 0 {
					a.wordStyles[id] = a.wordStyles[parent]
				}
			}
		}
	}
	if a.entries["word/numbering.xml"] != nil {
		data, e := a.load("word/numbering.xml")
		if e != nil {
			return e
		}
		root, e := readTree(data)
		if e != nil {
			return e
		}
		abstract := map[string]string{}
		walk(root, func(n *xmlNode) {
			if n.name.Space == wordNS && n.name.Local == "abstractNum" {
				abstract[n.attribute("abstractNumId")] = n.child("lvl").child("numFmt").attribute("val")
			}
		})
		walk(root, func(n *xmlNode) {
			if n.name.Space == wordNS && n.name.Local == "num" {
				a.wordLists[n.attribute("numId")] = abstract[n.child("abstractNumId").attribute("val")]
			}
		})
	}
	return a.ctx.Err()
}
