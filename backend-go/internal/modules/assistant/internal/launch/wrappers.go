package claudelauncher

import (
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"path/filepath"

	"sort"
	"strings"

	"encoding/json"
	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"

	"fmt"
	"os"

	"github.com/xmz14/lll/backend-go/internal/sessionstore"
)

func buildTUIWrapperShellScript(projectRoot, promptMdPath, stderrPath, settingsFile string, req LaunchRequest) string {
	rt := selectedRuntime(req)
	execLine := interactiveShellExecLine(rt, settingsFile)
	lines := []string{
		`#!/bin/sh`,
		`set +e`,
		fmt.Sprintf(`cd %s || exit 1`, shQuote(projectRoot)),
	}
	lines = append(lines, taskShellPrelude(req)...)
	lines = append(lines,
		fmt.Sprintf(`log_file=%s`, shQuote(stderrPath)),
		`log_err() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$1" >> "$log_file"; }`,
		`log_err "spawn env PATH=$PATH"`,
		`printf '%s\n' '============================================='`,
		fmt.Sprintf(`printf ' LLL Project: %s | Agent: %s | Runtime: %s | Zone: %s\n'`, req.ProjectSlug, req.Agent.Name, rt.Name, req.ZoneName),
		`printf '%s\n\n' '============================================='`,
		fmt.Sprintf(`prompt_file=%s`, shQuote(promptMdPath)),
		`prompt_text=''`,
		`if [ -f "$prompt_file" ]; then prompt_text=$(cat "$prompt_file"); else log_err "prompt.md missing: $prompt_file"; fi`,
		`command -v clip.exe >/dev/null 2>&1 && clip.exe < "$prompt_file" 2>/dev/null`,
		fmt.Sprintf(`agent_bin=%s`, shQuote(rt.Bin)),
		`agent_exe=$(command -v "$agent_bin" 2>/dev/null)`,
		`log_err "resolved runtime agentExe=$agent_exe"`,
		`if [ -z "$agent_exe" ]; then printf 'Cannot find %s. Install it inside this Linux/WSL environment, then rerun.\n' "$agent_bin"; read -r _; exit 1; fi`,
		fmt.Sprintf(`permission_mode=%s`, shQuote(NormalizePermissionMode(req.PermissionMode))),
		`export FORCE_COLOR=1`,
		`printf 'Starting %s. The initial prompt is loaded from prompt.md.\n\n' "$agent_bin"`,
		execLine,
		`code=$?`,
		taskShellFinished(req),
		runStatusShellLine(sessionID(req.Session), req.runToken),
		`printf '\n=============================================\n'`,
		`printf 'Agent CLI exited with code %s. Press Enter to close.\n' "$code"`,
		`printf '=============================================\n'`,
		`read -r _`,
		`exit "$code"`,
	)
	return strings.Join(lines, "\n")
}

func buildResumeWrapperShellScript(projectRoot, stderrPath string, req ResumeRequest) string {
	rt := selectedResumeRuntime(req)
	execLine := resumeShellExecLine(rt)
	command := resumeCommandDescription(rt)
	return strings.Join([]string{
		`#!/bin/sh`,
		`set +e`,
		fmt.Sprintf(`cd %s || exit 1`, shQuote(projectRoot)),
		fmt.Sprintf(`log_file=%s`, shQuote(stderrPath)),
		`log_err() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$1" >> "$log_file"; }`,
		`log_err "resume spawn env PATH=$PATH"`,
		fmt.Sprintf(`agent_bin=%s`, shQuote(rt.Bin)),
		`agent_exe=$(command -v "$agent_bin" 2>/dev/null)`,
		fmt.Sprintf(`log_err "resolved runtime=%s agentExe=$agent_exe"`, rt.ID),
		`if [ -z "$agent_exe" ]; then printf 'Cannot find %s in this Linux environment.\n' "$agent_bin"; read -r _; exit 1; fi`,
		fmt.Sprintf(`printf 'Resuming with %s...\n\n'`, command),
		execLine,
		`code=$?`,
		runStatusShellLine(sessionID(req.Session), req.runToken),
		`printf '\nAgent CLI exited with code %s. Press Enter to close.\n' "$code"`,
		`read -r _`,
		`exit "$code"`,
	}, "\n")
}

