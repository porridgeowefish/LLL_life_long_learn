package filter

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"io"
	"sync"
)

// Releaser is implemented by readers that hold pooled resources. A caller
// that has finished with a stream calls Release to return them; a reader
// that is never released is simply garbage collected. Reads after Release
// fail.
type Releaser interface {
	Release()
}

// Decompressors are pooled. Each holds a 32 KiB window plus a 4 KiB read
// buffer, and extraction opens a stream for every page, font program, and
// CMap, so the windows were among the largest transient allocations of a
// document — the ones the runtime pays most to hand out and reclaim.
var (
	bufioPool = sync.Pool{New: func() any { return bufio.NewReader(nil) }}
	zlibPool  = sync.Pool{New: func() any {
		zr, err := zlib.NewReader(bytes.NewReader(emptyZlib))
		if err != nil {
			panic(err) // emptyZlib is a constant, valid stream
		}
		return zr
	}}
	flatePool = sync.Pool{New: func() any { return flate.NewReader(bytes.NewReader(nil)) }}
)

// emptyZlib is a complete zlib stream with no payload: it constructs a
// pooled reader, which is then Reset onto each real stream.
var emptyZlib = []byte{0x78, 0x9c, 0x03, 0x00, 0x00, 0x00, 0x00, 0x01}

var errReleased = errors.New("read from released FlateDecode stream")

// flateReader decodes zlib data, falling back to raw deflate for the
// real-world files whose generators omit the zlib header. The header is
// checked with the rules compress/zlib applies (RFC 1950 §2.2); a
// stream that would need a preset dictionary takes the raw path, as it
// did when zlib.NewReader refused it.
type flateReader struct {
	br *bufio.Reader
	zr io.ReadCloser // pooled zlib reader, when the header is valid
	fr io.ReadCloser // pooled raw deflate reader otherwise
	r  io.Reader     // the active one; nil once released
}

func newFlateReader(src io.Reader) (*flateReader, error) {
	br, _ := bufioPool.Get().(*bufio.Reader)
	br.Reset(src)
	head, err := br.Peek(2)
	if len(head) < 2 {
		br.Reset(nil)
		bufioPool.Put(br)
		if err == nil || errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	f := &flateReader{br: br}
	if zlibHeaderOK(head[0], head[1]) {
		zr, _ := zlibPool.Get().(io.ReadCloser)
		if err := zr.(zlib.Resetter).Reset(br, nil); err != nil {
			f.zr = zr
			f.Release()
			return nil, err
		}
		f.zr, f.r = zr, zr
		return f, nil
	}
	fr, _ := flatePool.Get().(io.ReadCloser)
	_ = fr.(flate.Resetter).Reset(br, nil) // always nil for a non-nil reader
	f.fr, f.r = fr, fr
	return f, nil
}

func zlibHeaderOK(cmf, flg byte) bool {
	const deflate, maxWindow = 8, 7
	return cmf&0x0f == deflate &&
		cmf>>4 <= maxWindow &&
		(uint(cmf)<<8|uint(flg))%31 == 0 &&
		flg&0x20 == 0 // FDICT: a preset dictionary we cannot supply
}

func (f *flateReader) Read(p []byte) (int, error) {
	if f.r == nil {
		return 0, errReleased
	}
	return f.r.Read(p)
}

// Release returns the decompressor and buffer to their pools. The pooled
// readers are reset onto empty sources first so they do not keep the
// document's bytes reachable.
func (f *flateReader) Release() {
	if f.br == nil {
		return
	}
	f.r = nil
	f.br.Reset(nil)
	bufioPool.Put(f.br)
	f.br = nil
	if f.zr != nil {
		_ = f.zr.(zlib.Resetter).Reset(bytes.NewReader(emptyZlib), nil)
		zlibPool.Put(f.zr)
		f.zr = nil
	}
	if f.fr != nil {
		_ = f.fr.(flate.Resetter).Reset(bytes.NewReader(nil), nil)
		flatePool.Put(f.fr)
		f.fr = nil
	}
}
