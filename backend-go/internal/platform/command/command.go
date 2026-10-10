// Package command resolves native executables and Windows npm shims.
package command

import (
	"context"
	"os/exec"
)

func Context(ctx context.Context, bin string, args ...string) *exec.Cmd {
	return nativeCommand(ctx, bin, args)
}
