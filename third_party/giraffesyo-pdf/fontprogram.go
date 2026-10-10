package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/giraffesyo/pdf/internal/encoding"
	"github.com/giraffesyo/pdf/internal/object"
)

const maxFontCMapEntries = 250000

func embeddedSimpleFallback(font object.Value, streamLimit int) (map[uint32]string, error) {
	desc := font.Key("FontDescriptor")
	if desc.Kind() != object.Dict {
		return nil, nil
	}
	if stream, ok := sfntFontStream(desc); ok {
		data, err := readStreamBoundedLimitError(stream, streamLimit)
		if err != nil {
			return nil, fmt.Errorf("read embedded OpenType font: %w", err)
		}
		mapping, err := parseSFNTCMap(data)
		if err != nil {
			return nil, err
		}
		return simpleMapping(mapping), nil
	}
	if stream := desc.Key("FontFile"); stream.Kind() == object.Stream {
		data, err := readStreamBoundedLimitError(stream, streamLimit)
		if err != nil {
			return nil, fmt.Errorf("read embedded Type1 font: %w", err)
		}
		return parseType1Encoding(data), nil
	}
	if stream := desc.Key("FontFile3"); stream.Kind() == object.Stream &&
		stream.Key("Subtype").Name() == "Type1C" {
		data, err := readStreamBoundedLimitError(stream, streamLimit)
		if err != nil {
			return nil, fmt.Errorf("read embedded CFF font: %w", err)
		}
		return parseCFFEncoding(data)
	}
	return nil, nil
}

func embeddedCompositeFallback(desc object.Value, streamLimit int) (map[uint32]string, error) {
	fontDesc := desc.Key("FontDescriptor")
	if fontDesc.Kind() != object.Dict {
		return nil, nil
	}
	stream, ok := sfntFontStream(fontDesc)
	if !ok {
		return nil, nil
	}
	data, err := readStreamBoundedLimitError(stream, streamLimit)
	if err != nil {
		return nil, fmt.Errorf("read embedded OpenType font: %w", err)
	}
	mapping, err := parseSFNTCMap(data)
	if err != nil {
		return nil, err
	}
	reverse := reverseGlyphMapping(mapping)
	cidToGID := desc.Key("CIDToGIDMap")
	out := map[uint32]string{}
	if cidToGID.Kind() == object.Stream {
		raw, readErr := readStreamBoundedLimitError(cidToGID, 2*65536)
		if readErr != nil {
			return nil, fmt.Errorf("read CIDToGIDMap: %w", readErr)
		}
		for cid := 0; cid+1 < len(raw) && cid/2 <= 0xffff; cid += 2 {
			gid := binary.BigEndian.Uint16(raw[cid : cid+2])
			if text := reverse[gid]; text != "" {
				out[uint32(cid/2)] = text
			}
		}
		return out, nil
	}
	// /Identity is the default for CIDFontType2.
	for gid, text := range reverse {
		out[uint32(gid)] = text
	}
	return out, nil
}

func sfntFontStream(desc object.Value) (object.Value, bool) {
	if stream := desc.Key("FontFile2"); stream.Kind() == object.Stream {
		return stream, true
	}
	if stream := desc.Key("FontFile3"); stream.Kind() == object.Stream &&
		stream.Key("Subtype").Name() == "OpenType" {
		return stream, true
	}
	return object.Value{}, false
}

type glyphMapping map[uint32]uint16 // Unicode code point -> glyph ID

