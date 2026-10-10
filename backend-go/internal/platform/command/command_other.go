//go:build !windows

package command

import (
	"context"
	"os/exec"
)

func nativeCommand(ctx context.Context, bin string, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}
