//go:build !windows

package claudelauncher

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

func launchVisibleWindow(exe string, args []string, workDir string, onExit func(int)) error {
	if exe != "/bin/sh" || len(args) != 1 {
		return fmt.Errorf("native terminal expects a saved shell wrapper")
	}
	wrapper := args[0]
	available := func(name string) bool { _, err := exec.LookPath(name); return err == nil }
	bin, terminalArgs, err := terminalInvocation(runtime.GOOS, available, wrapper, workDir)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, terminalArgs...)
	cmd.Dir = workDir
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	if onExit != nil {
		go monitorTerminal(wrapper, onExit)
	}
	return nil
}
func monitorTerminal(wrapper string, onExit func(int)) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(30 * time.Second)
	started := false
	for range ticker.C {
		var finished struct {
			ExitCode int `json:"exitCode"`
		}
		if data, err := os.ReadFile(wrapper + ".exit.json"); err == nil && json.Unmarshal(data, &finished) == nil {
			onExit(finished.ExitCode)
			return
		}
		var executor struct {
			PID int `json:"pid"`
		}
		if data, err := os.ReadFile(wrapper + ".executor.json"); err == nil && json.Unmarshal(data, &executor) == nil && executor.PID > 0 {
			started = true
			if err := syscall.Kill(executor.PID, 0); err == syscall.ESRCH {
				onExit(1)
				return
			}
		}
		if !started && time.Now().After(deadline) {
			onExit(1)
			return
		}
	}
}
