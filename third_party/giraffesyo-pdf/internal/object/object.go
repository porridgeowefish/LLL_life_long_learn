// Package object is a native PDF object layer: a lexer and parser for the
// COS object syntax (ISO 32000-1 §7.2–7.3), cross-reference tables and
// streams (§7.5), and a Value model for navigating the resulting object
// graph. It replaces the third-party parser the extractor was built on.
//
// The API is deliberately shaped for the extractor, not for parity with
// any library: navigation (Key, Index) is total and returns null on
// mismatch, because optional keys with spec-prescribed defaults are the
// common case; scalar access is comma-ok, the Go idiom for "present or
// not"; and genuine I/O failures surface as errors from NewReader,
// Page and Reader.
package object

import (
	"fmt"
	"io"
)

// Kind is the type of a PDF object.
type Kind int

// The PDF object kinds (ISO 32000-1 §7.3). The zero value is Null.
const (
	Null Kind = iota
	Bool
	Integer
	Real
	String
	Name
	Dict
	Array
	Stream
)

func (k Kind) String() string {
	switch k {
	case Bool:
		return "boolean"
	case Integer:
		return "integer"
	case Real:
		return "real"
	case String:
		return "string"
	case Name:
		return "name"
	case Dict:
		return "dictionary"
	case Array:
		return "array"
	case Stream:
		return "stream"
	default:
		return "null"
	}
}

// ref is an indirect reference (N G R).
type ref struct {
	num, gen int
}

// name distinguishes a PDF /Name from a string literal in the object graph.
type name string

// dict is a PDF dictionary. Keys are stored without the leading slash.
type dict map[string]any

// stream is a dictionary plus a locator for its raw (still-encoded) bytes.
type stream struct {
	d      dict
	offset int64 // absolute file offset of the first data byte
	length int64 // resolved /Length, or -1 if it must be recovered
	owner  ref   // enclosing indirect object, for decryption
}

// A Value is one node of the object graph. The zero Value is null. A Value
// carries a back-pointer to its Reader so Key/Index can resolve indirect
// references lazily.
type Value struct {
	r     *Reader
	data  any // nil | bool | int64 | float64 | string | name | dict | array | *stream | ref
	owner ref // object this value was parsed from, for decryption context
}

// Trailer returns the trailer of the file v belongs to, resolved through
// the same Reader as v — a clone's value reaches the clone's trailer.
func (v Value) Trailer() Value {
	if v.r == nil {
		return Value{}
	}
	return v.r.Trailer()
}

// ObjectNumber returns the indirect object number represented by v. Direct
// values have no object number.
func (v Value) ObjectNumber() (int, bool) {
	ref, ok := v.data.(ref)
	if !ok {
		return 0, false
	}
	return ref.num, true
}

// data holds the concrete Go representation:
//
//	nil        Null
//	bool       Bool
//	int64      Integer
//	float64    Real
//	string     String
//	name       Name
//	dict       Dict
//	[]Value    Array
//	*stream    Stream
//	ref        an unresolved indirect reference (resolved on access)

// Kind reports the object's type, resolving a top-level indirect reference.
func (v Value) Kind() Kind {
	v = v.resolve()
	switch v.data.(type) {
	case bool:
		return Bool
	case int64:
		return Integer
	case float64:
		return Real
	case string:
		return String
	case name:
		return Name
	case dict:
		return Dict
	case []Value:
		return Array
	case *stream:
		return Stream
	default:
		return Null
	}
}

// IsNull reports whether the value is the PDF null object (or absent).
func (v Value) IsNull() bool { return v.Kind() == Null }

// Bool returns the boolean value; ok is false for any other kind.
func (v Value) Bool() (b, ok bool) {
	if x, is := v.resolve().data.(bool); is {
		return x, true
	}
	return false, false
}

// Int64 returns an integer value; ok is false for any other kind. A Real
// is not coerced (use Float64 when either is acceptable).
func (v Value) Int64() (int64, bool) {
	if x, is := v.resolve().data.(int64); is {
		return x, true
	}
	return 0, false
}

