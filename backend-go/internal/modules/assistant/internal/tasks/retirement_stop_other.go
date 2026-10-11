//go:build !windows

package assistanttask

import (
	"errors"
	"time"
)

// The retained Unix runtime only recorded a shell timestamp, not a verifiable
// kernel process birth identity. Keep data and block instead of killing a PID.
func stopVerifiedExecutor(pid int, started time.Time) error {
	return errors.New("legacy executor identity cannot be safely stopped; close the old terminal")
}
