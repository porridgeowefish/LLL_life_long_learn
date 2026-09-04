package claudelauncher

import (
	"fmt"
	"os"

	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"
	"runtime"

	"time"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"path/filepath"
)

func writeAndLaunchInteractiveWrapper(runDirAbs, projectRoot, promptMdPath, stderrPath, settingsFile string, req LaunchRequest) (string, error) {
	wrapperPath := filepath.Join(runDirAbs, "wrapper.sh")
	if runtime.GOOS == "windows" {
		wrapperPath = filepath.Join(runDirAbs, "wrapper.ps1")
	}
	err := launchInteractiveTerminal(interactiveTerminalRequest{
		WorkDir: projectRoot, PromptPath: promptMdPath, WrapperPath: wrapperPath,
		ErrorLogPath: stderrPath, SettingsFile: settingsFile, Launch: req,
		OnExit: func(exitCode int) { finishInteractiveSession(req.Store, req.Events, req.Session.ID, exitCode) },
	})
	return wrapperPath, err
}

// interactiveTerminalRequest is the infrastructure contract behind every
// visible Agent execution. Domain callers go through agentexecution.Service;
// project agents and assistant tasks converge here.
type interactiveTerminalRequest struct {
	WorkDir, PromptPath, WrapperPath, ErrorLogPath, SettingsFile string
	Launch                                                       LaunchRequest
	OnExit                                                       func(int)
}

func launchInteractiveTerminal(req interactiveTerminalRequest) error {
	if runtime.GOOS == "windows" {
		script := buildTUIWrapperScript(req.WorkDir, req.PromptPath, req.ErrorLogPath, req.SettingsFile, req.Launch)
		if err := workspace.AtomicWriteFile(req.WrapperPath, []byte("\ufeff"+script), 0o644); err != nil {
			return fmt.Errorf("write interactive wrapper: %w", err)
		}
		return launchVisibleWindow("powershell.exe", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-NoExit", "-File", req.WrapperPath}, req.WorkDir, req.OnExit)
	}
	script := buildTUIWrapperShellScript(req.WorkDir, req.PromptPath, req.ErrorLogPath, req.SettingsFile, req.Launch)
	if err := workspace.AtomicWriteFile(req.WrapperPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write interactive wrapper: %w", err)
	}
	return launchVisibleWindow("/bin/sh", []string{req.WrapperPath}, req.WorkDir, req.OnExit)
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
			func(exitCode int) {
				if req.Session != nil {
					finishInteractiveSession(req.Store, req.Events, req.Session.ID, exitCode)
				}
			},
		)
	}

	shCmd := buildResumeWrapperShellScript(projectRoot, stderrPath, req)
	wrapperShPath := filepath.Join(runDirAbs, "wrapper.sh")
	if err := os.WriteFile(wrapperShPath, []byte(shCmd), 0o755); err != nil {
		return wrapperShPath, fmt.Errorf("write wrapper.sh: %w", err)
	}
	return wrapperShPath, launchVisibleWindow("/bin/sh", []string{wrapperShPath}, projectRoot,
		func(exitCode int) {
			if req.Session != nil {
				finishInteractiveSession(req.Store, req.Events, req.Session.ID, exitCode)
			}
		})
}

func finishInteractiveSession(store *sessionstore.Store, events EventEmitter, sessionID string, exitCode int) {
	if store == nil || sessionID == "" {
		return
	}
	state := sessionstore.StateCompleted
	eventName := "session-completed"
	if exitCode != 0 {
		state = sessionstore.StateFailed
		eventName = "session-failed"
	}
	now := time.Now().UTC()
	changed := false
	store.Update(sessionID, func(sess *sessionstore.Session) {
		switch sess.State {
		case sessionstore.StatePreparing, sessionstore.StateLaunching, sessionstore.StateRunning, sessionstore.StateAwaitingFollowup:
			sess.State = state
			sess.FinishedAt = &now
			sess.ExitCode = &exitCode
			changed = true
		}
	})
	if changed && events != nil {
		events.Emit(eventName, map[string]any{"runId": sessionID, "exitCode": exitCode})
	}
}