// buildTUIWrapperScript returns the PowerShell command that:
//  1. cd into the project root; logs the spawn env PATH (ground truth)
//  2. reads prompt.md and resolves claude to a FULL path (Get-Command
//     + ~/.local/bin + AppData fallbacks) — a bare 'claude' can silently
//     fail to resolve in the ShellExecute child env; if unresolved it
//     aborts with a clear on-screen reason instead of a silent stall
//  3. prints a banner with project / agent / zone info
//  4. launches claude WITH the prompt as the initial message (positional
//     arg) — still interactive TUI mode (NOT -p), per LESSONS_LEARNED §1,
//     so the learner does not have to paste the prompt manually
//  5. on exit, prompts "press any key to close"
//
// The claude launch runs under ErrorActionPreference='Stop' so a missing
// binary / launch failure is a CAUGHT, logged terminating error — silent
// failures are not acceptable for diagnosis. The clipboard is kept as a
// silent fallback in case the arg is ever mangled for an edge-case prompt.
func buildTUIWrapperScript(projectRoot, promptMdPath, stderrPath, settingsFile string, req LaunchRequest) string {
	permissionMode := NormalizePermissionMode(req.PermissionMode)
	runtime := selectedRuntime(req)
	execLine := interactiveExecLine(runtime, projectRoot, promptMdPath, settingsFile)
	promptNotice := "初始 prompt 会自动带入（无需手动粘贴）。"
	if runtime.PromptDelivery == agentruntime.PromptClipboard {
		promptNotice = "已把初始 prompt 放入剪贴板；进入 CLI 后按 Ctrl+V 再回车。"
	}
	lines := []string{
		`$ErrorActionPreference='Continue'`,
		fmt.Sprintf(`Set-Location -LiteralPath '%s'`, projectRoot),
	}
	lines = append(lines, taskPowerShellPrelude(req)...)
	lines = append(lines,
		`$env:FORCE_COLOR='1'`,

		fmt.Sprintf(`function Log-Err($msg) { try { Add-Content -LiteralPath '%s' -Value "[$(Get-Date -Format 'HH:mm:ss')] $msg" } catch {} }`, stderrPath),

		`Log-Err "spawn env PATH=$env:PATH"`,
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		fmt.Sprintf(`Write-Host ' LLL 项目: %s  |  Agent: %s %s  |  Runtime: %s  |  Zone: %s ' -ForegroundColor Cyan`,
			req.ProjectSlug, req.Agent.Icon, req.Agent.Name, runtime.Name, req.ZoneName),
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		`Write-Host ''`,

		fmt.Sprintf(`$promptText = ''; try { $promptText = Get-Content -Raw -Encoding UTF8 -LiteralPath '%s' } catch { Log-Err "read prompt.md failed: $_"; Write-Host "⚠️  无法读取 prompt.md：$_" -ForegroundColor Red }`, promptMdPath),

		`$promptText = $promptText -replace [char]34, [char]0x201C`,

		fmt.Sprintf(`$agentExe = $null; $cmd = Get-Command '%s' -ErrorAction SilentlyContinue; if ($cmd) { $agentExe = $cmd.Source }; Log-Err "resolved runtime=%s agentExe=$agentExe"`, runtime.Bin, runtime.ID),
		fmt.Sprintf(`$permissionMode = '%s'; Log-Err "permissionMode=$permissionMode"`, permissionMode),

		`try { if ($promptText) { Set-Clipboard -Value $promptText } } catch { Log-Err "Set-Clipboard failed: $_" }`,
		fmt.Sprintf(`Write-Host '即将启动 %s，%s' -ForegroundColor Green`, runtime.Name, promptNotice),
		`Write-Host '模型、账号、密钥等配置由所选 CLI 自己管理，LLL 只负责选择运行时和组织项目上下文。' -ForegroundColor DarkGray`,
		`Write-Host ''`,

		fmt.Sprintf(`if (-not $agentExe) { Log-Err "%s NOT FOUND"; Write-Host "⚠️  找不到 %s。请确认已安装，或设置 %s 指向可执行文件全路径。" -ForegroundColor Red; Write-Host "    （诊断：本窗口 PATH 与解析结果已写入 stdout.log）" -ForegroundColor DarkGray; $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown'); exit 1 }`, runtime.DefaultBin, runtime.DefaultBin, runtime.BinEnv),
		fmt.Sprintf(`Write-Host '正在启动 %s...' -ForegroundColor Cyan`, runtime.Name),
		`Write-Host ''`,

		`$ErrorActionPreference='Stop'; $code = 0; try { `+execLine+`; if ($LASTEXITCODE) { $code = $LASTEXITCODE } } catch { Log-Err "runtime exec failed: $_"; Write-Host "⚠️  Agent CLI 启动失败：$_" -ForegroundColor Red; $code = 1 }; $ErrorActionPreference='Continue'`,
		taskPowerShellFinished(req),
		runStatusPowerShellLine(sessionID(req.Session), req.runToken),
		`Write-Host ''`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`Write-Host " Agent CLI 已退出 (退出码 $code). 按任意键关闭窗口" -ForegroundColor Cyan`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')`,
		`exit $code`,
	)
	return strings.Join(lines, "; ")
}

