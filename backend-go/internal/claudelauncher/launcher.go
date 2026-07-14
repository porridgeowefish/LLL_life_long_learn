// Package claudelauncher spawns the real Claude Code CLI in TRUE
// interactive TUI mode by default. It now also supports selected compatible
// agent CLIs through backend-go/internal/agentruntime.
//
// Trade-off accepted: with TUI mode we lose programmatic stdout capture.
// LLL no longer parses Claude's output stream. Instead, the frontend
// OutputViewer polls the zone's output.md (via TanStack Query staleTime)
// to render the curated artifact once Claude writes it. This matches
// BACKEND_ARCHITECTURE.md: "LLL launches and organizes Claude Code. The
// terminal is the execution surface."
package claudelauncher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/agentruntime"
	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// EventEmitter receives a typed SSE-able event.
type EventEmitter interface {
	Emit(event string, payload any)
}

// LaunchRequest carries everything needed to start a Claude run.
type LaunchRequest struct {
	ProjectSlug    string
	ZoneName       workspace.ZoneName
	Agent          *agentregistry.Agent
	PromptPackage  *promptassembly.Package
	PermissionMode string
	Session        *sessionstore.Session
	Store          *sessionstore.Store
	Events         EventEmitter
	ClaudeBin      string
	Runtime        *agentruntime.Runtime
	// RunProgress, when non-nil and the runtime is Claude Code, triggers
	// per-run hook injection: the launcher registers the run, writes a
	// run-scoped claude-settings.json whose PostToolUse/Stop hooks POST to
	// /api/runs/{runId}/status, and passes it via `claude --settings <file>`.
	// Nil = no hooks (non-Claude runtimes keep the Phase-A indeterminate bar).
	RunProgress *runprogress.Store
}

// ResumeRequest carries the minimal context needed to reopen Claude Code's
// most recent conversation for a project.
type ResumeRequest struct {
	ProjectSlug string
	ZoneName    workspace.ZoneName
	Session     *sessionstore.Session
	Store       *sessionstore.Store
	Events      EventEmitter
	ClaudeBin   string
	Runtime     *agentruntime.Runtime
	RunDirName  string
}

const defaultPermissionMode = "auto"

// NormalizePermissionMode keeps old clients working while making auto mode the
// default for every agent invocation.
func NormalizePermissionMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return defaultPermissionMode
	}
	return mode
}

// RunResult is what the launcher returns. With TUI mode, it returns
// immediately after spawning the wrapper — final result.md is not
// captured here (frontend polls output.md instead).
type RunResult struct {
	ExitCode        int
	StdoutLogPath   string
	StderrLogPath   string
	PromptMdPath    string
	ResultMdPath    string
	RunDirAbs       string
	RunDirRel       string
	PackageJSONPath string
}

