//go:build windows

package claudelauncher

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteExW     = shell32.NewProc("ShellExecuteExW")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess  = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
)

// SW_SHOWNORMAL — activate and show the window at its default size/position.
const swShowNormal = 1
const seeMaskNoCloseProcess = 0x00000040
const infiniteWait = 0xffffffff

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     uintptr
}

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
// ShellExecuteEx with the "open" verb routes through the Windows Shell — the
// same path as double-clicking in Explorer — which launches the process on
// the interactive desktop regardless of the caller's console/window-station.
// SEE_MASK_NOCLOSEPROCESS also returns a process handle, which lets the backend
// reconcile the session when the terminal exits or is closed directly.
func launchVisibleWindow(exe string, args []string, workDir string, onExit func(exitCode int)) error {
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

	info := shellExecuteInfo{
		cbSize:       uint32(unsafe.Sizeof(shellExecuteInfo{})),
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: p,
		lpDirectory:  dir,
		nShow:        swShowNormal,
	}
	ret, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return fmt.Errorf("ShellExecuteEx failed launching %s %s: %v", exe, params, callErr)
	}
	if info.hProcess != 0 {
		go func(handle uintptr) {
			procWaitForSingleObject.Call(handle, infiniteWait)
			var exitCode uint32
			ok, _, _ := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
			procCloseHandle.Call(handle)
			if onExit != nil {
				if ok == 0 {
					onExit(1)
					return
				}
				onExit(int(exitCode))
			}
		}(info.hProcess)
	}
	return nil
}
