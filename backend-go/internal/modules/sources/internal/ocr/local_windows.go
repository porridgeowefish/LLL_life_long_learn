//go:build windows

package ocr

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

//go:embed windows_ocr.ps1
var windowsOCRScript []byte

var localProbe struct {
	sync.Once
	available bool
}

func localPageProvider() (PageProvider, bool) {
	// Unit tests stay isolated from the user's system OCR installation.
	if testing.Testing() && os.Getenv("LLL_TEST_WINDOWS_OCR") != "1" {
		return PageProvider{}, false
	}
	localProbe.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := runWindowsOCR(ctx, map[string]string{"mode": "probe"}, nil)
		localProbe.available = err == nil && result.Available
	})
	return PageProvider{ID: "windows-local-ocr", Kind: "windows-local", Model: "windows-zh-Hans"}, localProbe.available
}

type localOCRResult struct {
	Available bool   `json:"available"`
	Text      string `json:"text"`
}

func runWindowsOCR(ctx context.Context, request map[string]string, image []byte) (localOCRResult, error) {
	dir, err := os.MkdirTemp("", "lll-ocr-")
	if err != nil {
		return localOCRResult{}, err
	}
	defer os.RemoveAll(dir)
	script := filepath.Join(dir, "recognize.ps1")
	// BOM is required by Windows PowerShell 5.1 if future script text is non-ASCII.
	if err = os.WriteFile(script, append([]byte{0xef, 0xbb, 0xbf}, windowsOCRScript...), 0600); err != nil {
		return localOCRResult{}, err
	}
	if image != nil {
		path := filepath.Join(dir, "page.png")
		if err = os.WriteFile(path, image, 0600); err != nil {
			return localOCRResult{}, err
		}
		request["path"] = path
	}
	input, err := json.Marshal(request)
	if err != nil {
		return localOCRResult{}, err
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return localOCRResult{}, errors.New("windows-ocr-unavailable")
	}
	var result localOCRResult
	if json.Unmarshal(bytes.TrimPrefix(output, []byte{0xef, 0xbb, 0xbf}), &result) != nil {
		return result, errors.New("windows-ocr-invalid-output")
	}
	return result, nil
}

func extractLocalPage(ctx context.Context, image []byte) (PageResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	result, err := runWindowsOCR(ctx, map[string]string{"mode": "recognize"}, image)
	if err != nil {
		return PageResult{}, err
	}
	if strings.TrimSpace(result.Text) == "" {
		return PageResult{}, errors.New("windows-ocr-no-text")
	}
	return PageResult{Content: normalizeLocalText(result.Text), Known: true, Tokens: map[string]int{"inputTokens": 0, "outputTokens": 0}}, nil
}