// Launch spawns the selected agent runtime in an interactive PowerShell window.
// The function returns immediately after the wrapper is started — it does
// NOT wait for Claude to exit. The user closes the window when done.
func Launch(ctx context.Context, req LaunchRequest) (*RunResult, error) {
	if req.Agent == nil || req.PromptPackage == nil || req.Session == nil {
		return nil, errors.New("missing required launch input")
	}
	projectRoot, err := workspace.ProjectRootForSlug(req.ProjectSlug)
	if err != nil {
		return nil, err
	}
	runDirAbs := filepath.Join(projectRoot, "runs", req.PromptPackage.RunDirName)
	runDirRel := filepath.Join("runs", req.PromptPackage.RunDirName)
	if err := os.MkdirAll(runDirAbs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir run dir: %w", err)
	}

	// Write prompt.md + package.json + run.json metadata.
	promptMdPath := filepath.Join(runDirAbs, "prompt.md")
	if err := os.WriteFile(promptMdPath, []byte(req.PromptPackage.PromptMd), 0o644); err != nil {
		return nil, err
	}
	packageJSON, err := req.PromptPackage.MarshalPackageMeta()
	if err != nil {
		return nil, err
	}
	packageJSONPath := filepath.Join(runDirAbs, "package.json")
	if err := os.WriteFile(packageJSONPath, packageJSON, 0o644); err != nil {
		return nil, err
	}
	stdoutPath := filepath.Join(runDirAbs, "stdout.log")
	stderrPath := filepath.Join(runDirAbs, "stderr.log")
	_ = os.WriteFile(stdoutPath, []byte("(interactive TUI mode — stdout is not captured; the conversation lives in the PowerShell window)\n"), 0o644)
	_ = os.WriteFile(stderrPath, []byte{}, 0o644)

	// Append the user turn (the assembled prompt) and mark session running.
	req.Store.AppendTurn(req.Session.ID, "user", req.PromptPackage.PromptMd, runDirRel)
	req.Store.Update(req.Session.ID, func(s *sessionstore.Session) {
		s.State = sessionstore.StateRunning
		s.RunDirRel = runDirRel
		s.PromptPath = filepath.Join(runDirRel, "prompt.md")
	})
	if req.Events != nil {
		req.Events.Emit("session-state", map[string]any{
			"sessionId": req.Session.ID, "state": "running", "runDirRel": runDirRel,
		})
		req.Events.Emit("turn-created", map[string]any{
			"sessionId": req.Session.ID,
			"turnType":  "user",
			"ordinal":   len(req.Session.Turns),
		})
	}

	runtime := selectedRuntime(req)

	// Phase C: when a run-progress store is wired in and the runtime is the
	// Claude CLI, register the run + emit a run-scoped Claude Code settings
	// file whose hooks POST activity/completion to /api/runs/{runId}/status.
	// The settings file is passed to `claude --settings <file>` below. Live
	// validation of the hook command + schema is a manual step — see
	// docs/01-iterations/iteration-06-ask-ai-and-live-progress/DELIVERY_NOTES.md.
	var settingsFile string
	if req.RunProgress != nil && runtime.ID == agentruntime.RuntimeClaude {
		sf, _, herr := writeHookSettings(runDirAbs, req.Session.ID, req.RunProgress)
		if herr != nil {
			// Non-fatal: hooks are best-effort. The run still proceeds; the
			// frontend falls back to the Phase-A indeterminate bar.
			println("runprogress: write hook settings warning:", herr.Error())
		} else {
			settingsFile = sf
		}
	}

	// Build the PowerShell wrapper script. For runtimes that support an
	// initial prompt argument, LLL passes the prompt directly. For runtimes
	// whose interactive prompt contract is not stable, LLL preloads the
	// assembled prompt onto the clipboard and opens the real CLI surface.
	wrapperPath, err := writeAndLaunchInteractiveWrapper(runDirAbs, projectRoot, promptMdPath, stdoutPath, settingsFile, req)
	if err != nil {
		_ = os.WriteFile(stderrPath, []byte("launch window: "+err.Error()+"\nwrapper: "+wrapperPath+"\n"), 0o644)
		req.Store.SetFinished(req.Session.ID, sessionstore.StateFailed, -1)
		if req.Events != nil {
			req.Events.Emit("session-failed", map[string]any{
				"sessionId": req.Session.ID, "error": err.Error(),
			})
		}
		return nil, fmt.Errorf("spawn agent wrapper: %w", err)
	}

	// run.json metadata (written immediately; exitCode is left 0 since
	// we do not track it in TUI mode).
	runMeta := map[string]any{
		"sessionId":      req.Session.ID,
		"agentId":        req.Agent.ID,
		"projectSlug":    req.ProjectSlug,
		"zoneName":       req.ZoneName,
		"exitCode":       0,
		"promptMd":       filepath.Join(runDirRel, "prompt.md"),
		"stdoutLog":      filepath.Join(runDirRel, "stdout.log"),
		"stderrLog":      filepath.Join(runDirRel, "stderr.log"),
		"mode":           "interactive-tui",
		"runtimeId":      runtime.ID,
		"runtimeBin":     runtime.Bin,
		"permissionMode": NormalizePermissionMode(req.PermissionMode),
		"startedAt":      time.Now().UTC(),
	}
	metaJSON, _ := json.MarshalIndent(runMeta, "", "  ")
	_ = os.WriteFile(filepath.Join(runDirAbs, "run.json"), metaJSON, 0o644)

	// Touch result.md as an empty placeholder so the frontend can detect
	// it exists even before Claude writes the real artifact. The actual
	// content the learner sees comes from the zone's output.md, which
	// the frontend polls via TanStack Query.
	_ = os.WriteFile(filepath.Join(runDirAbs, "result.md"), []byte(""), 0o644)

	// Fire-and-forget: do NOT wait for the wrapper. Return immediately so
	// the HTTP handler can respond to the frontend while the learner is
	// still chatting.
	return &RunResult{
		ExitCode:        0,
		StdoutLogPath:   stdoutPath,
		StderrLogPath:   stderrPath,
		PromptMdPath:    promptMdPath,
		ResultMdPath:    filepath.Join(runDirAbs, "result.md"),
		PackageJSONPath: packageJSONPath,
		RunDirAbs:       runDirAbs,
		RunDirRel:       runDirRel,
	}, nil
}

