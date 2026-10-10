package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/giraffesyo/pdf/internal/object"
	"github.com/giraffesyo/pdf/internal/safeio"
)

// This file lexes and executes raw content-stream bytes. The library's own
// Interpret is not used because it interprets each /Contents array element
// as an isolated stream (an array token split across elements spins its
// lexer forever at EOF) and it cannot skip inline-image (BI..EI) binary
// data. Lexing concatenated in-memory bytes cannot loop: the cursor only
// advances.

type opKind uint8

const (
	opNum opKind = iota
	opStr
	opName
	opBool
	opNull
	opArr
	opDict
	opProc
)

type operand struct {
	kind opKind
	num  float64
	str  []byte
	name string
	arr  []operand
	dict map[string]operand
}

const (
	maxOperandNest = 64
	maxStackDepth  = 4096
)

// readStreamBounded reads a stream's decoded bytes with a total-size cap
// and a stall guard (a filter reader that keeps returning (0, nil) would
// otherwise loop io.ReadAll forever). It returns any decoded prefix together
// with a non-nil error when decoding fails or the limit is exceeded.
func readStreamBoundedLimitError(v object.Value, limit int) ([]byte, error) {
	return appendStreamBoundedLimitError(nil, v, limit)
}

// appendStreamBoundedLimitError is readStreamBoundedLimitError appending
// to buf, reusing its capacity.
func appendStreamBoundedLimitError(buf []byte, v object.Value, limit int) ([]byte, error) {
	if v.Kind() != object.Stream {
		return buf, errors.New("content value is not a stream")
	}
	rc, err := v.Reader()
	if err != nil {
		return buf, err
	}
	defer func() { _ = rc.Close() }() // read-only handle
	return safeio.AppendAllGuardedLimitError(buf, rc, limit)
}

// contentBytes returns the page's full content: /Contents may be a single
// stream or an array of streams that form one logical stream.
func contentBytesLimit(contents object.Value, limit int) []byte {
	data, _ := contentBytesLimitError(contents, limit)
	return data
}

func contentBytesLimitError(contents object.Value, limit int) ([]byte, []error) {
	return contentBytesLimitErrorInto(nil, contents, limit)
}

// contentBytesLimitErrorInto is contentBytesLimitError decoding into buf's
// capacity (from its start), for a caller that walks page after page.
func contentBytesLimitErrorInto(buf []byte, contents object.Value, limit int) ([]byte, []error) {
	if limit <= 0 {
		return nil, nil
	}
	data := buf[:0]
	if contents.Kind() == object.Array {
		var errs []error
		processed := 0
		for i := 0; i < contents.Len() && len(data) < limit; i++ {
			processed = i + 1
			remaining := limit - len(data)
			var err error
			data, err = appendStreamBoundedLimitError(data, contents.Index(i), remaining)
			if err != nil {
				errs = append(errs, fmt.Errorf("content stream %d: %w", i+1, err))
			}
			if len(data) < limit {
				data = append(data, '\n')
			}
		}
		if processed < contents.Len() {
			errs = appendLimitError(errs)
		}
		return data, errs
	}
	data, err := appendStreamBoundedLimitError(data, contents, limit)
	if err != nil {
		return data, []error{err}
	}
	return data, nil
}

func appendLimitError(errs []error) []error {
	for _, err := range errs {
		if errors.Is(err, safeio.ErrLimitExceeded) {
			return errs
		}
	}
	return append(errs, safeio.ErrLimitExceeded)
}

type contentLexer struct {
	data []byte
	i    int
	err  error

	// arrays holds the element buffers of top-level array operands from
	// operators that have already run, for reuse. TJ arrays are long and
	// frequent, and growing a fresh slice for each one was the lexer's
	// main allocation. Only operand-stack arrays are pooled: arrays nested
	// in a dictionary may outlive the operator (BDC properties), and every
	// consumer reads a stack array before the operator returns.
	arrays [][]operand
}

// recycle returns the array buffers of a completed operator's operands
// to the pool.
func (lx *contentLexer) recycle(stack []operand) {
	for _, op := range stack {
		if op.kind == opArr && cap(op.arr) > 0 {
			lx.arrays = append(lx.arrays, op.arr[:0])
		}
	}
}

