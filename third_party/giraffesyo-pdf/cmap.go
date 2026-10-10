package pdf

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/giraffesyo/pdf/internal/object"
)

const maxCmapEntries = 100000

type codeKey struct {
	value uint32
	bytes uint8
}

type codeSpace struct {
	low   uint32
	high  uint32
	bytes uint8
}

type cmapData struct {
	spaces   []codeSpace
	unicode  map[codeKey]string
	cids     map[codeKey]uint32
	identity bool
	wmode    int
}

func identityCMap(vertical bool) *cmapData {
	wmode := 0
	if vertical {
		wmode = 1
	}
	return &cmapData{
		spaces:   []codeSpace{{low: 0, high: 0xffff, bytes: 2}},
		identity: true,
		wmode:    wmode,
	}
}

// parseToUnicode reads a /ToUnicode CMap stream. Codespace ranges are retained
// so composite-font strings can be split into variable-width character codes.
func parseToUnicode(v object.Value, resolver CMapResolver, streamLimit int) (*cmapData, error) {
	if v.Kind() != object.Stream {
		return nil, nil
	}
	data, err := readStreamBoundedLimitError(v, streamLimit)
	if err != nil {
		return nil, err
	}
	return parseCMap(data, resolver, map[string]bool{})
}

func parseEncodingCMap(v object.Value, resolver CMapResolver, streamLimit int) (*cmapData, error) {
	switch v.Kind() {
	case object.Name:
		return resolveCMap(v.Name(), resolver, map[string]bool{})
	case object.Stream:
		data, err := readStreamBoundedLimitError(v, streamLimit)
		if err != nil {
			return nil, err
		}
		return parseCMap(data, resolver, map[string]bool{})
	default:
		return nil, errors.New("composite font has no usable Encoding CMap")
	}
}

