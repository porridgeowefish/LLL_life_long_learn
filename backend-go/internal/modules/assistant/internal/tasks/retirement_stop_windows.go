//go:build windows

package assistanttask

import (
	"context"
	"errors"
	"github.com/xmz14/lll/backend-go/internal/platform/command"
	"strconv"
	"time"
)

func stopVerifiedExecutor(pid int, started time.Time) error {
	if started.IsZero() || !processMatches(pid, started) {
		return errors.New("cannot verify legacy executor")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e := command.Context(ctx, "taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run(); e != nil {
		return errors.New("cannot stop owned legacy executor")
	}
	for n := 0; n < 20; n++ {
		if !processMatches(pid, started) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("legacy executor remains active")
}