func (lx *contentLexer) setErr(err error) {
	if lx.err == nil {
		lx.err = err
	}
}

// interpretContent executes the operator stream: operands accumulate on a
// stack; each operator keyword invokes do and clears the stack.
func interpretContent(data []byte, do func(op []byte, args []operand)) {
	_ = interpretContentError(data, func(op []byte, args []operand) error {
		do(op, args)
		return nil
	})
}

func interpretContentError(data []byte, do func(op []byte, args []operand) error) error {
	return interpretContentPooled(data, nil, do)
}

// interpretContentPooled is interpretContentError drawing array operand
// buffers from *pool and leaving them there afterwards, so a caller that
// interprets many streams keeps one pool rather than warming a new one
// per stream. The pool may only be shared by lexers that do not run at
// the same time.
func interpretContentPooled(data []byte, pool *[][]operand, do func(op []byte, args []operand) error) error {
	lx := &contentLexer{data: data}
	if pool != nil {
		lx.arrays = *pool
		defer func() { *pool = lx.arrays }()
	}
	var stack []operand
	for lx.i < len(lx.data) {
		lx.skipSpace()
		if lx.i >= len(lx.data) {
			return lx.err
		}
		c := lx.data[lx.i]
		switch {
		case c == '/', c == '(', c == '<', c == '[', c == '{',
			c >= '0' && c <= '9', c == '+', c == '-', c == '.':
			if v, ok := lx.readOperand(0); ok {
				if len(stack) < maxStackDepth {
					stack = append(stack, v)
				} else {
					lx.setErr(errors.New("content operand stack exceeds limit"))
				}
			}
		case c == ']' || c == '>' || c == ')' || c == '}':
			lx.setErr(fmt.Errorf("stray content delimiter %q", c))
			lx.i++ // stray closer in malformed content
		default:
			kw := lx.readKeyword()
			switch string(kw) {
			case "":
				lx.i++
			case "true":
				stack = append(stack, operand{kind: opBool, num: 1})
			case "false":
				stack = append(stack, operand{kind: opBool})
			case "null":
				stack = append(stack, operand{kind: opNull})
			case "BI":
				// An inline image reaches do as the EI operator with its
				// dictionary and data as operands, so the interpreter sees
				// it like any other operator without lexing binary data.
				dict, data, ok := lx.readInlineImage()
				lx.recycle(stack)
				stack = stack[:0]
				if ok {
					if err := do([]byte("EI"), []operand{{kind: opDict, dict: dict}, {kind: opStr, str: data}}); err != nil {
						return err
					}
				}
			default:
				if err := do(kw, stack); err != nil {
					return err
				}
				lx.recycle(stack)
				stack = stack[:0]
			}
		}
	}
	return lx.err
}

func (lx *contentLexer) skipSpace() {
	for lx.i < len(lx.data) {
		c := lx.data[lx.i]
		if c == '%' { // comment to end of line
			for lx.i < len(lx.data) && lx.data[lx.i] != '\n' && lx.data[lx.i] != '\r' {
				lx.i++
			}
			continue
		}
		if !isPDFSpace(c) {
			return
		}
		lx.i++
	}
}

func isPDFSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

