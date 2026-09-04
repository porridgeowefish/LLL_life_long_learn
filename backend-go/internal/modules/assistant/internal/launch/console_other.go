//go:build !windows

package claudelauncher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Non-Windows fallback for launchVisibleWindow. ShellExecute is Windows-only;
// here we spawn directly and let the host terminal own the window. A follow-up
// iteration should add xterm / Terminal.app equivalents (see iter-03 README
// "Deferred to iter-04", B-class infra — cross-platform console).
func launchVisibleWindow(exe string, args []string, workDir string, onExit func(exitCode int)) error {
	if isWSL() {
		wtArgs := []string{"/C", "start", "", "wt.exe", "-w", "0", "wsl.exe", "--cd", workDir, "--exec", exe}
		wtArgs = append(wtArgs, args...)
		if err := exec.Command("cmd.exe", wtArgs...).Start(); err == nil {
			return nil
		}
		consoleArgs := []string{"/C", "start", "", "wsl.exe", "--cd", workDir, "--exec", exe}
		consoleArgs = append(consoleArgs, args...)
		if err := exec.Command("cmd.exe", consoleArgs...).Start(); err == nil {
			return nil
		}
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = workDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("non-windows spawn: %w", err)
	}
	go func() {
		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			exitCode = 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		if onExit != nil {
			onExit(exitCode)
		}
	}()
	return nil
}

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/version")
	return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
}
