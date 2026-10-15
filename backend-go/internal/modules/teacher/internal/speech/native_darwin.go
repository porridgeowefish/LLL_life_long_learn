//go:build darwin

package speech

import (
	"bytes"
	"context"
	"encoding/json"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
	"os"
	"os/exec"
	"path/filepath"
)

func run(ctx context.Context, mode, language string, wave []byte) (Result, error) {
	dir, err := os.MkdirTemp("", "lll-speech-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "speech.wav")
	if wave != nil {
		if err = os.WriteFile(file, wave, 0600); err != nil {
			return Result{}, err
		}
	}
	input, _ := json.Marshal(map[string]string{"mode": mode, "language": language, "path": file})
	cmd := exec.CommandContext(ctx, filepath.Join(paths.PROJECT_ROOT, "libexec", "lll-speech"))
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return Result{}, err
	}
	var result Result
	err = json.Unmarshal(output, &result)
	return result, err
}
