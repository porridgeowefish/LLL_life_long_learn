//go:build !windows

package claudelauncher

import (
	"fmt"
	"os/exec"
)

// Non-Windows fallback for launchVisibleWindow. ShellExecute is Windows-only;
// here we spawn directly and let the host terminal own the window. A follow-up
// iteration should add xterm / Terminal.app equivalents (see iter-03 README
// "Deferred to iter-04", B-class infra — cross-platform console).
func launchVisibleWindow(exe string, args []string, workDir string) error {
	cmd := exec.Command(exe, args...)
	cmd.Dir = workDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("non-windows spawn: %w", err)
	}
	return nil
}
