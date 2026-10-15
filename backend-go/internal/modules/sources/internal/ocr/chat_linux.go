//go:build linux

package ocr

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

func extractChatImage(ctx context.Context, image []byte, language string) (PageResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	lang := "chi_sim+eng"
	if language == "en-US" {
		lang = "eng"
	}
	cmd := exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", lang, "--psm", "3")
	cmd.Stdin = bytes.NewReader(image)
	out, err := cmd.Output()
	if err != nil {
		return PageResult{}, errors.New("local-ocr-unavailable")
	}
	return PageResult{Content: normalizeLocalText(string(out))}, nil
}