func parseSFNTCMap(data []byte) (glyphMapping, error) {
	base := 0
	if len(data) >= 16 && string(data[:4]) == "ttcf" {
		if count, ok := u32(data, 8); !ok || count == 0 {
			return nil, errors.New("embedded TrueType collection has no fonts")
		}
		offset, ok := u32(data, 12)
		if !ok || uint64(offset) > uint64(len(data)) {
			return nil, errors.New("invalid TrueType collection offset")
		}
		base = int(offset)
	}
	numTables, ok := u16(data, base+4)
	if !ok || numTables > 4096 {
		return nil, errors.New("invalid OpenType table directory")
	}
	var cmap []byte
	for i := range int(numTables) {
		record := base + 12 + 16*i
		if record+16 > len(data) {
			return nil, errors.New("truncated OpenType table directory")
		}
		if string(data[record:record+4]) != "cmap" {
			continue
		}
		offsetValue := binary.BigEndian.Uint32(data[record+8 : record+12])
		lengthValue := binary.BigEndian.Uint32(data[record+12 : record+16])
		if uint64(offsetValue)+uint64(lengthValue) > uint64(len(data)) {
			return nil, errors.New("invalid OpenType cmap bounds")
		}
		offset, length := int(offsetValue), int(lengthValue)
		cmap = data[offset : offset+length]
		break
	}
	if cmap == nil {
		return nil, errors.New("embedded OpenType font has no cmap")
	}
	return parseCMapTable(cmap)
}