func taskPowerShellPrelude(req LaunchRequest) []string {
	var lines []string
	keys := make([]string, 0, len(req.taskEnv))
	for key := range req.taskEnv {
		if strings.HasPrefix(key, "LLL_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf(`$env:%s = '%s'`, key, psSingleQuote(req.taskEnv[key])))
	}
	if req.taskExecutorPath != "" {
		lines = append(lines, fmt.Sprintf(`$started = @{schemaVersion=1; pid=$PID; startedAt=[DateTime]::UtcNow.ToString('o')} | ConvertTo-Json -Compress; [System.IO.File]::WriteAllText('%s', $started, [System.Text.UTF8Encoding]::new($false))`, psSingleQuote(req.taskExecutorPath)))
	}
	return lines
}

func taskPowerShellFinished(req LaunchRequest) string {
	if req.taskExitPath == "" {
		return ""
	}
	return fmt.Sprintf(`$finished = @{schemaVersion=1; pid=$PID; exitCode=$code; finishedAt=[DateTime]::UtcNow.ToString('o')} | ConvertTo-Json -Compress; [System.IO.File]::WriteAllText('%s', $finished, [System.Text.UTF8Encoding]::new($false))`, psSingleQuote(req.taskExitPath))
}

func taskShellPrelude(req LaunchRequest) []string {
	var lines []string
	keys := make([]string, 0, len(req.taskEnv))
	for key := range req.taskEnv {
		if strings.HasPrefix(key, "LLL_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, "export "+key+"="+shQuote(req.taskEnv[key]))
	}
	if req.taskExecutorPath != "" {
		lines = append(lines, `printf '{"schemaVersion":1,"pid":%s,"startedAt":"%s"}\n' "$$" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > `+shQuote(req.taskExecutorPath))
	}
	return lines
}

func taskShellFinished(req LaunchRequest) string {
	if req.taskExitPath == "" {
		return ":"
	}
	return `printf '{"schemaVersion":1,"pid":%s,"exitCode":%s,"finishedAt":"%s"}\n' "$$" "$code" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > ` + shQuote(req.taskExitPath)
}

func buildResumeWrapperScript(projectRoot, stderrPath string, req ResumeRequest) string {
	rt := selectedResumeRuntime(req)
	command := resumeCommandDescription(rt)
	execLine := resumePowerShellExecLine(rt, projectRoot)
	resolveLines := []string{}
	if rt.Mode != "wsl" {
		resolveLines = append(resolveLines,
			fmt.Sprintf(`$agentExe = $null; $cmd = Get-Command '%s' -ErrorAction SilentlyContinue; if ($cmd) { $agentExe = $cmd.Source }; Log-Err "resolved runtime=%s agentExe=$agentExe"`, rt.Bin, rt.ID),
			fmt.Sprintf(`if (-not $agentExe) { Log-Err "%s NOT FOUND"; Write-Host "⚠️  找不到 %s。请确认已安装，或设置 %s 指向可执行文件全路径。" -ForegroundColor Red; $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown'); exit 1 }`, rt.DefaultBin, rt.DefaultBin, rt.BinEnv),
		)
	}
	lines := []string{
		`$ErrorActionPreference='Continue'`,
		fmt.Sprintf(`Set-Location -LiteralPath '%s'`, projectRoot),
		`$env:FORCE_COLOR='1'`,
		fmt.Sprintf(`function Log-Err($msg) { try { Add-Content -LiteralPath '%s' -Value "[$(Get-Date -Format 'HH:mm:ss')] $msg" } catch {} }`, stderrPath),
		`Log-Err "resume spawn env PATH=$env:PATH"`,
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		fmt.Sprintf(`Write-Host ' LLL 项目: %s  |  Explain 继续上次会话  |  Runtime: %s ' -ForegroundColor Cyan`, req.ProjectSlug, rt.Name),
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		`Write-Host ''`,
	}
	lines = append(lines, resolveLines...)
	lines = append(lines,
		fmt.Sprintf(`Write-Host '正在执行 %s，恢复最近一次会话...' -ForegroundColor Green`, command),
		`Write-Host ''`,
		`$ErrorActionPreference='Stop'; $code = 0; try { `+execLine+`; if ($LASTEXITCODE) { $code = $LASTEXITCODE } } catch { Log-Err "resume failed: $_"; Write-Host "⚠️  Agent 继续会话失败：$_" -ForegroundColor Red; $code = 1 }; $ErrorActionPreference='Continue'`,
		runStatusPowerShellLine(sessionID(req.Session), req.runToken),
		`Write-Host ''`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`Write-Host " Agent CLI 已退出 (退出码 $code). 按任意键关闭窗口" -ForegroundColor Cyan`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')`,
		`exit $code`,
	)
	return strings.Join(lines, "; ")
}

func sessionID(sess *sessionstore.Session) string {
	if sess == nil {
		return ""
	}
	return sess.ID
}

func runStatusPowerShellLine(runID, token string) string {
	if runID == "" || token == "" {
		return ""
	}
	url := "http://127.0.0.1:8787/api/runs/" + runID + "/status"
	return fmt.Sprintf(`$statusBody = if ($code -eq 0) { '{"done":true}' } else { '{"failed":true}' }; try { Invoke-RestMethod -Method Post -Uri '%s' -Headers @{'X-Run-Token'='%s'} -ContentType 'application/json' -Body $statusBody | Out-Null } catch { Log-Err "report runtime exit failed: $_" }`, url, token)
}

func runStatusShellLine(runID, token string) string {
	if runID == "" || token == "" {
		return ":"
	}
	url := "http://127.0.0.1:8787/api/runs/" + runID + "/status"
	return fmt.Sprintf(`if [ "$code" -eq 0 ]; then status_body='{"done":true}'; else status_body='{"failed":true}'; fi; curl -s -o /dev/null -X POST %s -H %s -H 'Content-Type: application/json' -d "$status_body" || log_err "report runtime exit failed"`, shQuote(url), shQuote("X-Run-Token: "+token))
}

func selectedClaudeBin(bin string) string {
	if strings.TrimSpace(bin) == "" {
		return "claude"
	}
	return strings.TrimSpace(bin)
}

func selectedResumeRuntime(req ResumeRequest) agentruntime.Runtime {
	if req.Runtime != nil {
		return *req.Runtime
	}
	def, _ := agentruntime.DefinitionByID(agentruntime.RuntimeClaude)
	return agentruntime.Runtime{Definition: def, Bin: selectedClaudeBin(req.ClaudeBin), Available: true, Mode: "native"}
}

func resumeCommandDescription(rt agentruntime.Runtime) string {
	if rt.ID == agentruntime.RuntimeCodex {
		return "codex resume --last --yolo"
	}
	return "claude -c"
}

func resumePowerShellExecLine(rt agentruntime.Runtime, projectRoot string) string {
	if rt.Mode == "wsl" {
		project := psSingleQuote(projectRoot)
		if rt.ID == agentruntime.RuntimeCodex {
			return fmt.Sprintf(`$wslProject = (& wsl.exe wslpath -a '%s').Trim(); & wsl.exe -e sh -lc "cd ""$wslProject"" && exec %s resume --last --yolo"`, project, shQuote(rt.Bin))
		}
		return fmt.Sprintf(`$wslProject = (& wsl.exe wslpath -a '%s').Trim(); & wsl.exe -e sh -lc "cd ""$wslProject"" && exec %s -c"`, project, shQuote(rt.Bin))
	}
	if rt.ID == agentruntime.RuntimeCodex {
		return `& $agentExe resume --last --yolo`
	}
	return `& $agentExe -c`
}

func resumeShellExecLine(rt agentruntime.Runtime) string {
	if rt.ID == agentruntime.RuntimeCodex {
		return `"$agent_exe" resume --last --yolo`
	}
	return `"$agent_exe" -c`
}

func selectedRuntime(req LaunchRequest) agentruntime.Runtime {
	if req.Runtime != nil {
		return *req.Runtime
	}
	def, _ := agentruntime.DefinitionByID(agentruntime.RuntimeClaude)
	bin := req.ClaudeBin
	if bin == "" {
		bin = def.DefaultBin
	}
	return agentruntime.Runtime{Definition: def, Bin: bin, Available: true}
}

func interactiveExecLine(rt agentruntime.Runtime, projectRoot, promptMdPath, settingsFile string) string {
	if rt.Mode == "wsl" {
		return interactiveWSLExecLine(rt, projectRoot, promptMdPath, settingsFile)
	}

	settingsArg := ""
	if strings.TrimSpace(settingsFile) != "" && rt.ID == agentruntime.RuntimeClaude {
		settingsArg = fmt.Sprintf(" --settings '%s'", settingsFile)
	}
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		return "if ($promptText) { & $agentExe --permission-mode $permissionMode" + settingsArg + " $promptText } else { & $agentExe --permission-mode $permissionMode" + settingsArg + " }"
	case agentruntime.RuntimeCodex:
		return "if ($promptText) { & $agentExe --yolo $promptText } else { & $agentExe --yolo }"
	case agentruntime.RuntimeTrae:
		return fmt.Sprintf("if ($promptText) { & $agentExe run $promptText --working-dir '%s' } else { & $agentExe interactive }", projectRoot)
	default:
		return "& $agentExe"
	}
}

func interactiveShellExecLine(rt agentruntime.Runtime, settingsFile string) string {
	settingsArg := ""
	if strings.TrimSpace(settingsFile) != "" && rt.ID == agentruntime.RuntimeClaude {
		settingsArg = " --settings " + shQuote(settingsFile)
	}
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		return `if [ -n "$prompt_text" ]; then "$agent_exe" --permission-mode "$permission_mode"` + settingsArg + ` "$prompt_text"; else "$agent_exe" --permission-mode "$permission_mode"` + settingsArg + `; fi`
	case agentruntime.RuntimeCodex:
		return `if [ -n "$prompt_text" ]; then "$agent_exe" --yolo "$prompt_text"; else "$agent_exe" --yolo; fi`
	case agentruntime.RuntimeTrae:
		return `if [ -n "$prompt_text" ]; then "$agent_exe" run "$prompt_text" --working-dir "$PWD"; else "$agent_exe" interactive; fi`
	default:
		return `"$agent_exe"`
	}
}