func resolveCMap(name string, resolver CMapResolver, seen map[string]bool) (*cmapData, error) {
	switch name {
	case "Identity-H":
		return identityCMap(false), nil
	case "Identity-V":
		return identityCMap(true), nil
	}
	if seen[name] {
		return nil, fmt.Errorf("cyclic usecmap reference %q", name)
	}
	if resolver == nil {
		return nil, fmt.Errorf("predefined CMap %q requires a CMapResolver", name)
	}
	seen[name] = true
	data, err := resolver(name)
	if err != nil {
		return nil, fmt.Errorf("resolve CMap %q: %w", name, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("resolve CMap %q: empty CMap", name)
	}
	cmap, err := parseCMap(data, resolver, seen)
	delete(seen, name)
	return cmap, err
}

func parseCMap(data []byte, resolver CMapResolver, seen map[string]bool) (*cmapData, error) {
	toks := lexCMap(data)
	out := &cmapData{
		unicode: map[codeKey]string{},
		cids:    map[codeKey]uint32{},
	}
	var parseErr error
	for i := 0; i < len(toks); i++ {
		switch toks[i].kw {
		case "begincodespacerange":
			for i+2 < len(toks) && toks[i+1].isHex() && toks[i+2].isHex() {
				lo, hi := toks[i+1].hex, toks[i+2].hex
				i += 2
				if len(lo) == 0 || len(lo) != len(hi) || len(lo) > 4 {
					parseErr = errors.New("invalid CMap codespace range")
					continue
				}
				out.spaces = append(out.spaces, codeSpace{
					low:   beUint(lo),
					high:  beUint(hi),
					bytes: uint8(len(lo)), //nolint:gosec // validated as 1..4 above
				})
			}
		case "beginbfchar":
			for i+2 < len(toks) && toks[i+1].isHex() && toks[i+2].isHex() {
				src, dst := toks[i+1].hex, toks[i+2].hex
				i += 2
				if key, ok := makeCodeKey(src); ok {
					out.unicode[key] = utf16be(dst)
				}
				if out.entryCount() >= maxCmapEntries {
					return out, errors.New("CMap entry limit reached")
				}
			}
		case "beginbfrange":
			i = parseBFRange(toks, i, out)
			if out.entryCount() >= maxCmapEntries {
				return out, errors.New("CMap entry limit reached")
			}
		case "begincidchar":
			for i+2 < len(toks) && toks[i+1].isHex() {
				cid, ok := tokenUint(toks[i+2])
				if !ok {
					break
				}
				if key, keyOK := makeCodeKey(toks[i+1].hex); keyOK {
					out.cids[key] = cid
				}
				i += 2
				if out.entryCount() >= maxCmapEntries {
					return out, errors.New("CMap entry limit reached")
				}
			}
		case "begincidrange":
			for i+3 < len(toks) && toks[i+1].isHex() && toks[i+2].isHex() {
				startCID, ok := tokenUint(toks[i+3])
				if !ok {
					break
				}
				loBytes, hiBytes := toks[i+1].hex, toks[i+2].hex
				i += 3
				if len(loBytes) == 0 || len(loBytes) != len(hiBytes) || len(loBytes) > 4 {
					parseErr = errors.New("invalid CMap CID range")
					continue
				}
				lo, hi := beUint(loBytes), beUint(hiBytes)
				remaining := maxCmapEntries - out.entryCount()
				if hi < lo {
					parseErr = errors.New("invalid CMap CID range")
					continue
				}
				span := uint64(hi) - uint64(lo) + 1
				if span > uint64(remaining) || uint64(startCID)+span-1 > 0xffff { //nolint:gosec // remaining is nonnegative
					parseErr = errors.New("CMap CID range exceeds limit")
					continue
				}
				for code := lo; ; code++ {
					key := codeKey{value: code, bytes: uint8(len(loBytes))} //nolint:gosec // validated as 1..4
					out.cids[key] = startCID + code - lo
					if code == hi {
						break
					}
				}
			}
		case "usecmap":
			if i == 0 {
				continue
			}
			name := strings.TrimPrefix(toks[i-1].kw, "/")
			if name == "" {
				continue
			}
			base, err := resolveCMap(name, resolver, seen)
			if err != nil {
				parseErr = err
			} else {
				out.inherit(base)
			}
		case "def":
			if i >= 2 && toks[i-2].kw == "/WMode" {
				if mode, ok := tokenUint(toks[i-1]); ok && mode == 1 {
					out.wmode = 1
				}
			}
		}
	}
	if len(out.spaces) == 0 {
		out.spaces = inferCodeSpaces(out)
	}
	if len(out.unicode) == 0 {
		out.unicode = nil
	}
	if len(out.cids) == 0 {
		out.cids = nil
	}
	return out, parseErr
}

func parseBFRange(toks []cmTok, i int, out *cmapData) int {
	for i+3 < len(toks) && toks[i+1].isHex() && toks[i+2].isHex() {
		loBytes, hiBytes := toks[i+1].hex, toks[i+2].hex
		if len(loBytes) == 0 || len(loBytes) != len(hiBytes) || len(loBytes) > 4 {
			return i + 2
		}
		lo, hi := beUint(loBytes), beUint(hiBytes)
		if hi < lo || hi-lo > 65535 {
			return i + 2
		}
		dst := toks[i+3]
		switch {
		case dst.isHex():
			units := utf16beUnits(dst.hex)
			for code := lo; out.entryCount() < maxCmapEntries; code++ {
				key := codeKey{value: code, bytes: uint8(len(loBytes))} //nolint:gosec // validated as 1..4
				out.unicode[key] = incrementedUTF16(units, code-lo)
				if code == hi {
					break
				}
			}
			i += 3
		case dst.kw == "[":
			j := i + 4
			code := lo
			for j < len(toks) && toks[j].kw != "]" {
				if toks[j].isHex() && code <= hi && out.entryCount() < maxCmapEntries {
					key := codeKey{value: code, bytes: uint8(len(loBytes))} //nolint:gosec // validated as 1..4
					out.unicode[key] = utf16be(toks[j].hex)
					code++
				}
				j++
			}
			i = j
		default:
			return i + 3
		}
	}
	return i
}

func (c *cmapData) entryCount() int {
	return len(c.unicode) + len(c.cids)
}

func (c *cmapData) inherit(base *cmapData) {
	if base == nil {
		return
	}
	if len(c.spaces) == 0 {
		c.spaces = append(c.spaces, base.spaces...)
	}
	if base.wmode == 1 {
		c.wmode = 1
	}
	if base.identity {
		c.identity = true
	}
	for k, v := range base.unicode {
		if _, exists := c.unicode[k]; !exists {
			c.unicode[k] = v
		}
	}
	for k, v := range base.cids {
		if _, exists := c.cids[k]; !exists {
			c.cids[k] = v
		}
	}
}

func inferCodeSpaces(c *cmapData) []codeSpace {
	seen := [5]bool{}
	var spaces []codeSpace
	for key := range c.unicode {
		seen[key.bytes] = true
	}
	for key := range c.cids {
		seen[key.bytes] = true
	}
	for n := 1; n <= 4; n++ {
		if seen[n] {
			high := uint32(1)<<(8*n) - 1
			if n == 4 {
				high = ^uint32(0)
			}
			spaces = append(spaces, codeSpace{high: high, bytes: uint8(n)})
		}
	}
	return spaces
}

func makeCodeKey(raw []byte) (codeKey, bool) {
	if len(raw) == 0 || len(raw) > 4 {
		return codeKey{}, false
	}
	return codeKey{value: beUint(raw), bytes: uint8(len(raw))}, true //nolint:gosec // validated as 1..4
}

func (c *cmapData) nextCode(raw []byte) (codeKey, int) {
	if c != nil {
		for _, space := range c.spaces {
			n := int(space.bytes)
			if n == 0 || n > len(raw) {
				continue
			}
			value := beUint(raw[:n])
			if value >= space.low && value <= space.high {
				return codeKey{value: value, bytes: space.bytes}, n
			}
		}
	}
	if len(raw) == 0 {
		return codeKey{}, 0
	}
	return codeKey{value: uint32(raw[0]), bytes: 1}, 1
}

func (c *cmapData) cid(key codeKey) uint32 {
	if c == nil {
		return key.value
	}
	if cid, ok := c.cids[key]; ok {
		return cid
	}
	if c.identity {
		return key.value
	}
	return key.value
}

func tokenUint(t cmTok) (uint32, bool) {
	if t.kw == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(t.kw, 10, 32)
	return uint32(n), err == nil
}

type cmTok struct {
	hex []byte
	kw  string
}

func (t cmTok) isHex() bool { return t.hex != nil }

func lexCMap(data []byte) []cmTok {
	var toks []cmTok
	for i := 0; i < len(data); {
		c := data[i]
		switch {
		case c == '%':
			for i < len(data) && data[i] != '\n' {
				i++
			}
		case c == '<':
			j := i + 1
			var hexDigits []byte
			for j < len(data) && data[j] != '>' {
				if isHexDigit(data[j]) {
					hexDigits = append(hexDigits, data[j])
				}
				j++
			}
			if len(hexDigits)%2 == 1 {
				hexDigits = append(hexDigits, '0')
			}
			raw := make([]byte, len(hexDigits)/2)
			_, _ = hex.Decode(raw, hexDigits)
			toks = append(toks, cmTok{hex: raw})
			i = min(j+1, len(data))
		case c == '[' || c == ']':
			toks = append(toks, cmTok{kw: string(c)})
			i++
		case c == '(':
			depth := 1
			i++
			for i < len(data) && depth > 0 {
				switch data[i] {
				case '\\':
					i++
				case '(':
					depth++
				case ')':
					depth--
				}
				i++
			}
		case c <= ' ':
			i++
		default:
			j := i
			for j < len(data) && data[j] > ' ' && data[j] != '<' && data[j] != '[' &&
				data[j] != ']' && data[j] != '(' && data[j] != '%' {
				j++
			}
			toks = append(toks, cmTok{kw: string(data[i:j])})
			i = j
		}
		if len(toks) > 6*maxCmapEntries {
			break
		}
	}
	return toks
}

func isHexDigit(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

func beUint(b []byte) uint32 {
	var v uint32
	for _, x := range b {
		v = v<<8 | uint32(x)
	}
	return v
}

func utf16beUnits(b []byte) []uint16 {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
	}
	return units
}

func utf16be(b []byte) string {
	return string(utf16.Decode(utf16beUnits(b)))
}

func incrementedUTF16(units []uint16, off uint32) string {
	if len(units) == 0 {
		return ""
	}
	out := make([]uint16, len(units))
	copy(out, units)
	out[len(out)-1] += uint16(off & 0xFFFF)
	return string(utf16.Decode(out))
}
