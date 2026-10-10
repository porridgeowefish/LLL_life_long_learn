package object

import (
	"fmt"
	"io"

	"github.com/giraffesyo/pdf/internal/filter"
)

// applyFilters wraps body with the stream's decode chain from /Filter and
// /DecodeParms (each of which may be a single value or an array). The
// returned release function hands pooled decoder state back once the
// reader is no longer used; it is nil when the chain holds none.
//
// With untilImage set the chain stops at the first image codec, which is
// returned with its parameters rather than applied; otherwise an image
// codec in the chain is an error, as it is for any non-image stream.
func (r *Reader) applyFilters(body io.Reader, d dict, untilImage bool) (io.Reader, func(), ImageFilter, error) {
	filters := filterNames(d)
	if len(filters) == 0 {
		return body, nil, ImageFilter{}, nil
	}
	if len(filters) > maxFilterChain {
		return nil, nil, ImageFilter{}, fmt.Errorf("pdf: filter chain of %d exceeds limit", len(filters))
	}
	parms := decodeParms(r, d, len(filters))
	rd := body
	var releasers []filter.Releaser
	release := func() {
		for _, rel := range releasers {
			rel.Release()
		}
	}
	for i, name := range filters {
		if codec, ok := filter.ImageCodec(name); ok && untilImage {
			// Only the last filter may be an image codec: nothing decodes
			// the codec's output into a further filter's input.
			if i != len(filters)-1 {
				release()
				return nil, nil, ImageFilter{}, fmt.Errorf("pdf: image codec /%s is not the last stream filter", codec)
			}
			if len(releasers) == 0 {
				release = nil
			}
			return rd, release, ImageFilter{Name: codec, Parms: decodeParmValue(r, d, i)}, nil
		}
		var err error
		rd, err = filter.Apply(rd, name, parms[i])
		if err != nil {
			release()
			return nil, nil, ImageFilter{}, err
		}
		if rel, ok := rd.(filter.Releaser); ok {
			releasers = append(releasers, rel)
		}
	}
	if len(releasers) == 0 {
		return rd, nil, ImageFilter{}, nil
	}
	return rd, release, ImageFilter{}, nil
}

// ImageFilter is the image codec an image stream's data is still encoded
// with after ImageReader applied the general-purpose filters, together
// with the codec's /DecodeParms dictionary (null when absent). Name is the
// codec's full name — DCTDecode, JPXDecode, CCITTFaxDecode or JBIG2Decode —
// even when the stream abbreviates it, or "" when the data is fully
// decoded samples.
type ImageFilter struct {
	Name  string
	Parms Value
}

// decodeParmValue returns the i-th filter's /DecodeParms dictionary as a
// Value, so callers can resolve entries the filter layer does not model
// (a /JBIG2Globals stream, for example).
func decodeParmValue(r *Reader, d dict, i int) Value {
	pv, ok := d["DecodeParms"]
	if !ok {
		pv = d["DP"]
	}
	switch x := pv.(type) {
	case dict:
		if i == 0 {
			return Value{r: r, data: x}
		}
	case []Value:
		if i < len(x) {
			return x[i]
		}
	}
	return Value{}
}

// filterNames returns the /Filter (or abbreviated /F) entries in order.
func filterNames(d dict) []string {
	v, ok := d["Filter"]
	if !ok {
		v, ok = d["F"]
		if !ok {
			return nil
		}
	}
	switch x := v.(type) {
	case name:
		return []string{string(x)}
	case []Value:
		names := make([]string, 0, len(x))
		for _, e := range x {
			names = append(names, e.Name())
		}
		return names
	default:
		return nil
	}
}

// decodeParms returns per-filter parameters aligned with the filter list.
func decodeParms(r *Reader, d dict, n int) []filter.Params {
	out := make([]filter.Params, n)
	pv, ok := d["DecodeParms"]
	if !ok {
		pv = d["DP"]
	}
	switch x := pv.(type) {
	case dict:
		if n > 0 {
			out[0] = paramsFromDict(r, x)
		}
	case []Value:
		for i := 0; i < n && i < len(x); i++ {
			if pd, ok := x[i].resolve().data.(dict); ok {
				out[i] = paramsFromDict(r, pd)
			}
		}
	}
	return out
}

func paramsFromDict(r *Reader, d dict) filter.Params {
	p := filter.Params{
		Predictor:        int(r.asInt(d["Predictor"])),
		Colors:           int(r.asInt(d["Colors"])),
		BitsPerComponent: int(r.asInt(d["BitsPerComponent"])),
		Columns:          int(r.asInt(d["Columns"])),
	}
	if ec, ok := d["EarlyChange"].(int64); ok && ec == 0 {
		p.NoEarlyChange = true
	}
	return p
}