func interactiveWSLExecLine(rt agentruntime.Runtime, projectRoot, promptMdPath, settingsFile string) string {
	bin := shQuote(rt.Bin)
	project := psSingleQuote(projectRoot)
	prompt := psSingleQuote(promptMdPath)
	settingsPrefix := ""
	settingsArg := ""
	if strings.TrimSpace(settingsFile) != "" && rt.ID == agentruntime.RuntimeClaude {
		settingsPrefix = fmt.Sprintf(`$wslSettings = (& wsl.exe wslpath -a '%s').Trim(); `, psSingleQuote(settingsFile))
		settingsArg = ` --settings ""$wslSettings""`
	}
	prefix := fmt.Sprintf(`$wslProject = (& wsl.exe wslpath -a '%s').Trim(); $wslPrompt = (& wsl.exe wslpath -a '%s').Trim(); %s`, project, prompt, settingsPrefix)
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		return fmt.Sprintf(`%s& wsl.exe -e sh -lc "cd ""$wslProject"" && prompt_text=\$(cat ""$wslPrompt"" 2>/dev/null) && if [ -n ""\$prompt_text"" ]; then exec %s --permission-mode auto%s ""\$prompt_text""; else exec %s --permission-mode auto%s; fi"`, prefix, bin, settingsArg, bin, settingsArg)
	case agentruntime.RuntimeCodex:
		return fmt.Sprintf(`%s& wsl.exe -e sh -lc "cd ""$wslProject"" && prompt_text=\$(cat ""$wslPrompt"" 2>/dev/null) && if [ -n ""\$prompt_text"" ]; then exec %s --yolo ""\$prompt_text""; else exec %s --yolo; fi"`, prefix, bin, bin)
	default:
		return `& $agentExe`
	}
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// writeHookSettings writes a run-scoped Claude Code settings JSON that reports
// progress to LLL via the run-status endpoint, and returns its path + the
// generated auth token. The launcher passes the file to `claude --settings`.
//
// Hook command + schema are the best Windows-curl form for the current Claude
// Code CLI (confirmed: `claude --help` lists `--settings <file-or-json>`).
// Live end-to-end validation of the PostToolUse/Stop hooks firing against a
// real Explain run is a manual step — see DELIVERY_NOTES.md (Phase C). Per
// the project's Windows + UTF-8 rule (CLAUDE.md), only ASCII goes on the curl
// command line; Chinese activity text would be mangled via argv.
func writeHookSettings(runDir, runId string, store *runprogress.Store) (path, token string, err error) {
	token = store.Register(runId)
	base := "http://127.0.0.1:8787/api/runs/" + runId + "/status"
	post := `curl.exe -s -o /dev/null -X POST ` + base +
		` -H "X-Run-Token: ` + token + `" -H "Content-Type: application/json" -d ` +
		`"{\\"activity\\":\\"page write\\"}"`
	done := `curl.exe -s -o /dev/null -X POST ` + base +
		` -H "X-Run-Token: ` + token + `" -H "Content-Type: application/json" -d ` +
		`"{\\"done\\":true}"`
	settings := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []map[string]any{{
				"matcher": "Write|Edit",
				"hooks":   []map[string]any{{"type": "command", "command": post}},
			}},
			"Stop": []map[string]any{{
				"hooks": []map[string]any{{"type": "command", "command": done}},
			}},
		},
	}
	path = filepath.Join(runDir, "claude-settings.json")
	var data []byte
	data, err = json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return path, token, err
	}
	err = os.WriteFile(path, data, 0o644)
	return path, token, err
}
