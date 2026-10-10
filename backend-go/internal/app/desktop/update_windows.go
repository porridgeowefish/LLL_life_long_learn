//go:build windows

package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// launchInstaller starts a hidden helper that waits for this process to exit,
// installs the verified MSI silently (the in-app click is the consent), records
// the result for the next start and reopens Learn-With-AI from wherever the MSI
// registered it.
func launchInstaller(msi, version, statusPath string) error {
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	log := filepath.Join(filepath.Dir(msi), "install-"+version+".log")
	script := fmt.Sprintf(`$ErrorActionPreference = 'Continue'
try { Wait-Process -Id %d -Timeout 60 -ErrorAction SilentlyContinue } catch {}
$p = Start-Process -FilePath msiexec.exe -ArgumentList @('/i', '"' + %s + '"', '/qn', '/norestart', 'MIGRATELEGACY=1', '/l*v', '"' + %s + '"') -Wait -PassThru
$result = @{ version = %s; exitCode = $p.ExitCode; log = %s } | ConvertTo-Json -Compress
[IO.File]::WriteAllText(%s, $result)
$location = (Get-ItemProperty 'HKCU:\Software\learn-with-ai\Installer' -ErrorAction SilentlyContinue).InstallLocation
if ($location) { Start-Process -FilePath (Join-Path $location 'learn-with-ai.exe') }
`, os.Getpid(), quote(msi), quote(log), quote(version), quote(log), quote(statusPath))
	helper := filepath.Join(filepath.Dir(msi), "install-update.ps1")
	// UTF-8 with BOM so Windows PowerShell 5.1 reads non-ASCII paths correctly.
	if err := os.WriteFile(helper, append([]byte{0xEF, 0xBB, 0xBF}, []byte(script)...), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", helper)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008 | 0x00000200} // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	return cmd.Start()
}
