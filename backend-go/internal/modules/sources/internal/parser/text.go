package parser

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func decodeText(ctx context.Context, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(contextReader{ctx, r}, MaxInputBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > MaxInputBytes {
		return "", failure("resource_limit", errors.New("text input exceeds 50 MiB"))
	}
	var decoder transform.Transformer
	switch {
	case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
		data = data[3:]
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}):
		decoder = unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder()
	case bytes.HasPrefix(data, []byte{0xfe, 0xff}):
		decoder = unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder()
	default:
		if !utf8.Valid(data) {
			decoder = simplifiedchinese.GB18030.NewDecoder()
		}
	}
	if decoder != nil {
		data, err = io.ReadAll(io.LimitReader(transform.NewReader(contextReader{ctx, bytes.NewReader(data)}, decoder), 3*MaxInputBytes+1))
		if err != nil {
			return "", failure("invalid_document", err)
		}
		if bytes.Contains(data, []byte("\ufffd")) {
			return "", failure("invalid_document", errors.New("unsupported or malformed text encoding"))
		}
	}
	if !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return "", failure("invalid_document", errors.New("text contains invalid encoding or binary NUL bytes"))
	}
	return string(data), nil
}

func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "<br>")
}
func tableRow(out *boundedText, cells []string) error {
	if err := out.text("| "); err != nil {
		return err
	}
	for i, cell := range cells {
		if i > 0 {
			if err := out.text(" | "); err != nil {
				return err
			}
		}
		if err := out.text(escapeCell(cell)); err != nil {
			return err
		}
	}
	return out.text(" |\n")
}
func extractCSV(ctx context.Context, r io.Reader, tsv bool, out *boundedText) error {
	text, err := decodeText(ctx, r)
	if err != nil {
		return err
	}
	reader := csv.NewReader(contextReader{ctx, strings.NewReader(text)})
	reader.FieldsPerRecord = -1
	if tsv {
		reader.Comma = '\t'
	}
	row := 0
	for {
		cells, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return failure("invalid_document", err)
		}
		if len(cells) > 10000 {
			return failure("resource_limit", errors.New("CSV row exceeds 10000 columns"))
		}
		if err := tableRow(out, cells); err != nil {
			return err
		}
		if row == 0 {
			rules := make([]string, len(cells))
			for i := range rules {
				rules[i] = "---"
			}
			if err := tableRow(out, rules); err != nil {
				return err
			}
		}
		row++
	}
}
