//go:build darwin

package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

type visionResult struct {
	Available bool   `json:"available"`
	Text      string `json:"text"`
}

func runVision(ctx context.Context, image []byte) (visionResult, error) {
	request := map[string]string{"mode": "probe"}
	if image != nil {
		dir, err := os.MkdirTemp("", "lll-vision-")
		if err != nil {
			return visionResult{}, err
		}
		defer os.RemoveAll(dir)
		file := filepath.Join(dir, "page.png")
		if err := os.WriteFile(file, image, 0600); err != nil {
			return visionResult{}, err
		}
		request["mode"], request["path"] = "recognize", file
	}
	input, _ := json.Marshal(request)
	cmd := exec.CommandContext(ctx, filepath.Join(paths.PROJECT_ROOT, "libexec", "lll-vision-ocr"))
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return visionResult{}, errors.New("local-ocr-unavailable")
	}
	var result visionResult
	if json.Unmarshal(output, &result) != nil || !result.Available {
		return visionResult{}, errors.New("local-ocr-unavailable")
	}
	return result, nil
}

func localPageProvider() (PageProvider, bool) {
	id, model := localIdentity()
	p := PageProvider{ID: id, Kind: "system-local", Model: model}
	if testing.Testing() && os.Getenv("LLL_TEST_LOCAL_OCR") != "1" {
		return p, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runVision(ctx, nil)
	return p, err == nil && result.Available
}

func extractLocalPage(ctx context.Context, image []byte) (PageResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	result, err := runVision(ctx, image)
	if err != nil {
		return PageResult{}, err
	}
	if strings.TrimSpace(result.Text) == "" {
		return PageResult{}, errors.New("local-ocr-no-text")
	}
	return PageResult{Content: normalizeLocalText(result.Text), Known: true, Tokens: map[string]int{"inputTokens": 0, "outputTokens": 0}}, nil
}
