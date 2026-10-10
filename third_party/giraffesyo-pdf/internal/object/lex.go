package object

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

// tokenKind classifies a lexed token.
type tokenKind int

const (
	tokEOF tokenKind = iota
	tokInt
	tokReal
	tokString  // decoded literal or hex string
	tokName    // without leading slash
	tokKeyword // obj, endobj, stream, R, true, false, null, xref, trailer, ...
	tokDictOpen
	tokDictClose
	tokArrayOpen
	tokArrayClose
)

type token struct {
	kind tokenKind
	num  int64   // tokInt
	real float64 // tokReal
	str  string  // tokString, tokName, tokKeyword
}

// lexer tokenizes a byte slice of PDF object syntax. The whole region
// being parsed (an indirect object body, an xref section, an object
// stream) is held in memory, so the lexer works over a slice and cannot
// block or loop: the cursor only advances.
type lexer struct {
	buf []byte
	pos int
}

func newLexer(buf []byte) *lexer { return &lexer{buf: buf} }

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

func isDelim(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.buf) {
		c := l.buf[l.pos]
		if c == '%' {
			for l.pos < len(l.buf) && l.buf[l.pos] != '\n' && l.buf[l.pos] != '\r' {
				l.pos++
			}
			continue
		}
		if !isSpace(c) {
			return
		}
		l.pos++
	}
}

// next returns the next token. tokEOF is returned at end of input.
func (l *lexer) next() token {
	l.skipSpace()
	if l.pos >= len(l.buf) {
		return token{kind: tokEOF}
	}
	c := l.buf[l.pos]
	switch {
	case c == '<':
		if l.pos+1 < len(l.buf) && l.buf[l.pos+1] == '<' {
			l.pos += 2
			return token{kind: tokDictOpen}
		}
		return token{kind: tokString, str: l.readHexString()}
	case c == '>':
		if l.pos+1 < len(l.buf) && l.buf[l.pos+1] == '>' {
			l.pos += 2
			return token{kind: tokDictClose}
		}
		l.pos++ // stray '>'
		return l.next()
	case c == '[':
		l.pos++
		return token{kind: tokArrayOpen}
	case c == ']':
		l.pos++
		return token{kind: tokArrayClose}
	case c == '(':
		return token{kind: tokString, str: l.readLiteralString()}
	case c == '/':
		return token{kind: tokName, str: l.readName()}
	case c == '{' || c == '}':
		l.pos++ // procedure delimiters are not valid at object level; skip
		return l.next()
	case c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return l.readNumber()
	default:
		return l.readKeyword()
	}
}

func (l *lexer) readName() string {
	l.pos++ // consume '/'
	start := l.pos
	for l.pos < len(l.buf) && !isSpace(l.buf[l.pos]) && !isDelim(l.buf[l.pos]) {
		l.pos++
	}
	raw := l.buf[start:l.pos]
	if indexByte(raw, '#') < 0 {
		return string(raw)
	}
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] == '#' && i+2 < len(raw) {
			if hi, ok := hexVal(raw[i+1]); ok {
				if lo, ok2 := hexVal(raw[i+2]); ok2 {
					out = append(out, hi<<4|lo)
					i += 2
					continue
				}
			}
		}
		out = append(out, raw[i])
	}
	return string(out)
}

func (l *lexer) readLiteralString() string {
	l.pos++ // consume '('
	var out []byte
	depth := 1
	for l.pos < len(l.buf) {
		c := l.buf[l.pos]
		switch c {
		case '\\':
			l.pos++
			if l.pos >= len(l.buf) {
				return string(out)
			}
			e := l.buf[l.pos]
			switch e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '\n':
			case '\r':
				if l.pos+1 < len(l.buf) && l.buf[l.pos+1] == '\n' {
					l.pos++
				}
			default:
				if e >= '0' && e <= '7' {
					v := int(e - '0')
					for k := 0; k < 2 && l.pos+1 < len(l.buf); k++ {
						nx := l.buf[l.pos+1]
						if nx < '0' || nx > '7' {
							break
						}
						v = v*8 + int(nx-'0')
						l.pos++
					}
					out = append(out, byte(v))
				} else {
					out = append(out, e)
				}
			}
			l.pos++
		case '(':
			depth++
			out = append(out, c)
			l.pos++
		case ')':
			depth--
			l.pos++
			if depth == 0 {
				return string(out)
			}
			out = append(out, c)
		default:
			out = append(out, c)
			l.pos++
		}
	}
	return string(out)
}

