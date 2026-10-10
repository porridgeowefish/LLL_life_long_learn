package browser

import (
	"context"
	command "github.com/xmz14/lll/backend-go/internal/platform/command"
	"runtime"
)

func Open(url string) error {
	bin, args := "xdg-open", []string{url}
	if runtime.GOOS == "darwin" {
		bin = "open"
	}
	if runtime.GOOS == "windows" {
		bin = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", url}
	}
	cmd := command.Context(context.Background(), bin, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
