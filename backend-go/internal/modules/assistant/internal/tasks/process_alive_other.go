//go:build !windows

package assistanttask

import (
	"os"
	"syscall"
	"time"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	return err == nil && process.Signal(syscall.Signal(0)) == nil
}

func processMatches(pid int, _ time.Time) bool { return processAlive(pid) }
