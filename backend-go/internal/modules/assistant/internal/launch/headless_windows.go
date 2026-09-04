//go:build windows

package claudelauncher

import (
	"os/exec"
	"syscall"
)

func configureHeadlessCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
