//go:build windows

package speech

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

//go:embed speech_windows.ps1
var windowsScript []byte

func run(ctx context.Context, mode, language string, wave []byte) (Result, error) {
	dir, err := os.MkdirTemp("", "lll-speech-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)
	script := filepath.Join(dir, "speech.ps1")
	if err = os.WriteFile(script, append([]byte{0xef, 0xbb, 0xbf}, windowsScript...), 0600); err != nil {
		return Result{}, err
	}
	file := filepath.Join(dir, "speech.wav")
	if wave != nil {
		if err = os.WriteFile(file, wave, 0600); err != nil {
			return Result{}, err
		}
	}
	input, _ := json.Marshal(map[string]string{"mode": mode, "language": language, "path": file})
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return Result{}, err
	}
	var result Result
	err = json.Unmarshal(bytes.TrimPrefix(output, []byte{0xef, 0xbb, 0xbf}), &result)
	return result, err
}