// LaunchResume opens a visible terminal and resumes the selected runtime's
// latest conversation from the project root. It intentionally does not pass a
// freshly assembled prompt: the runtime owns the conversation history.
func LaunchResume(ctx context.Context, req ResumeRequest) (*RunResult, error) {
	if req.ProjectSlug == "" {
		return nil, errors.New("missing project slug")
	}
	rt := selectedResumeRuntime(req)
	if rt.ID != agentruntime.RuntimeClaude && rt.ID != agentruntime.RuntimeCodex {
		return nil, fmt.Errorf("runtime %s does not support interactive resume", rt.ID)
	}
	projectRoot, err := workspace.ProjectRootForSlug(req.ProjectSlug)
	if err != nil {
		return nil, err
	}
	runDirName := strings.TrimSpace(req.RunDirName)
	if runDirName == "" {
		runDirName = "resume"
	}
	runDirAbs := filepath.Join(projectRoot, "runs", runDirName)
	runDirRel := filepath.Join("runs", runDirName)
	if err := os.MkdirAll(runDirAbs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir run dir: %w", err)
	}
	stdoutPath := filepath.Join(runDirAbs, "stdout.log")
	stderrPath := filepath.Join(runDirAbs, "stderr.log")
	command := resumeCommandDescription(rt)
	_ = os.WriteFile(stdoutPath, []byte("resume mode — launching `"+command+"` in a visible terminal\n"), 0o644)
	_ = os.WriteFile(stderrPath, []byte{}, 0o644)

	if req.Session != nil && req.Store != nil {
		req.Store.AppendTurn(req.Session.ID, "system", "resume requested with "+command, runDirRel)
		req.Store.Update(req.Session.ID, func(s *sessionstore.Session) {
			s.State = sessionstore.StateRunning
			s.RunDirRel = runDirRel
		})
		if req.Events != nil {
			req.Events.Emit("session-state", map[string]any{
				"sessionId": req.Session.ID, "state": "running", "runDirRel": runDirRel,
			})
		}
	}

	wrapperPath, err := writeAndLaunchResumeWrapper(runDirAbs, projectRoot, stdoutPath, req)
	if err != nil {
		_ = os.WriteFile(stderrPath, []byte("launch resume window: "+err.Error()+"\nwrapper: "+wrapperPath+"\n"), 0o644)
		if req.Session != nil && req.Store != nil {
			req.Store.SetFinished(req.Session.ID, sessionstore.StateFailed, -1)
		}
		if req.Events != nil && req.Session != nil {
			req.Events.Emit("session-failed", map[string]any{
				"sessionId": req.Session.ID, "error": err.Error(),
			})
		}
		return nil, fmt.Errorf("spawn %s resume wrapper: %w", rt.ID, err)
	}

	runMeta := map[string]any{
		"projectSlug": req.ProjectSlug,
		"zoneName":    req.ZoneName,
		"mode":        "interactive-tui-resume",
		"runtimeId":   rt.ID,
		"runtimeBin":  rt.Bin,
		"command":     command,
		"stdoutLog":   filepath.Join(runDirRel, "stdout.log"),
		"stderrLog":   filepath.Join(runDirRel, "stderr.log"),
		"startedAt":   time.Now().UTC(),
	}
	if req.Session != nil {
		runMeta["sessionId"] = req.Session.ID
	}
	metaJSON, _ := json.MarshalIndent(runMeta, "", "  ")
	_ = os.WriteFile(filepath.Join(runDirAbs, "run.json"), metaJSON, 0o644)

	return &RunResult{
		ExitCode:      0,
		StdoutLogPath: stdoutPath,
		StderrLogPath: stderrPath,
		RunDirAbs:     runDirAbs,
		RunDirRel:     runDirRel,
	}, nil
}

