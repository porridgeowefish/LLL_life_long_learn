package claudelauncher

import (
	promptassembly "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/prompt"

	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"
	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
	runprogress "github.com/xmz14/lll/backend-go/internal/modules/learning"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"path/filepath"

	"strings"

	"fmt"
	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"
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
	// RunProgress registers every interactive runtime for authenticated exit
	// reporting. Claude additionally receives a run-scoped settings file whose
	// PostToolUse/Stop hooks provide granular progress.
	RunProgress *runprogress.RunStore
	runToken    string
	// taskExecutorPath/taskExitPath are optional durable markers used by the
	// assistant-task dispatcher. They are deliberately part of the same
	// interactive launcher used by project agents, rather than a second CLI
	// execution implementation.
	taskExecutorPath string
	taskExitPath     string
	taskEnv          map[string]string
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
	RunProgress *runprogress.RunStore
	runToken    string
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

// LaunchTaskTerminal opens the configured Agent CLI in a real, visible
// terminal for one assistant-task attempt. The durable UTF-8 prompt file is
// loaded by the visible wrapper and injected as the CLI's initial interactive
// turn. The CLI works solely in attemptWorkspace and reports through
// result-manifest.json.
func LaunchTaskTerminal(rt agentruntime.Runtime, attemptWorkspace, promptPath, wrapperPath string, env map[string]string, onExit func(int)) error {
	if !rt.Available {
		return fmt.Errorf("agent runtime %s is unavailable", rt.ID)
	}
	attemptDir := filepath.Dir(promptPath)
	req := LaunchRequest{
		ProjectSlug:      "助教任务",
		ZoneName:         workspace.ZoneName("助教工作区"),
		Agent:            &agentregistry.Agent{ID: "assistant-task", Name: "助教", Icon: "A"},
		PermissionMode:   "bypassPermissions",
		Runtime:          &rt,
		taskExecutorPath: filepath.Join(attemptDir, "executor.json"),
		taskExitPath:     filepath.Join(attemptDir, "exit.json"),
		taskEnv:          env,
	}
	return launchInteractiveTerminal(interactiveTerminalRequest{
		WorkDir:      attemptWorkspace,
		PromptPath:   promptPath,
		WrapperPath:  wrapperPath,
		ErrorLogPath: filepath.Join(attemptDir, "terminal.log"),
		Launch:       req,
		OnExit:       onExit,
	})
}

func buildTaskWrapperPowerShell(rt agentruntime.Runtime, workDir, promptPath string, env map[string]string) string {
	attemptDir := filepath.Dir(promptPath)
	return buildTUIWrapperScript(workDir, promptPath, filepath.Join(attemptDir, "terminal.log"), "", LaunchRequest{
		ProjectSlug: "助教任务", ZoneName: workspace.ZoneName("助教工作区"),
		Agent:          &agentregistry.Agent{ID: "assistant-task", Name: "助教", Icon: "A"},
		PermissionMode: "bypassPermissions", Runtime: &rt,
		taskExecutorPath: filepath.Join(attemptDir, "executor.json"), taskExitPath: filepath.Join(attemptDir, "exit.json"), taskEnv: env,
	})
}

func buildTaskWrapperShell(rt agentruntime.Runtime, workDir, promptPath string, env map[string]string) string {
	attemptDir := filepath.Dir(promptPath)
	return buildTUIWrapperShellScript(workDir, promptPath, filepath.Join(attemptDir, "terminal.log"), "", LaunchRequest{
		ProjectSlug: "助教任务", ZoneName: workspace.ZoneName("助教工作区"),
		Agent:          &agentregistry.Agent{ID: "assistant-task", Name: "助教", Icon: "A"},
		PermissionMode: "bypassPermissions", Runtime: &rt,
		taskExecutorPath: filepath.Join(attemptDir, "executor.json"), taskExitPath: filepath.Join(attemptDir, "exit.json"), taskEnv: env,
	})
}
