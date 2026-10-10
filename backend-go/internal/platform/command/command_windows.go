//go:build windows

package command

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

func nativeCommand(ctx context.Context, bin string, args []string) *exec.Cmd {
	resolved, err := exec.LookPath(bin)
	if err != nil {
		resolved = bin
	}
	ext := strings.ToLower(filepath.Ext(resolved))
	var cmd *exec.Cmd
	if ext == ".cmd" || ext == ".bat" {
		quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
		parts := []string{"&", quote(resolved)}
		for _, arg := range args {
			parts = append(parts, quote(arg))
		}
		script := "$OutputEncoding=[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); " + strings.Join(parts, " ") + "; exit $LASTEXITCODE"
		units := utf16.Encode([]rune(script))
		data := make([]byte, len(units)*2)
		for i, u := range units {
			binary.LittleEndian.PutUint16(data[i*2:], u)
		}
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	} else {
		cmd = exec.CommandContext(ctx, resolved, args...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd
}