func writeAndLaunchInteractiveWrapper(runDirAbs, projectRoot, promptMdPath, stderrPath, settingsFile string, req LaunchRequest) (string, error) {
	if runtime.GOOS == "windows" {
		psCmd := buildTUIWrapperScript(projectRoot, promptMdPath, stderrPath, settingsFile, req)
		wrapperPs1Path := filepath.Join(runDirAbs, "wrapper.ps1")
		if err := os.WriteFile(wrapperPs1Path, []byte("\ufeff"+psCmd), 0o644); err != nil {
			return wrapperPs1Path, fmt.Errorf("write wrapper.ps1: %w", err)
		}
		return wrapperPs1Path, launchVisibleWindow("powershell.exe",
			[]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-NoExit", "-File", wrapperPs1Path},
			projectRoot,
		)
	}

	shCmd := buildTUIWrapperShellScript(projectRoot, promptMdPath, stderrPath, settingsFile, req)
	wrapperShPath := filepath.Join(runDirAbs, "wrapper.sh")
	if err := os.WriteFile(wrapperShPath, []byte(shCmd), 0o755); err != nil {
		return wrapperShPath, fmt.Errorf("write wrapper.sh: %w", err)
	}
	return wrapperShPath, launchVisibleWindow("/bin/sh", []string{wrapperShPath}, projectRoot)
}

func writeAndLaunchResumeWrapper(runDirAbs, projectRoot, stderrPath string, req ResumeRequest) (string, error) {
	if runtime.GOOS == "windows" {
		psCmd := buildResumeWrapperScript(projectRoot, stderrPath, req)
		wrapperPs1Path := filepath.Join(runDirAbs, "wrapper.ps1")
		if err := os.WriteFile(wrapperPs1Path, []byte("\ufeff"+psCmd), 0o644); err != nil {
			return wrapperPs1Path, fmt.Errorf("write wrapper.ps1: %w", err)
		}
		return wrapperPs1Path, launchVisibleWindow("powershell.exe",
			[]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-NoExit", "-File", wrapperPs1Path},
			projectRoot,
		)
	}

	shCmd := buildResumeWrapperShellScript(projectRoot, stderrPath, req)
	wrapperShPath := filepath.Join(runDirAbs, "wrapper.sh")
	if err := os.WriteFile(wrapperShPath, []byte(shCmd), 0o755); err != nil {
		return wrapperShPath, fmt.Errorf("write wrapper.sh: %w", err)
	}
	return wrapperShPath, launchVisibleWindow("/bin/sh", []string{wrapperShPath}, projectRoot)
}