func parseCMapTable(data []byte) (glyphMapping, error) {
	numTables, ok := u16(data, 2)
	if !ok || numTables > 1024 || 4+8*int(numTables) > len(data) {
		return nil, errors.New("invalid OpenType cmap header")
	}
	type subtable struct {
		priority int
		offset   int
		macRoman bool
	}
	var tables []subtable
	for i := range int(numTables) {
		record := 4 + 8*i
		platform := binary.BigEndian.Uint16(data[record : record+2])
		encodingID := binary.BigEndian.Uint16(data[record+2 : record+4])
		offsetValue := binary.BigEndian.Uint32(data[record+4 : record+8])
		if uint64(offsetValue)+2 > uint64(len(data)) {
			continue
		}
		offset := int(offsetValue)
		var priority int
		macRoman := false
		switch {
		case platform == 0:
			priority = 5
		case platform == 3 && encodingID == 10:
			priority = 4
		case platform == 3 && encodingID == 1:
			priority = 3
		case platform == 1 && encodingID == 0:
			priority = 2
			macRoman = true
		case platform == 3 && encodingID == 0:
			priority = 1
		default:
			continue
		}
		tables = append(tables, subtable{priority: priority, offset: offset, macRoman: macRoman})
	}
	out := glyphMapping{}
	for priority := 1; priority <= 5; priority++ {
		for _, table := range tables {
			if table.priority != priority {
				continue
			}
			target := out
			if table.macRoman {
				target = glyphMapping{}
			}
			if err := parseCMapSubtable(data[table.offset:], target); err != nil {
				continue
			}
			if table.macRoman {
				convertMacRomanMapping(target, out)
			}
			if len(out) >= maxFontCMapEntries {
				return out, errors.New("embedded font cmap entry limit reached")
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("embedded OpenType font has no supported Unicode cmap")
	}
	return out, nil
}

func convertMacRomanMapping(source, target glyphMapping) {
	macRoman := encoding.New("MacRomanEncoding", nil)
	for code, gid := range source {
		if code > 0xff {
			continue
		}
		runes := []rune(macRoman.Decode(byte(code)))
		if len(runes) == 1 {
			// Retain the original byte for simple fonts and add its Unicode
			// scalar for glyph-ID reversal in composite fonts.
			target[code] = gid
			target[uint32(runes[0])] = gid
		}
	}
}

func parseCMapSubtable(data []byte, out glyphMapping) error {
	format, ok := u16(data, 0)
	if !ok {
		return ioBoundsError("cmap format")
	}
	switch format {
	case 0:
		if len(data) < 262 {
			return ioBoundsError("cmap format 0")
		}
		for code, gid := range data[6:262] {
			if gid != 0 {
				out[uint32(code)] = uint16(gid)
			}
		}
	case 4:
		return parseCMapFormat4(data, out)
	case 6:
		first, ok1 := u16(data, 6)
		count, ok2 := u16(data, 8)
		if !ok1 || !ok2 || 10+2*int(count) > len(data) {
			return ioBoundsError("cmap format 6")
		}
		for i := range int(count) {
			gid := binary.BigEndian.Uint16(data[10+2*i : 12+2*i])
			if gid != 0 {
				out[uint32(first)+uint32(i)] = gid
			}
		}
	case 12, 13:
		return parseCMapFormat12(data, out, format == 13)
	default:
		return fmt.Errorf("unsupported OpenType cmap format %d", format)
	}
	return nil
}

func parseCMapFormat4(data []byte, out glyphMapping) error {
	length, ok1 := u16(data, 2)
	segCountX2, ok2 := u16(data, 6)
	if !ok1 || !ok2 || int(length) > len(data) || segCountX2 == 0 || segCountX2%2 != 0 {
		return ioBoundsError("cmap format 4")
	}
	data = data[:length]
	segCount := int(segCountX2 / 2)
	endBase := 14
	startBase := endBase + 2*segCount + 2
	deltaBase := startBase + 2*segCount
	rangeBase := deltaBase + 2*segCount
	if rangeBase+2*segCount > len(data) {
		return ioBoundsError("cmap format 4 arrays")
	}
	for segment := range segCount {
		end := binary.BigEndian.Uint16(data[endBase+2*segment : endBase+2*segment+2])
		start := binary.BigEndian.Uint16(data[startBase+2*segment : startBase+2*segment+2])
		delta := binary.BigEndian.Uint16(data[deltaBase+2*segment : deltaBase+2*segment+2])
		rangeOffset := binary.BigEndian.Uint16(data[rangeBase+2*segment : rangeBase+2*segment+2])
		remaining := maxFontCMapEntries - len(out)
		if end < start || int(end)-int(start)+1 > remaining {
			continue
		}
		for code := uint32(start); code <= uint32(end); code++ {
			if code == 0xffff {
				continue
			}
			var gid uint16
			if rangeOffset == 0 {
				gid = uint16(code) + delta //nolint:gosec // code is bounded by uint16 start/end
			} else {
				pos := rangeBase + 2*segment + int(rangeOffset) + 2*int(code-uint32(start))
				if pos < 0 || pos+2 > len(data) {
					continue
				}
				gid = binary.BigEndian.Uint16(data[pos : pos+2])
				if gid != 0 {
					gid += delta
				}
			}
			if gid != 0 {
				out[code] = gid
			}
		}
	}
	return nil
}

func parseCMapFormat12(data []byte, out glyphMapping, constant bool) error {
	length, ok1 := u32(data, 4)
	groups, ok2 := u32(data, 12)
	if !ok1 || !ok2 || uint64(length) > uint64(len(data)) || groups > maxFontCMapEntries ||
		16+12*uint64(groups) > uint64(length) {
		return ioBoundsError("cmap format 12")
	}
	for i := range int(groups) {
		offset := 16 + 12*i
		start := binary.BigEndian.Uint32(data[offset : offset+4])
		end := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		startGID := binary.BigEndian.Uint32(data[offset+8 : offset+12])
		remaining := maxFontCMapEntries - len(out)
		if end < start ||
			uint64(end)-uint64(start)+1 > uint64(remaining) { //nolint:gosec // remaining is nonnegative
			continue
		}
		for code := start; ; code++ {
			gid := startGID
			if !constant {
				gid += code - start
			}
			if gid > 0 && gid <= 0xffff && code <= utf8.MaxRune {
				out[code] = uint16(gid)
			}
			if code == end {
				break
			}
		}
	}
	return nil
}

func simpleMapping(mapping glyphMapping) map[uint32]string {
	reverse := reverseGlyphMapping(mapping)
	out := map[uint32]string{}
	for code, gid := range mapping {
		text := reverse[gid]
		switch {
		case code <= 0xff:
			out[code] = text
		case code >= 0xf000 && code <= 0xf0ff:
			out[code&0xff] = text
		}
	}
	return out
}

func reverseGlyphMapping(mapping glyphMapping) map[uint16]string {
	out := map[uint16]string{}
	quality := map[uint16]int{}
	chosen := map[uint16]uint32{}
	for code, gid := range mapping {
		if code > utf8.MaxRune {
			continue
		}
		r := rune(code)
		if gid == 0 || unicode.Is(unicode.Co, r) || !unicode.IsGraphic(r) && r != ' ' {
			continue
		}
		q := 2
		if code < 0x80 {
			q = 3
		}
		if q > quality[gid] || q == quality[gid] && code < chosen[gid] {
			out[gid] = string(r)
			quality[gid] = q
			chosen[gid] = code
		}
	}
	return out
}

var type1EncodingPattern = regexp.MustCompile(`(?m)\bdup[ \t]+([0-9]{1,3})[ \t]+/([A-Za-z0-9_.]+)[ \t]+put\b`)

// parseType1Encoding reads the /Encoding array of a Type1 font program.
// Only the cleartext portion before the eexec operator is scanned: the
// encoding lives there by construction (Adobe Type 1 Font Format, §2.3),
// and the encrypted private portion that follows is typically far larger,
// so matching across it made embedded-font decoding dominate extraction.
func parseType1Encoding(data []byte) map[uint32]string {
	if i := bytes.Index(data, []byte("eexec")); i >= 0 {
		data = data[:i]
	}
	out := map[uint32]string{}
	for _, match := range type1EncodingPattern.FindAllSubmatch(data, 512) {
		code, err := strconv.ParseUint(string(match[1]), 10, 8)
		if err != nil {
			continue
		}
		if text := encoding.GlyphToText(string(match[2])); text != "" {
			out[uint32(code)] = text
		}
	}
	return out
}

func u16(data []byte, offset int) (uint16, bool) {
	if offset < 0 || offset+2 > len(data) {
		return 0, false
	}
	return binary.BigEndian.Uint16(data[offset : offset+2]), true
}

func u32(data []byte, offset int) (uint32, bool) {
	if offset < 0 || offset+4 > len(data) {
		return 0, false
	}
	return binary.BigEndian.Uint32(data[offset : offset+4]), true
}

func ioBoundsError(part string) error { return fmt.Errorf("truncated OpenType %s", part) }

type cffIndex struct {
	objects [][]byte
	next    int
}

func parseCFFEncoding(data []byte) (map[uint32]string, error) {
	if len(data) < 4 {
		return nil, errors.New("truncated embedded CFF font")
	}
	headerSize := int(data[2])
	if headerSize < 4 || headerSize > len(data) {
		return nil, errors.New("invalid CFF header size")
	}
	nameIndex, err := parseCFFIndex(data, headerSize)
	if err != nil || len(nameIndex.objects) == 0 {
		return nil, errors.New("invalid CFF Name INDEX")
	}
	topIndex, err := parseCFFIndex(data, nameIndex.next)
	if err != nil || len(topIndex.objects) == 0 {
		return nil, errors.New("invalid CFF Top DICT INDEX")
	}
	stringIndex, err := parseCFFIndex(data, topIndex.next)
	if err != nil {
		return nil, fmt.Errorf("invalid CFF String INDEX: %w", err)
	}
	globalSubrs, err := parseCFFIndex(data, stringIndex.next)
	if err != nil {
		return nil, fmt.Errorf("invalid CFF Global Subrs INDEX: %w", err)
	}
	_ = globalSubrs
	top := parseCFFDict(topIndex.objects[0])
	charStringsOffset := top[17]
	if charStringsOffset <= 0 || charStringsOffset >= len(data) {
		return nil, errors.New("CFF Top DICT has no valid CharStrings offset")
	}
	charStrings, err := parseCFFIndex(data, charStringsOffset)
	if err != nil || len(charStrings.objects) == 0 || len(charStrings.objects) > 65536 {
		return nil, errors.New("invalid CFF CharStrings INDEX")
	}
	charset, err := parseCFFCharset(data, top[15], len(charStrings.objects))
	if err != nil {
		return nil, err
	}
	codeToGID, err := parseCFFCodeToGID(data, top[16], len(charStrings.objects), charset)
	if err != nil {
		return nil, err
	}
	out := map[uint32]string{}
	for code, gid := range codeToGID {
		if code < 0 || code > 255 || gid < 0 || gid >= len(charset) {
			continue
		}
		name := cffString(charset[gid], stringIndex.objects)
		if text := encoding.GlyphToText(name); text != "" {
			out[uint32(code)] = text
		}
	}
	return out, nil
}

func parseCFFIndex(data []byte, offset int) (cffIndex, error) {
	count, ok := u16(data, offset)
	if !ok {
		return cffIndex{}, ioBoundsError("CFF INDEX")
	}
	offset += 2
	if count == 0 {
		return cffIndex{next: offset}, nil
	}
	if offset >= len(data) {
		return cffIndex{}, ioBoundsError("CFF INDEX offSize")
	}
	offSize := int(data[offset])
	offset++
	if offSize < 1 || offSize > 4 || int(count) > 65535 ||
		offset+(int(count)+1)*offSize > len(data) {
		return cffIndex{}, errors.New("invalid CFF INDEX offsets")
	}
	offsets := make([]int, int(count)+1)
	for i := range offsets {
		var value uint64
		for _, b := range data[offset+i*offSize : offset+(i+1)*offSize] {
			value = value<<8 | uint64(b)
		}
		if value < 1 || value > uint64(len(data))+1 {
			return cffIndex{}, errors.New("invalid CFF INDEX object offset")
		}
		offsets[i] = int(value)
	}
	dataStart := offset + len(offsets)*offSize
	dataEnd := dataStart + offsets[len(offsets)-1] - 1
	if dataEnd < dataStart || dataEnd > len(data) {
		return cffIndex{}, ioBoundsError("CFF INDEX data")
	}
	objects := make([][]byte, count)
	for i := range objects {
		start := dataStart + offsets[i] - 1
		end := dataStart + offsets[i+1] - 1
		if start > end || end > dataEnd {
			return cffIndex{}, errors.New("unordered CFF INDEX offsets")
		}
		objects[i] = data[start:end]
	}
	return cffIndex{objects: objects, next: dataEnd}, nil
}

func parseCFFDict(data []byte) map[int]int {
	out := map[int]int{}
	var stack []int
	for i := 0; i < len(data); {
		b := data[i]
		i++
		switch {
		case b >= 32 && b <= 246:
			stack = append(stack, int(b)-139)
		case b >= 247 && b <= 250 && i < len(data):
			stack = append(stack, (int(b)-247)*256+int(data[i])+108)
			i++
		case b >= 251 && b <= 254 && i < len(data):
			stack = append(stack, -(int(b)-251)*256-int(data[i])-108)
			i++
		case b == 28 && i+2 <= len(data):
			// CFF DICT short integers are signed two's-complement values.
			stack = append(stack, int(int16(binary.BigEndian.Uint16(data[i:i+2])))) //nolint:gosec
			i += 2
		case b == 29 && i+4 <= len(data):
			// CFF DICT long integers are signed two's-complement values.
			stack = append(stack, int(int32(binary.BigEndian.Uint32(data[i:i+4])))) //nolint:gosec
			i += 4
		case b == 30:
			for i < len(data) {
				nibble := data[i]
				i++
				if nibble&0x0f == 0x0f || nibble>>4 == 0x0f {
					break
				}
			}
		case b <= 21:
			op := int(b)
			if b == 12 && i < len(data) {
				op = 1200 + int(data[i])
				i++
			}
			if len(stack) > 0 {
				out[op] = stack[len(stack)-1]
			}
			stack = stack[:0]
		default:
			stack = stack[:0]
		}
	}
	return out
}

func parseCFFCharset(data []byte, offset, glyphs int) ([]int, error) {
	charset := make([]int, glyphs)
	if glyphs == 0 {
		return charset, nil
	}
	if offset == 0 {
		// ISOAdobe's first 229 glyphs use matching standard string IDs.
		for gid := 1; gid < glyphs && gid < 229; gid++ {
			charset[gid] = gid
		}
		return charset, nil
	}
	if offset == 1 || offset == 2 {
		return nil, errors.New("CFF Expert predefined charsets are not Unicode-decodable")
	}
	if offset < 0 || offset >= len(data) {
		return nil, errors.New("invalid CFF charset offset")
	}
	format := data[offset]
	offset++
	gid := 1
	switch format {
	case 0:
		for gid < glyphs {
			sid, ok := u16(data, offset)
			if !ok {
				return nil, ioBoundsError("CFF charset")
			}
			charset[gid] = int(sid)
			offset += 2
			gid++
		}
	case 1, 2:
		for gid < glyphs {
			first, ok := u16(data, offset)
			if !ok {
				return nil, ioBoundsError("CFF charset range")
			}
			offset += 2
			var left int
			if format == 1 {
				if offset >= len(data) {
					return nil, ioBoundsError("CFF charset range")
				}
				left = int(data[offset])
				offset++
			} else {
				value, ok := u16(data, offset)
				if !ok {
					return nil, ioBoundsError("CFF charset range")
				}
				left = int(value)
				offset += 2
			}
			for n := 0; n <= left && gid < glyphs; n++ {
				charset[gid] = int(first) + n
				gid++
			}
		}
	default:
		return nil, fmt.Errorf("unsupported CFF charset format %d", format)
	}
	return charset, nil
}

func parseCFFCodeToGID(data []byte, offset, glyphs int, charset []int) (map[int]int, error) {
	out := map[int]int{}
	if offset == 0 {
		// Match CFF StandardEncoding text to the font's charset. Building the
		// reverse table from names also handles non-ASCII StandardEncoding
		// slots without baking a second code-to-SID table into the package.
		glyphByText := map[string]int{}
		for gid, sid := range charset {
			name := cffString(sid, nil)
			if text := encoding.GlyphToText(name); text != "" {
				if _, exists := glyphByText[text]; !exists {
					glyphByText[text] = gid
				}
			}
		}
		standard := encoding.New("StandardEncoding", nil)
		for code := range 256 {
			if gid, ok := glyphByText[standard.Decode(byte(code))]; ok {
				out[code] = gid
			}
		}
		return out, nil
	}
	if offset == 1 {
		return nil, errors.New("CFF ExpertEncoding is not Unicode-decodable")
	}
	if offset < 0 || offset >= len(data) {
		return nil, errors.New("invalid CFF Encoding offset")
	}
	format := data[offset]
	offset++
	switch format & 0x7f {
	case 0:
		if offset >= len(data) {
			return nil, ioBoundsError("CFF Encoding")
		}
		count := int(data[offset])
		offset++
		if offset+count > len(data) {
			return nil, ioBoundsError("CFF Encoding codes")
		}
		for i, code := range data[offset : offset+count] {
			gid := i + 1
			if gid < glyphs {
				out[int(code)] = gid
			}
		}
		offset += count
	case 1:
		if offset >= len(data) {
			return nil, ioBoundsError("CFF Encoding ranges")
		}
		ranges := int(data[offset])
		offset++
		gid := 1
		for range ranges {
			if offset+2 > len(data) {
				return nil, ioBoundsError("CFF Encoding range")
			}
			first, left := int(data[offset]), int(data[offset+1])
			offset += 2
			for n := 0; n <= left && gid < glyphs; n++ {
				out[first+n] = gid
				gid++
			}
		}
	default:
		return nil, fmt.Errorf("unsupported CFF Encoding format %d", format&0x7f)
	}
	if format&0x80 != 0 {
		if offset >= len(data) {
			return nil, ioBoundsError("CFF Encoding supplements")
		}
		supplements := int(data[offset])
		offset++
		for range supplements {
			if offset+3 > len(data) {
				return nil, ioBoundsError("CFF Encoding supplement")
			}
			code := int(data[offset])
			sid := int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
			offset += 3
			for gid, value := range charset {
				if value == sid {
					out[code] = gid
					break
				}
			}
		}
	}
	return out, nil
}

func cffString(sid int, custom [][]byte) string {
	if sid >= 391 {
		index := sid - 391
		if index >= 0 && index < len(custom) {
			return string(custom[index])
		}
		return ""
	}
	if sid >= 0 && sid < len(cffStandardStrings) {
		return cffStandardStrings[sid]
	}
	return ""
}

// The CFF standard strings through SID 228 are the complete ISOAdobe charset.
// Higher predefined SIDs are expert/style/metadata names; custom character
// names are read from the font's String INDEX.
var cffStandardStrings = []string{
	".notdef", "space", "exclam", "quotedbl", "numbersign", "dollar", "percent", "ampersand",
	"quoteright", "parenleft", "parenright", "asterisk", "plus", "comma", "hyphen", "period",
	"slash", "zero", "one", "two", "three", "four", "five", "six", "seven", "eight",
	"nine", "colon", "semicolon", "less", "equal", "greater", "question", "at",
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	"bracketleft", "backslash", "bracketright", "asciicircum", "underscore", "quoteleft",
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"braceleft", "bar", "braceright", "asciitilde",
	"exclamdown", "cent", "sterling", "fraction", "yen", "florin", "section", "currency",
	"quotesingle", "quotedblleft", "guillemotleft", "guilsinglleft", "guilsinglright", "fi", "fl",
	"endash", "dagger", "daggerdbl", "periodcentered", "paragraph", "bullet", "quotesinglbase",
	"quotedblbase", "quotedblright", "guillemotright", "ellipsis", "perthousand", "questiondown",
	"grave", "acute", "circumflex", "tilde", "macron", "breve", "dotaccent", "dieresis",
	"ring", "cedilla", "hungarumlaut", "ogonek", "caron", "emdash", "AE", "ordfeminine",
	"Lslash", "Oslash", "OE", "ordmasculine", "ae", "dotlessi", "lslash", "oslash", "oe",
	"germandbls", "onesuperior", "logicalnot", "mu", "trademark", "Eth", "onehalf", "plusminus",
	"Thorn", "onequarter", "divide", "brokenbar", "degree", "thorn", "threequarters",
	"twosuperior", "registered", "minus", "eth", "multiply", "threesuperior", "copyright",
	"Aacute", "Acircumflex", "Adieresis", "Agrave", "Aring", "Atilde", "Ccedilla", "Eacute",
	"Ecircumflex", "Edieresis", "Egrave", "Iacute", "Icircumflex", "Idieresis", "Igrave",
	"Ntilde", "Oacute", "Ocircumflex", "Odieresis", "Ograve", "Otilde", "Scaron", "Uacute",
	"Ucircumflex", "Udieresis", "Ugrave", "Yacute", "Ydieresis", "Zcaron", "aacute",
	"acircumflex", "adieresis", "agrave", "aring", "atilde", "ccedilla", "eacute",
	"ecircumflex", "edieresis", "egrave", "iacute", "icircumflex", "idieresis", "igrave",
	"ntilde", "oacute", "ocircumflex", "odieresis", "ograve", "otilde", "scaron", "uacute",
	"ucircumflex", "udieresis", "ugrave", "yacute", "ydieresis", "zcaron",
}
