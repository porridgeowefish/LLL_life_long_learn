//go:build !windows

package filesystem

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockInstance(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}