func buildTUIWrapperShellScript(projectRoot, promptMdPath, stderrPath, settingsFile string, req LaunchRequest) string {
	rt := selectedRuntime(req)
	execLine := interactiveShellExecLine(rt, settingsFile)
	return strings.Join([]string{
		`#!/bin/sh`,
		`set +e`,
		fmt.Sprintf(`cd %s || exit 1`, shQuote(projectRoot)),
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
		`printf '\n=============================================\n'`,
		`printf 'Agent CLI exited with code %s. Press Enter to close.\n' "$code"`,
		`printf '=============================================\n'`,
		`read -r _`,
		`exit "$code"`,
	}, "\n")
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
	return strings.Join([]string{
		`$ErrorActionPreference='Continue'`,
		fmt.Sprintf(`Set-Location -LiteralPath '%s'`, projectRoot),
		`$env:FORCE_COLOR='1'`,
		// Diagnostic helper: append a timestamped line to the log so a
		// failure is visible even if the window closes too fast. The caller
		// passes the stdout path as the third arg (see Launch); the param
		// keeps the stderrPath name for source stability.
		fmt.Sprintf(`function Log-Err($msg) { try { Add-Content -LiteralPath '%s' -Value "[$(Get-Date -Format 'HH:mm:ss')] $msg" } catch {} }`, stderrPath),
		// Ground truth: record the spawn env's PATH up front so a bare
		// 'claude' resolution failure can be diagnosed post-hoc.
		`Log-Err "spawn env PATH=$env:PATH"`,
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		fmt.Sprintf(`Write-Host ' LLL 项目: %s  |  Agent: %s %s  |  Runtime: %s  |  Zone: %s ' -ForegroundColor Cyan`,
			req.ProjectSlug, req.Agent.Icon, req.Agent.Name, runtime.Name, req.ZoneName),
		`Write-Host '=============================================' -ForegroundColor Cyan`,
		`Write-Host ''`,
		// Read prompt.md (the assembled prompt). Guarded so a missing file
		// does not abort the script.
		fmt.Sprintf(`$promptText = ''; try { $promptText = Get-Content -Raw -Encoding UTF8 -LiteralPath '%s' } catch { Log-Err "read prompt.md failed: $_"; Write-Host "⚠️  无法读取 prompt.md：$_" -ForegroundColor Red }`, promptMdPath),
		// Sanitize ASCII double-quotes out of the prompt BEFORE launching
		// claude.exe. PowerShell 5.1's legacy native-argument passing splits a
		// string containing an embedded `"` (U+0022) into multiple argv
		// elements — verified: a 4422-char prompt with 62 ASCII double-quotes
		// arrived at the native exe as 51 args, so claude took only argv[1]
		// (the first 642 chars) and silently dropped the rest (output contract,
		// behavior rules, expanded primitives) — the agent then "didn't write
		// the doc" because it never received the requirements. Replacing U+0022
		// with the Chinese curly quote U+201C removes the breaker, preserves the
		// meaning, and matches the charter's 全中文 mandate. Backticks (code
		// spans) are NOT a breaker — only U+0022 is. See LESSONS_LEARNED §13.
		`$promptText = $promptText -replace [char]34, [char]0x201C`,
		// Resolve the selected runtime to a FULL path. A bare binary can silently fail
		// to resolve in the ShellExecute child env (which may differ from
		// the backend's probe env), and under ErrorActionPreference=Continue
		// that failure is non-terminating — not caught, not logged — so the
		// window stalls at the banner with no Claude TUI. Get-Command honors
		// both bare names (PATH lookup) and a CLAUDE_BIN full path; the two
		// Test-Path fallbacks cover the standard Windows install locations.
		fmt.Sprintf(`$agentExe = $null; $cmd = Get-Command '%s' -ErrorAction SilentlyContinue; if ($cmd) { $agentExe = $cmd.Source }; Log-Err "resolved runtime=%s agentExe=$agentExe"`, runtime.Bin, runtime.ID),
		fmt.Sprintf(`$permissionMode = '%s'; Log-Err "permissionMode=$permissionMode"`, permissionMode),
		// Silent clipboard insurance: if the auto-injected arg is ever
		// mangled for an edge-case prompt, the learner can still paste.
		`try { if ($promptText) { Set-Clipboard -Value $promptText } } catch { Log-Err "Set-Clipboard failed: $_" }`,
		fmt.Sprintf(`Write-Host '即将启动 %s，%s' -ForegroundColor Green`, runtime.Name, promptNotice),
		`Write-Host '模型、账号、密钥等配置由所选 CLI 自己管理，LLL 只负责选择运行时和组织项目上下文。' -ForegroundColor DarkGray`,
		`Write-Host ''`,
		// If the runtime could not be resolved anywhere, stop with a clear reason
		// instead of the previous silent fall-through to "Claude 已退出".
		fmt.Sprintf(`if (-not $agentExe) { Log-Err "%s NOT FOUND"; Write-Host "⚠️  找不到 %s。请确认已安装，或设置 %s 指向可执行文件全路径。" -ForegroundColor Red; Write-Host "    （诊断：本窗口 PATH 与解析结果已写入 stdout.log）" -ForegroundColor DarkGray; $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown'); exit 1 }`, runtime.DefaultBin, runtime.DefaultBin, runtime.BinEnv),
		fmt.Sprintf(`Write-Host '正在启动 %s...' -ForegroundColor Cyan`, runtime.Name),
		`Write-Host ''`,
		// Launch the selected runtime. Temporarily raise ErrorActionPreference
		// to 'Stop' so a missing binary / launch failure becomes a CAUGHT,
		// logged terminating error instead of the silent skip we had before.
		`$ErrorActionPreference='Stop'; $code = 0; try { ` + execLine + `; if ($LASTEXITCODE) { $code = $LASTEXITCODE } } catch { Log-Err "runtime exec failed: $_"; Write-Host "⚠️  Agent CLI 启动失败：$_" -ForegroundColor Red; $code = 1 }; $ErrorActionPreference='Continue'`,
		`Write-Host ''`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`Write-Host " Agent CLI 已退出 (退出码 $code). 按任意键关闭窗口" -ForegroundColor Cyan`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')`,
		`exit $code`,
	}, "; ")
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
		`Write-Host ''`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`Write-Host " Agent CLI 已退出 (退出码 $code). 按任意键关闭窗口" -ForegroundColor Cyan`,
		`Write-Host "=============================================" -ForegroundColor Cyan`,
		`$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')`,
		`exit $code`,
	)
	return strings.Join(lines, "; ")
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
	// Phase C: --settings <file> loads the run-scoped hook config. Only the
	// Claude CLI takes it; other runtimes ignore the arg.
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
