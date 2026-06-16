//go:build !windows

package claudelauncher

import "os/exec"

func configureHeadlessCommand(_ *exec.Cmd) {}
