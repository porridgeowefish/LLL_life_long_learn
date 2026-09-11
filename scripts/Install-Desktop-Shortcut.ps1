param(
  [string]$ShortcutName = 'LifeLongLearn',
  [string]$WorkspaceRoot = ''
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
$arguments = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$startScript`""
if (-not [string]::IsNullOrWhiteSpace($WorkspaceRoot)) {
  $resolvedWorkspace = (Resolve-Path -LiteralPath $WorkspaceRoot).Path
  $arguments += " -WorkspaceRoot `"$resolvedWorkspace`""
}
$shortcut.Arguments = $arguments
$shortcut.WorkingDirectory = $repoRoot
$shortcut.IconLocation = $iconPath
$shortcut.Description = 'Start LifeLongLearn'
$shortcut.Save()

Write-Host "Created desktop shortcut: $shortcutPath"
