//go:build windows

package assistanttask

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	processKernel32    = syscall.NewLazyDLL("kernel32.dll")
	processOpenProcess = processKernel32.NewProc("OpenProcess")
	processGetExitCode = processKernel32.NewProc("GetExitCodeProcess")
	processGetTimes    = processKernel32.NewProc("GetProcessTimes")
	processCloseHandle = processKernel32.NewProc("CloseHandle")
)

const processQueryLimitedInformation = 0x1000
const stillActive = 259

func processAlive(pid int) bool {
	return processMatches(pid, time.Time{})
}

func processMatches(pid int, recordedStart time.Time) bool {
	if pid <= 0 {
		return false
	}
	handle, _, _ := processOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(uint32(pid)))
	if handle == 0 {
		return false
	}
	defer processCloseHandle.Call(handle)
	var code uint32
	ok, _, _ := processGetExitCode.Call(handle, uintptr(unsafe.Pointer(&code)))
	if ok == 0 || code != stillActive {
		return false
	}
	if recordedStart.IsZero() {
		return true
	}
	var created, exited, kernel, user syscall.Filetime
	ok, _, _ = processGetTimes.Call(handle, uintptr(unsafe.Pointer(&created)), uintptr(unsafe.Pointer(&exited)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	if ok == 0 {
		return false
	}
	createdAt := time.Unix(0, created.Nanoseconds()).UTC()
	delta := recordedStart.UTC().Sub(createdAt)
	return delta >= -5*time.Second && delta <= 2*time.Minute
}