func isDelim(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

// readOperand parses one object starting at the cursor. Returns ok=false
// only for tokens that produce no value (e.g. a lone ">>").
func (lx *contentLexer) readOperand(depth int) (operand, bool) {
	if lx.i >= len(lx.data) {
		return operand{}, false
	}
	if depth > maxOperandNest {
		lx.setErr(errors.New("content operand nesting exceeds limit"))
		lx.i++
		return operand{}, false
	}
	switch lx.data[lx.i] {
	case '/':
		return operand{kind: opName, name: lx.readName()}, true
	case '(':
		return operand{kind: opStr, str: lx.readLiteralString()}, true
	case '<':
		if lx.i+1 < len(lx.data) && lx.data[lx.i+1] == '<' {
			return lx.readDict(depth)
		}
		return operand{kind: opStr, str: lx.readHexString()}, true
	case '[':
		return lx.readArray(depth)
	case '{':
		return lx.readProc(), true
	default:
		return lx.readNumber()
	}
}

func (lx *contentLexer) readName() string {
	lx.i++ // consume '/'
	start := lx.i
	for lx.i < len(lx.data) && !isPDFSpace(lx.data[lx.i]) && !isDelim(lx.data[lx.i]) {
		lx.i++
	}
	raw := lx.data[start:lx.i]
	// Decode #xx escapes if present.
	if bytes.IndexByte(raw, '#') >= 0 {
		b := make([]byte, 0, len(raw))
		for j := 0; j < len(raw); j++ {
			if raw[j] == '#' && j+2 < len(raw) {
				if hi, ok1 := hexVal(raw[j+1]); ok1 {
					if lo, ok2 := hexVal(raw[j+2]); ok2 {
						b = append(b, hi<<4|lo)
						j += 2
						continue
					}
				}
			}
			b = append(b, raw[j])
		}
		return string(b)
	}
	return string(raw)
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

func (lx *contentLexer) readLiteralString() []byte {
	lx.i++ // consume '('
	start := lx.i
	escaped := false
	depth := 1
	for lx.i < len(lx.data) {
		switch lx.data[lx.i] {
		case '\\':
			escaped = true
			lx.i++
			if lx.i < len(lx.data) {
				lx.i++ // escaped delimiters do not affect nesting
			}
		case '(':
			depth++
			lx.i++
		case ')':
			depth--
			if depth == 0 {
				raw := lx.data[start:lx.i]
				lx.i++
				if escaped {
					return decodeLiteralString(raw)
				}
				return raw
			}
			lx.i++
		default:
			lx.i++
		}
	}
	raw := lx.data[start:]
	lx.setErr(errors.New("unterminated literal string in content stream"))
	if escaped {
		return decodeLiteralString(raw)
	}
	return raw
}

func decodeLiteralString(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			out = append(out, raw[i])
			continue
		}
		i++
		if i >= len(raw) {
			break
		}
		e := raw[i]
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
		case '\n': // line continuation
		case '\r':
			if i+1 < len(raw) && raw[i+1] == '\n' {
				i++
			}
		default:
			if e >= '0' && e <= '7' { // 1-3 octal digits
				v := int(e - '0')
				for range 2 {
					if i+1 >= len(raw) || raw[i+1] < '0' || raw[i+1] > '7' {
						break
					}
					v = v*8 + int(raw[i+1]-'0')
					i++
				}
				out = append(out, byte(v))
			} else {
				out = append(out, e)
			}
		}
	}
	return out
}

func (lx *contentLexer) readHexString() []byte {
	lx.i++ // consume '<'
	encodedLen := len(lx.data) - lx.i
	if end := bytes.IndexByte(lx.data[lx.i:], '>'); end >= 0 {
		encodedLen = end
	}
	out := make([]byte, 0, min((encodedLen+1)/2, 256))
	var hi byte
	haveHi := false
	for lx.i < len(lx.data) && lx.data[lx.i] != '>' {
		if v, ok := hexVal(lx.data[lx.i]); ok {
			if haveHi {
				out = append(out, hi<<4|v)
				haveHi = false
			} else {
				hi = v
				haveHi = true
			}
		}
		lx.i++
	}
	if lx.i < len(lx.data) {
		lx.i++ // consume '>'
	} else {
		lx.setErr(errors.New("unterminated hexadecimal string in content stream"))
	}
	if haveHi {
		out = append(out, hi<<4)
	}
	return out
}

func (lx *contentLexer) readArray(depth int) (operand, bool) {
	lx.i++ // consume '['
	arr := operand{kind: opArr}
	if depth == 0 {
		if n := len(lx.arrays); n > 0 {
			arr.arr = lx.arrays[n-1]
			lx.arrays = lx.arrays[:n-1]
		}
	}
	closed := false
	for lx.i < len(lx.data) {
		lx.skipSpace()
		if lx.i >= len(lx.data) {
			break
		}
		if lx.data[lx.i] == ']' {
			lx.i++
			closed = true
			break
		}
		before := lx.i
		if v, ok := lx.readOperand(depth + 1); ok && len(arr.arr) < maxStackDepth {
			arr.arr = append(arr.arr, v)
		}
		if lx.i == before {
			lx.i++ // always advance
		}
	}
	if !closed {
		lx.setErr(errors.New("unterminated array in content stream"))
	}
	return arr, true
}