func (l *lexer) readHexString() string {
	l.pos++ // consume '<'
	var digits []byte
	for l.pos < len(l.buf) && l.buf[l.pos] != '>' {
		if _, ok := hexVal(l.buf[l.pos]); ok {
			digits = append(digits, l.buf[l.pos])
		}
		l.pos++
	}
	if l.pos < len(l.buf) {
		l.pos++ // consume '>'
	}
	if len(digits)%2 == 1 {
		digits = append(digits, '0')
	}
	out := make([]byte, len(digits)/2)
	for i := range out {
		hi, _ := hexVal(digits[2*i])
		lo, _ := hexVal(digits[2*i+1])
		out[i] = hi<<4 | lo
	}
	return string(out)
}

func (l *lexer) readNumber() token {
	start := l.pos
	for l.pos < len(l.buf) && !isSpace(l.buf[l.pos]) && !isDelim(l.buf[l.pos]) {
		l.pos++
	}
	tok := string(l.buf[start:l.pos])
	if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
		return token{kind: tokInt, num: n}
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return token{kind: tokReal, real: f}
	}
	// Malformed number (e.g. "--", "1.2.3"): treat as a keyword so the
	// parser can decide, rather than aborting the whole object.
	return token{kind: tokKeyword, str: tok}
}

func (l *lexer) readKeyword() token {
	start := l.pos
	for l.pos < len(l.buf) && !isSpace(l.buf[l.pos]) && !isDelim(l.buf[l.pos]) {
		l.pos++
	}
	if l.pos == start {
		l.pos++ // never stall on an unexpected delimiter byte
		return l.next()
	}
	return token{kind: tokKeyword, str: string(l.buf[start:l.pos])}
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// errRecursion caps nested container depth.
var errRecursion = errors.New("pdf: object nesting too deep")

// parseObject parses one object at the lexer cursor. It resolves the
// "N G R" reference grammar and "N G obj" headers via two-token
// lookahead. The returned value's data is one of the concrete types
// listed in object.go (streams are handled by the caller, which knows the
// file offset).
func (p *objParser) parseObject(depth int) (any, error) {
	if depth > maxParseDepth {
		return nil, errRecursion
	}
	tok := p.lex.next()
	return p.parseFrom(tok, depth)
}

// objParser wraps a lexer with the small amount of state the reference
// grammar needs.
type objParser struct {
	lex *lexer
	r   *Reader
}

func (p *objParser) parseFrom(tok token, depth int) (any, error) {
	switch tok.kind {
	case tokEOF:
		return nil, io.EOF
	case tokInt:
		return p.parseIntOrRef(tok.num)
	case tokReal:
		return tok.real, nil
	case tokString:
		return tok.str, nil
	case tokName:
		return name(tok.str), nil
	case tokArrayOpen:
		return p.parseArray(depth)
	case tokDictOpen:
		return p.parseDict(depth)
	case tokKeyword:
		switch tok.str {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null":
			return nil, nil
		default:
			return nil, fmt.Errorf("pdf: unexpected keyword %q", tok.str)
		}
	default:
		return nil, errors.New("pdf: unexpected token")
	}
}

// parseIntOrRef disambiguates a plain integer from "N G R": on seeing an
// integer it peeks two tokens; int + "R" is a reference, otherwise the
// lookahead is unread.
func (p *objParser) parseIntOrRef(n int64) (any, error) {
	save := p.lex.pos
	t2 := p.lex.next()
	if t2.kind == tokInt {
		t3 := p.lex.next()
		if t3.kind == tokKeyword && t3.str == "R" {
			return ref{int(n), int(t2.num)}, nil
		}
	}
	p.lex.pos = save // not a reference: unread the lookahead
	return n, nil
}

func (p *objParser) parseArray(depth int) (any, error) {
	var arr []Value
	for {
		tok := p.lex.next()
		if tok.kind == tokArrayClose || tok.kind == tokEOF {
			return arr, nil
		}
		v, err := p.parseFrom(tok, depth+1)
		if err != nil {
			if errors.Is(err, errRecursion) {
				return nil, err
			}
			continue // skip a malformed element, keep the array
		}
		arr = append(arr, Value{r: p.r, data: v})
	}
}

func (p *objParser) parseDict(depth int) (any, error) {
	d := dict{}
	for {
		tok := p.lex.next()
		switch tok.kind {
		case tokDictClose, tokEOF:
			return d, nil
		case tokName:
			v, err := p.parseObject(depth + 1)
			if err != nil {
				if errors.Is(err, errRecursion) {
					return nil, err
				}
				return d, nil
			}
			d[tok.str] = v
		default:
			// A non-name key is malformed — stray tokens such as the "0 R"
			// in "/X 1 0 R 0 R". Skip the object and read on to the next
			// name, so the keys after it survive.
			if _, err := p.parseFrom(tok, depth+1); errors.Is(err, errRecursion) {
				return nil, err
			}
		}
	}
}
