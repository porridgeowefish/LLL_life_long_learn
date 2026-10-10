package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	pdf "github.com/giraffesyo/pdf"
)

func extractPDF(ctx context.Context, f io.ReaderAt, size int64, out *boundedText) error {
	number := 0
	_, err := pdf.ExtractPages(ctx, contextReaderAt{ctx, f}, size, pdf.Options{
		Strict: true, Concurrency: 1,
		Limits: pdf.Limits{MaxStreamBytes: 16 << 20, MaxOperatorsPerPage: 1000000, MaxGlyphsPerPage: 200000, MaxFormDepth: 32, MaxImagesPerPage: 10000, MaxImageBytesPerPage: 16 << 20},
	}, func(page pdf.Page) error {
		number++
		text := page.Text()
		if strings.TrimSpace(text) == "" {
			return nil
		}
		if err := out.text(fmt.Sprintf("## 第 %d 页\n\n", number)); err != nil {
			return err
		}
		if err := out.text(text); err != nil {
			return err
		}
		return out.text("\n\n")
	})
	if err == nil || errors.Is(err, errOutputFull) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, pdf.ErrPasswordRequired) {
		return failure("encrypted_document", err)
	}
	var strict *pdf.StrictError
	if errors.As(err, &strict) {
		switch strict.Warning.Code {
		case pdf.WarningStreamLimit, pdf.WarningWorkLimit:
			return failure("resource_limit", err)
		case pdf.WarningUnsupported:
			return failure("unsupported_format", err)
		}
	}
	return failure("invalid_document", err)
}
