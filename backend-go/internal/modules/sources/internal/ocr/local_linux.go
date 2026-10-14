//go:build linux

package ocr

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

var linuxProbe struct {
	sync.Mutex
	checked   time.Time
	available bool
}

func localPageProvider() (PageProvider, bool) {
	id, model := localIdentity()
	p := PageProvider{ID: id, Kind: "system-local", Model: model}
	if testing.Testing() && os.Getenv("LLL_TEST_LOCAL_OCR") != "1" {
		return p, false
	}
	linuxProbe.Lock()
	defer linuxProbe.Unlock()
	if time.Since(linuxProbe.checked) > 5*time.Second {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, "tesseract", "--list-langs").Output()
		linuxProbe.available = false
		if err == nil {
			for _, lang := range strings.Fields(string(output)) {
				if lang == "chi_sim" {
					linuxProbe.available = true
				}
			}
		}
		linuxProbe.checked = time.Now()
	}
	return p, linuxProbe.available
}

func extractLocalPage(ctx context.Context, image []byte) (PageResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", "chi_sim", "--psm", "3")
	cmd.Stdin = bytes.NewReader(image)
	output, err := cmd.Output()
	if err != nil {
		return PageResult{}, errors.New("local-ocr-unavailable")
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return PageResult{}, errors.New("local-ocr-no-text")
	}
	return PageResult{Content: normalizeLocalText(text), Known: true, Tokens: map[string]int{"inputTokens": 0, "outputTokens": 0}}, nil
}
