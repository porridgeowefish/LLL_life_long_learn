package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func showMessage(title, text string, confirm bool) bool {
	caption, _ := syscall.UTF16PtrFromString(title)
	message, _ := syscall.UTF16PtrFromString(text)
	flags := uintptr(0x40)
	if confirm {
		flags = 0x24
	}
	result, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(caption)), flags)
	return !confirm || result == 6
}

func psLiteral(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func powershell(script string, detach bool) error {
	units := utf16.Encode([]rune("$ErrorActionPreference = 'Stop'; " + script))
	data := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}
	command := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if detach {
		return command.Start()
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Windows shortcut integration: %w: %s", err, output)
	}
	return nil
}

func registryPath(identity string) string {
	return `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + identity
}

func createIntegration(root, identity string) error {
	script := `$shell=New-Object -ComObject WScript.Shell; ` +
		`$desktop=[Environment]::GetFolderPath('DesktopDirectory'); $programs=[Environment]::GetFolderPath('Programs'); ` +
		`$menu=Join-Path $programs ` + psLiteral(identity) + `; [IO.Directory]::CreateDirectory($menu) | Out-Null; ` +
		`foreach($path in @((Join-Path $desktop ` + psLiteral(identity+".lnk") + `),(Join-Path $menu 'learn-with-ai.lnk'))) { $link=$shell.CreateShortcut($path); $link.TargetPath=` + psLiteral(filepath.Join(root, "learn-with-ai.exe")) + `; $link.WorkingDirectory=` + psLiteral(root) + `; $link.WindowStyle=7; $link.Description='learn-with-ai'; $link.Save() }; ` +
		`$link=$shell.CreateShortcut((Join-Path $menu 'Uninstall.lnk')); $link.TargetPath=` + psLiteral(filepath.Join(root, uninstallerName)) + `; $link.Arguments='--uninstall'; $link.WorkingDirectory=` + psLiteral(root) + `; $link.Save()`
	if err := powershell(script, false); err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryPath(identity), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	for name, value := range map[string]string{"DisplayName": identity, "DisplayVersion": Version, "Publisher": "learn-with-ai contributors", "InstallLocation": root, "DisplayIcon": filepath.Join(root, "learn-with-ai.exe"), "UninstallString": `"` + filepath.Join(root, uninstallerName) + `" --uninstall`, "QuietUninstallString": `"` + filepath.Join(root, uninstallerName) + `" --uninstall --silent`} {
		if err := key.SetStringValue(name, value); err != nil {
			return err
		}
	}
	if err := key.SetDWordValue("NoModify", 1); err != nil {
		return err
	}
	return key.SetDWordValue("NoRepair", 1)
}

func removeIntegration(identity string) error {
	script := `$desktop=[Environment]::GetFolderPath('DesktopDirectory'); $programs=[Environment]::GetFolderPath('Programs'); ` +
		`$menu=Join-Path $programs ` + psLiteral(identity) + `; ` +
		`foreach($path in @((Join-Path $desktop ` + psLiteral(identity+".lnk") + `),(Join-Path $menu 'learn-with-ai.lnk'),(Join-Path $menu 'Uninstall.lnk'))) { if(Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Force } }; ` +
		`if((Test-Path -LiteralPath $menu) -and @(Get-ChildItem -LiteralPath $menu).Count -eq 0) { Remove-Item -LiteralPath $menu }`
	if err := powershell(script, false); err != nil {
		return err
	}
	err := registry.DeleteKey(registry.CURRENT_USER, registryPath(identity))
	if err == syscall.ERROR_FILE_NOT_FOUND {
		return nil
	}
	return err
}

func deleteAfterExit(self, root string) error {
	// Removal is limited to the executable and an empty installation folder.
	// The child waits for this exact PID; there is no recursive deletion.
	return powershell(fmt.Sprintf(`$parentProcess=Get-Process -Id %d -ErrorAction SilentlyContinue; if($parentProcess) { $parentProcess.WaitForExit() }; for($i=0; $i -lt 30; $i++) { try { Remove-Item -LiteralPath %s -Force; break } catch { Start-Sleep -Milliseconds 100 } }; if((Test-Path -LiteralPath %s) -and @(Get-ChildItem -LiteralPath %s).Count -eq 0) { Remove-Item -LiteralPath %s }`, os.Getpid(), psLiteral(self), psLiteral(root), psLiteral(root), psLiteral(root)), true)
}

func launchApplication(root string) error {
	command := exec.Command(filepath.Join(root, "learn-with-ai.exe"))
	command.Dir = root
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Start()
}
