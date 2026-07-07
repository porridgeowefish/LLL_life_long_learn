param(
  [string]$ShortcutName = 'LifeLongLearn'
)

$ErrorActionPreference = 'Stop'

function Get-RepoRoot {
  return (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
}

$repoRoot = Get-RepoRoot
$startScript = Join-Path $repoRoot 'scripts\Start-LLL.ps1'
$iconPath = Join-Path $repoRoot 'scripts\assets\lll.ico'
$desktop = [Environment]::GetFolderPath('Desktop')
$shortcutPath = Join-Path $desktop "$ShortcutName.lnk"

if (-not (Test-Path -LiteralPath $startScript)) {
  throw "Missing launcher script: $startScript"
}
if (-not (Test-Path -LiteralPath $iconPath)) {
  throw "Missing icon: $iconPath"
}

$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
$shortcut.Arguments = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$startScript`""
$shortcut.WorkingDirectory = $repoRoot
$shortcut.IconLocation = $iconPath
$shortcut.Description = 'Start LifeLongLearn'
$shortcut.Save()

Write-Host "Created desktop shortcut: $shortcutPath"