// Float64 returns a numeric value, coercing Integer to float; ok is false
// for non-numeric kinds. PDF has a single "number" type split into
// Integer and Real, so most callers want this.
func (v Value) Float64() (float64, bool) {
	switch x := v.resolve().data.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

// Name returns the name without its leading slash, or "" for other kinds.
func (v Value) Name() string {
	if x, ok := v.resolve().data.(name); ok {
		return string(x)
	}
	return ""
}

// RawString returns the decoded bytes of a string object, or "" otherwise.
func (v Value) RawString() string {
	if x, ok := v.resolve().data.(string); ok {
		return x
	}
	return ""
}

// Len returns the element count of an array or the entry count of a
// dictionary/stream; 0 for other kinds.
func (v Value) Len() int {
	switch x := v.resolve().data.(type) {
	case []Value:
		return len(x)
	case dict:
		return len(x)
	case *stream:
		return len(x.d)
	default:
		return 0
	}
}

// Key looks up a dictionary (or stream-dictionary) entry, resolving the
// receiver if it is an indirect reference. It returns null when the
// receiver is not a dictionary or the key is absent.
func (v Value) Key(key string) Value {
	v = v.resolve()
	var d dict
	switch x := v.data.(type) {
	case dict:
		d = x
	case *stream:
		d = x.d
	default:
		return Value{}
	}
	child, ok := d[key]
	if !ok {
		return Value{}
	}
	return Value{r: v.r, data: child, owner: v.owner}
}

// Index returns the i-th element of an array, resolving the receiver if
// needed. It returns null when out of range or not an array.
func (v Value) Index(i int) Value {
	v = v.resolve()
	arr, ok := v.data.([]Value)
	if !ok || i < 0 || i >= len(arr) {
		return Value{}
	}
	return arr[i]
}

// Keys returns the dictionary's keys in unspecified order, or nil.
func (v Value) Keys() []string {
	v = v.resolve()
	var d dict
	switch x := v.data.(type) {
	case dict:
		d = x
	case *stream:
		d = x.d
	default:
		return nil
	}
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	return keys
}

// resolve follows an indirect reference to the object it names, iterating
// through reference chains with a hop cap so a cyclic ref graph cannot
// loop. Non-reference values are returned unchanged.
func (v Value) resolve() Value {
	for hops := 0; ; hops++ {
		r, ok := v.data.(ref)
		if !ok {
			return v
		}
		if hops >= maxRefHops || v.r == nil {
			return Value{}
		}
		obj, err := v.r.object(r.num, r.gen)
		if err != nil {
			v.r.setErr(err)
			return Value{}
		}
		v = Value{r: v.r, data: obj, owner: ref{r.num, r.gen}}
	}
}

// Reader returns a reader over the stream's fully decoded bytes
// (decryption then filter chain). It errors for non-stream values and on
// decode failure.
func (v Value) Reader() (io.ReadCloser, error) {
	v = v.resolve()
	s, ok := v.data.(*stream)
	if !ok {
		return nil, fmt.Errorf("pdf: Reader on %s, not a stream", v.Kind())
	}
	return v.r.streamReader(s)
}

// ImageReader returns a reader over an image stream's bytes after
// decryption and the general-purpose filters, stopping at the image
// codec (DCTDecode, JPXDecode, CCITTFaxDecode or JBIG2Decode) if the
// chain ends in one: the codec and its parameters are returned for the
// caller to decode, and the reader yields the still-encoded image data.
// Without an image codec the reader yields raw samples and the returned
// ImageFilter is zero. It errors for non-stream values and on decode
// failure.
func (v Value) ImageReader() (io.ReadCloser, ImageFilter, error) {
	v = v.resolve()
	s, ok := v.data.(*stream)
	if !ok {
		return nil, ImageFilter{}, fmt.Errorf("pdf: ImageReader on %s, not a stream", v.Kind())
	}
	return v.r.imageReader(s)
}
