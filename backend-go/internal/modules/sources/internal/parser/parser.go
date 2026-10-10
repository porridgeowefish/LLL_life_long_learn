// Package parser extracts source text locally. It never executes uploaded code,
// follows external relationships, or calls an assistant/OCR service.
package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxContentBytes  = 4 << 20
	MaxInputBytes    = 50 << 20
	truncationMarker = "\n\n[内容已截断：资料文本超过提取上限。]\n"
)

type Result struct {
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	Parser    string `json:"parser"`
}

type parseError struct {
	code  string
	cause error
}

func (e *parseError) Error() string        { return e.code + ": " + e.cause.Error() }
func (e *parseError) Unwrap() error        { return e.cause }
func failure(code string, err error) error { return &parseError{code, err} }
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "parse_cancelled"
	}
	var e *parseError
	if errors.As(err, &e) {
		return e.code
	}
	return "parse_failed"
}

var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".rst": true, ".json": true, ".jsonl": true,
	".yaml": true, ".yml": true, ".toml": true, ".xml": true, ".html": true, ".htm": true, ".css": true,
	".go": true, ".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".rs": true, ".rb": true, ".php": true,
	".swift": true, ".kt": true, ".kts": true, ".sql": true, ".r": true, ".m": true, ".scala": true, ".lua": true,
}

func extension(filename string) string {
	return strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
}
func Supported(filename string) bool {
	ext := extension(filename)
	return textExtensions[ext] || ext == ".pdf" || ext == ".docx" || ext == ".pptx" || ext == ".xlsx" || ext == ".csv" || ext == ".tsv"
}

// Extract reads a regular original file. Content includes a truncation marker
// within MaxContentBytes when the output cap is reached. Invalid, encrypted or
// textless documents fail explicitly rather than becoming ready with empty text.
func Extract(ctx context.Context, path, filename string) (result Result, err error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	defer func() {
		if caught := recover(); caught != nil {
			result = Result{}
			err = failure("invalid_document", fmt.Errorf("document parser panic: %v", caught))
		}
	}()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	ext := extension(filename)
	if !Supported(filename) {
		if ext == ".doc" || ext == ".ppt" || ext == ".xls" {
			return Result{}, failure("unsupported_legacy_format", errors.New("save the document as DOCX, PPTX or XLSX"))
		}
		return Result{}, failure("unsupported_format", errors.New("no local text parser for this format"))
	}
	f, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return Result{}, err
	}
	if !stat.Mode().IsRegular() {
		return Result{}, failure("invalid_document", errors.New("source must be a regular file"))
	}
	if stat.Size() > MaxInputBytes {
		return Result{}, failure("resource_limit", errors.New("source exceeds 50 MiB"))
	}
	out := &boundedText{ctx: ctx}
	switch ext {
	case ".pdf":
		result.Parser = "pdf"
		err = extractPDF(ctx, f, stat.Size(), out)
	case ".docx", ".pptx", ".xlsx":
		result.Parser = strings.TrimPrefix(ext, ".")
		err = extractOffice(ctx, f, stat.Size(), ext, out)
	case ".csv", ".tsv":
		result.Parser = "csv"
		err = extractCSV(ctx, f, ext == ".tsv", out)
	default:
		result.Parser = "text"
		var text string
		text, err = decodeText(ctx, f)
		if err == nil {
			_, err = out.Write([]byte(text))
		}
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if err != nil && !errors.Is(err, errOutputFull) {
		return Result{}, err
	}
	result.Content, result.Truncated = out.String(), out.truncated
	if strings.TrimSpace(result.Content) == "" {
		return Result{}, failure("no_extractable_text", errors.New("document has no extractable text; scanned or outlined pages require OCR"))
	}
	return result, nil
}

var errOutputFull = errors.New("extracted output cap reached")

type boundedText struct {
	ctx       context.Context
	b         bytes.Buffer
	truncated bool
}

func (b *boundedText) Write(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if b.truncated {
		return 0, errOutputFull
	}
	remaining := MaxContentBytes - len(truncationMarker) - b.b.Len()
	if len(p) <= remaining {
		return b.b.Write(p)
	}
	n := remaining
	for n > 0 && !utf8.Valid(p[:n]) {
		n--
	}
	_, _ = b.b.Write(p[:n])
	b.truncated = true
	return n, errOutputFull
}
func (b *boundedText) String() string {
	if b.truncated {
		return b.b.String() + truncationMarker
	}
	return b.b.String()
}
func (b *boundedText) text(s string) error { _, err := b.Write([]byte(s)); return err }

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type contextReaderAt struct {
	ctx context.Context
	r   io.ReaderAt
}

func (r contextReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.ReadAt(p, offset)
}
