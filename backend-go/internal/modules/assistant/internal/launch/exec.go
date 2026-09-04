package claudelauncher

import (
	"strings"

	"encoding/json"
	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"

	"fmt"
	"os"

	"context"
	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"

	"errors"

	"time"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"path/filepath"
)

// Launch spawns the selected agent runtime in an interactive PowerShell window.
// The function returns immediately after the wrapper is started, while a
// background process-handle waiter reconciles the session when the terminal
// exits or is closed.
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
	// docs/99-archive/iterations/iteration-06-ask-ai-and-live-progress/DELIVERY_NOTES.md.
	var settingsFile string
	if req.RunProgress != nil && runtime.ID == agentruntime.RuntimeClaude {
		sf, token, herr := writeHookSettings(runDirAbs, req.Session.ID, req.RunProgress)
		req.runToken = token
		if herr != nil {

			println("runprogress: write hook settings warning:", herr.Error())
		} else {
			settingsFile = sf
		}
	} else if req.RunProgress != nil {
		req.runToken = req.RunProgress.Register(req.Session.ID)
	}

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

	_ = os.WriteFile(filepath.Join(runDirAbs, "result.md"), []byte(""), 0o644)

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
	if req.Session != nil && req.RunProgress != nil {
		req.runToken = req.RunProgress.Register(req.Session.ID)
	}
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