func (lx *contentLexer) readDict(depth int) (operand, bool) {
	lx.i += 2 // consume '<<'
	out := operand{kind: opDict, dict: map[string]operand{}}
	for lx.i < len(lx.data) {
		lx.skipSpace()
		if lx.i+1 < len(lx.data) && lx.data[lx.i] == '>' && lx.data[lx.i+1] == '>' {
			lx.i += 2
			return out, true
		}
		if lx.i >= len(lx.data) {
			break
		}
		before := lx.i
		switch lx.data[lx.i] {
		case '/':
			key := lx.readName()
			lx.skipSpace()
			if lx.i < len(lx.data) && (lx.data[lx.i] == '/' || lx.data[lx.i] == '(' ||
				lx.data[lx.i] == '<' || lx.data[lx.i] == '[' || lx.data[lx.i] == '{' ||
				lx.data[lx.i] == '+' || lx.data[lx.i] == '-' || lx.data[lx.i] == '.' ||
				lx.data[lx.i] >= '0' && lx.data[lx.i] <= '9') {
				if value, ok := lx.readOperand(depth + 1); ok {
					out.dict[key] = value
				}
			} else if kw := lx.readKeyword(); len(kw) > 0 {
				switch string(kw) {
				case "true":
					out.dict[key] = operand{kind: opBool, num: 1}
				case "false":
					out.dict[key] = operand{kind: opBool}
				case "null":
					out.dict[key] = operand{kind: opNull}
				}
			}
		case '(', '<', '[', '{':
			_, _ = lx.readOperand(depth + 1)
		default:
			lx.readKeyword()
		}
		if lx.i == before {
			lx.i++
		}
	}
	lx.setErr(errors.New("unterminated dictionary in content stream"))
	return out, true
}

// readProc consumes a { ... } PostScript procedure (Type 4 functions).
func (lx *contentLexer) readProc() operand {
	depth := 0
	for lx.i < len(lx.data) {
		switch lx.data[lx.i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				lx.i++
				return operand{kind: opProc}
			}
		case '(':
			lx.readLiteralString()
			continue
		}
		lx.i++
	}
	lx.setErr(errors.New("unterminated procedure in content stream"))
	return operand{kind: opProc}
}

func (lx *contentLexer) readNumber() (operand, bool) {
	start := lx.i
	for lx.i < len(lx.data) && !isPDFSpace(lx.data[lx.i]) && !isDelim(lx.data[lx.i]) {
		lx.i++
	}
	if lx.i == start {
		lx.i++
		return operand{}, false
	}
	tok := lx.data[start:lx.i]
	if f, ok := parsePlainNumber(tok); ok {
		return operand{kind: opNum, num: f}, true
	}
	f, err := strconv.ParseFloat(string(tok), 64)
	if err != nil {
		return operand{}, false
	}
	return operand{kind: opNum, num: f}, true
}

// parsePlainNumber parses the decimal forms content streams use —
// [+-]digits[.digits] — without allocating. The mantissa is kept below
// 2^53 and the scale within the exactly representable powers of ten, so
// the single division is correctly rounded and the result is the one
// strconv.ParseFloat returns. Anything else (exponents, malformed
// tokens) is left to strconv.
func parsePlainNumber(tok []byte) (float64, bool) {
	i, neg := 0, false
	if i < len(tok) && (tok[i] == '+' || tok[i] == '-') {
		neg = tok[i] == '-'
		i++
	}
	var mant uint64
	digits, frac, dot := 0, 0, false
	for ; i < len(tok); i++ {
		switch c := tok[i]; {
		case c >= '0' && c <= '9':
			mant = mant*10 + uint64(c-'0')
			digits++
			if dot {
				frac++
			}
		case c == '.' && !dot:
			dot = true
		default:
			return 0, false
		}
	}
	if digits == 0 || digits > 15 || frac >= len(exactPow10) {
		return 0, false
	}
	f := float64(mant)
	if frac > 0 {
		f /= exactPow10[frac]
	}
	if neg {
		f = -f
	}
	return f, true
}

// exactPow10 holds the powers of ten that are exact in float64.
var exactPow10 = [...]float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11,
	1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22,
}

