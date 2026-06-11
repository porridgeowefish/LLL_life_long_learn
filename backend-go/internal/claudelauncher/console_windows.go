//go:build windows

package claudelauncher

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// SW_SHOWNORMAL — activate and show the window at its default size/position.
const swShowNormal = 1

// launchVisibleWindow spawns a process on the interactive desktop via
// ShellExecute("open", ...), independent of the caller's console allocation.
//
// Root cause this fixes (Stack Overflow #30182508 "Launching a new command
// window from Golang in Windows"; hashicorp/terraform-exec#570):
// os/exec (CreateProcess) child console window visibility depends on the
// PARENT process's console. A server process — e.g. lll.exe started via
// `go run` with redirected stdio — produces NO visible child window even
// with CREATE_NEW_CONSOLE (0x10) or `cmd /c start`. The window is created
// but never reaches the interactive desktop (verified: spawned
// powershell/claude processes run yet no window appears).
//
// ShellExecute with the "open" verb routes through the Windows Shell — the
// same path as double-clicking in Explorer — which launches the process on
// the interactive desktop regardless of the caller's console/window-station.
// Verified end-to-end: a green-text PowerShell window appears at the project
// path when spawned from a headless (no-console) Go process.
func launchVisibleWindow(exe string, args []string, workDir string) error {
	// Build lpParameters: each arg quoted Windows-style, space-separated.
	params := ""
	for _, a := range args {
		params += syscall.EscapeArg(a) + " "
	}
	if len(params) > 0 {
		params = params[:len(params)-1] // trim trailing space
	}

	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(exe)
	p, _ := syscall.UTF16PtrFromString(params)
	dir, _ := syscall.UTF16PtrFromString(workDir)

	// ShellExecuteW returns an HINSTANCE; values <= 32 are error codes.
	ret, _, _ := procShellExecuteW.Call(0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(dir)),
		swShowNormal,
	)
	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed (code %d) launching %s %s", ret, exe, params)
	}
	return nil
}
