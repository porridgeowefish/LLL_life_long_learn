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
	RunProgress    *runprogress.Store
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
	psCmd := buildTUIWrapperScript(projectRoot, promptMdPath, stdoutPath, settingsFile, req)

	// 写 wrapper 脚本到 runDir/wrapper.ps1，用 -File 启动 —— 规避 -Command
	// 内联引号地狱（中文项目路径 / 单引号 / `& 'claude'` 全部免转义）。
	wrapperPs1Path := filepath.Join(runDirAbs, "wrapper.ps1")
	// Write wrapper.ps1 with a UTF-8 BOM. PowerShell 5.1 decodes a .ps1 with
	// no BOM using the system ANSI codepage (GBK/CP936 on Chinese Windows),
	// which mangles every Chinese path in the script — e.g. the project root
	// projects/金融投资 is read as projects/閲戣瀺鎶曲祫, so Set-Location and
	// the prompt.md Get-Content both fail and Claude launches with an empty
	// prompt. The BOM forces PS to decode the script as UTF-8. Verified by a
	// control test: no-BOM Set-Location fails with the garbled path; with-BOM
	// it resolves the Chinese path correctly. See LESSONS_LEARNED (encoding).
	if err := os.WriteFile(wrapperPs1Path, []byte("\ufeff"+psCmd), 0o644); err != nil {
		return nil, fmt.Errorf("write wrapper.ps1: %w", err)
	}

	// 用 ShellExecute("open", powershell.exe, ...) 开可见窗口。
	// 根因（SO #30182508 / terraform-exec#570，已 repro + 用户端到端确认）：
	// os/exec（CreateProcess）的子控制台窗口可见性取决于父进程控制台分配；
	// 服务进程（go run 起、stdio 重定向）即使用 CREATE_NEW_CONSOLE 或
	// cmd /c start 也拿不到可见窗口——子进程在跑但窗口永不出现。ShellExecute
	// 走 Windows Shell（等同 Explorer 双击），强制在交互桌面开可见窗口。
	// wrapper 脚本仍落 wrapper.ps1，用 -File 启动规避 -Command 引号地狱。
	if err := launchVisibleWindow("powershell.exe",
		[]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-NoExit", "-File", wrapperPs1Path},
		projectRoot,
	); err != nil {
		// Persist the spawn failure to stderr.log so the user can see
		// why nothing happened (instead of a silent failure).
		_ = os.WriteFile(stderrPath, []byte("launch window: "+err.Error()+"\nwrapperPs1: "+wrapperPs1Path+"\n"), 0o644)
		req.Store.SetFinished(req.Session.ID, sessionstore.StateFailed, -1)
		if req.Events != nil {
			req.Events.Emit("session-failed", map[string]any{
				"sessionId": req.Session.ID, "error": err.Error(),
			})
		}
		return nil, fmt.Errorf("spawn claude wrapper: %w", err)
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
	execLine := interactiveExecLine(runtime, projectRoot, settingsFile)
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

func interactiveExecLine(rt agentruntime.Runtime, projectRoot, settingsFile string) string {
	// Phase C: --settings <file> loads the run-scoped hook config. Only the
	// Claude CLI takes it; other runtimes ignore the arg.
	settingsArg := ""
	if strings.TrimSpace(settingsFile) != "" {
		settingsArg = fmt.Sprintf(" --settings '%s'", settingsFile)
	}
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		return "if ($promptText) { & $agentExe --permission-mode $permissionMode" + settingsArg + " $promptText } else { & $agentExe --permission-mode $permissionMode" + settingsArg + " }"
	case agentruntime.RuntimeCodex:
		return "if ($promptText) { & $agentExe $promptText } else { & $agentExe }"
	case agentruntime.RuntimeTrae:
		return fmt.Sprintf("if ($promptText) { & $agentExe run $promptText --working-dir '%s' } else { & $agentExe interactive }", projectRoot)
	default:
		return "& $agentExe"
	}
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