func (lx *contentLexer) readKeyword() []byte {
	start := lx.i
	for lx.i < len(lx.data) && !isPDFSpace(lx.data[lx.i]) && !isDelim(lx.data[lx.i]) {
		lx.i++
	}
	if lx.i == start {
		return nil
	}
	return lx.data[start:lx.i]
}

// readInlineImage consumes everything from after BI through the EI
// marker — the image dictionary, the ID keyword, then binary data no lexer
// can parse — and returns the dictionary and the data. ok is false when
// the sequence is malformed; the lexer still skips what it can.
//
// The data's extent is known exactly for unfiltered images, from the
// dictionary. Otherwise it ends at the first whitespace-delimited EI that
// is followed by something that looks like content again, since encoded
// data can contain the bytes "EI" by chance.
func (lx *contentLexer) readInlineImage() (dict map[string]operand, data []byte, ok bool) {
	dict = map[string]operand{}
	foundData := false
	for lx.i < len(lx.data) {
		lx.skipSpace()
		if lx.i >= len(lx.data) {
			break
		}
		before := lx.i
		switch c := lx.data[lx.i]; c {
		case '/':
			key := lx.readName()
			lx.skipSpace()
			if lx.i < len(lx.data) {
				if c := lx.data[lx.i]; c == '/' || c == '(' || c == '<' || c == '[' || c == '{' ||
					c == '+' || c == '-' || c == '.' || c >= '0' && c <= '9' {
					if value, ok := lx.readOperand(1); ok {
						dict[key] = value
					}
				} else if kw := lx.readKeyword(); len(kw) > 0 {
					switch string(kw) {
					case "true":
						dict[key] = operand{kind: opBool, num: 1}
					case "false":
						dict[key] = operand{kind: opBool}
					case "null":
						dict[key] = operand{kind: opNull}
					}
				}
			}
		case '(', '<', '[':
			lx.readOperand(1) // a stray value with no key
		default:
			if kw := lx.readKeyword(); len(kw) == 2 && kw[0] == 'I' && kw[1] == 'D' {
				lx.i++ // the single whitespace byte after ID
				foundData = true
			}
		}
		if foundData {
			break
		}
		if lx.i == before {
			lx.i++
		}
	}
	if !foundData {
		lx.setErr(errors.New("inline image missing ID marker"))
		return nil, nil, false
	}
	start := min(lx.i, len(lx.data))
	if n := inlineImageLength(dict); n >= 0 && start+n <= len(lx.data) {
		if end, found := lx.inlineImageEnd(start + n); found {
			lx.i = end
			return dict, lx.data[start : start+n], true
		}
	}
	for lx.i = start; lx.i+1 < len(lx.data); lx.i++ {
		if lx.data[lx.i] != 'E' || lx.data[lx.i+1] != 'I' {
			continue
		}
		if end, found := lx.inlineImageEnd(lx.i); found {
			data := lx.data[start:lx.i]
			if n := len(data); n > 0 && isPDFSpace(data[n-1]) {
				data = data[:n-1] // the whitespace before EI is not image data
			}
			lx.i = end
			return dict, data, true
		}
	}
	lx.i = len(lx.data)
	lx.setErr(errors.New("inline image missing EI marker"))
	return nil, nil, false
}

// inlineImageEnd reports whether an EI marker ends the inline image at
// or just after position i: optional whitespace, EI, then a delimiter or
// whitespace, then — since encoded image data can contain those bytes
// too — only printable ASCII in the bytes that follow.
func (lx *contentLexer) inlineImageEnd(i int) (end int, found bool) {
	for i < len(lx.data) && isPDFSpace(lx.data[i]) {
		i++
	}
	if i+1 >= len(lx.data) || lx.data[i] != 'E' || lx.data[i+1] != 'I' {
		return 0, false
	}
	if i > 0 && !isPDFSpace(lx.data[i-1]) {
		return 0, false
	}
	after := i + 2
	if after < len(lx.data) && !isPDFSpace(lx.data[after]) && !isDelim(lx.data[after]) {
		return 0, false
	}
	for _, c := range lx.data[after:min(after+16, len(lx.data))] {
		if !isPDFSpace(c) && (c < 0x20 || c > 0x7E) {
			return 0, false
		}
	}
	return after, true
}
