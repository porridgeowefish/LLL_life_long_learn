// Package render is a memory-only PDFium/WASM page renderer. No host directory,
// document JavaScript, forms, URLs or external programs are exposed.
package render

import (
	"bytes"
	"context"
	"errors"
	pdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"image/png"
	"io"
	"sync"
	"time"
)

var gate = make(chan struct{}, 1)
var poolOnce sync.Once
var pool pdfium.Pool
var poolErr error

func Render(ctx context.Context, document []byte, page int) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(document) > 100<<20 || page < 1 {
		return nil, errors.New("render-input-limit")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	poolOnce.Do(func() {
		pool, poolErr = webassembly.Init(webassembly.Config{Context: context.Background(), MaxTotal: 1, MaxIdle: 0, MinIdle: 0, ReuseWorkers: false, FSConfig: wazero.NewFSConfig(), Stdout: io.Discard, Stderr: io.Discard, RuntimeConfig: wazero.NewRuntimeConfig().WithMemoryLimitPages(4096).WithCloseOnContextDone(true).WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling)})
	})
	if poolErr != nil {
		return nil, errors.New("renderer-unavailable")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	instance, e := pool.GetInstanceWithContext(ctx)
	if e != nil {
		return nil, e
	}
	finished := make(chan struct{})
	watched := make(chan bool, 1)
	go func() {
		select {
		case <-ctx.Done():
			_ = instance.Kill()
			watched <- true
		case <-finished:
			watched <- false
		}
	}()
	var cleanup func()
	defer func() {
		close(finished)
		killed := <-watched
		if !killed {
			if cleanup != nil {
				cleanup()
			}
			_ = instance.Close()
		}
	}()
	doc, e := instance.FPDF_LoadMemDocument(&requests.FPDF_LoadMemDocument{Data: &document})
	if e != nil {
		return nil, e
	}
	cleanup = func() { _, _ = instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document}) }
	result, e := instance.RenderPageInPixels(&requests.RenderPageInPixels{Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: page - 1}}, Width: 2400, Height: 2400, RenderForm: false})
	if e != nil {
		return nil, e
	}
	previousCleanup := cleanup
	cleanup = func() { result.Cleanup(); previousCleanup() }
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if result.Result.Width > 2400 || result.Result.Height > 2400 || result.Result.Width*result.Result.Height > 6000000 {
		return nil, errors.New("render-image-limit")
	}
	var b bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if e = encoder.Encode(&b, result.Result.RenderedImage); e != nil {
		return nil, e
	}
	if b.Len() > 8<<20 {
		return nil, errors.New("render-image-limit")
	}
	return b.Bytes(), nil
}
