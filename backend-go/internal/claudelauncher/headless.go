package claudelauncher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentruntime"
	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// LaunchHeadless runs a single-shot background task for product workflows
// that must not expose the interactive Claude terminal to the learner.
// The model parameter, when non-empty, injects a --model flag before --permission-mode.
func LaunchHeadless(
	ctx context.Context,
	projectSlug string,
	agentID string,
	claudeBin string,
	model string,
	pkg *promptassembly.Package,
	runtime *agentruntime.Runtime,
) error {
	projectRoot, err := workspace.ProjectRootForSlug(projectSlug)
	if err != nil {
		return err
	}
	runDir := filepath.Join(projectRoot, "runs", pkg.RunDirName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "prompt.md"), []byte(pkg.PromptMd), 0o644); err != nil {
		return err
	}
	meta, err := pkg.MarshalPackageMeta()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "package.json"), meta, 0o644); err != nil {
		return err
	}

	stdout, err := os.Create(filepath.Join(runDir, "stdout.log"))
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(runDir, "stderr.log"))
	if err != nil {
		return err
	}
	defer stderr.Close()

	startedAt := time.Now().UTC()
	rt := headlessRuntime(claudeBin, runtime)
	if !rt.SupportsHeadless {
		return fmt.Errorf("%s does not support headless execution yet", rt.Name)
	}
	args := headlessArgs(rt, projectRoot, pkg.PromptMd)
	if rt.ID == agentruntime.RuntimeClaude && model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, rt.Bin, args...)
	if rt.Mode == "wsl" {
		var wslProject string
		if out, err := exec.CommandContext(ctx, "wsl.exe", "wslpath", "-a", projectRoot).Output(); err == nil {
			wslProject = strings.TrimSpace(string(out))
		}
		if wslProject == "" {
			wslProject = projectRoot
		}
		cmd = exec.CommandContext(ctx, "wsl.exe", "-e", "sh", "-lc", "cd "+shQuote(wslProject)+" && "+headlessShellCommand(rt, model))
	}
	cmd.Dir = projectRoot
	if headlessUsesStdin(rt) {
		cmd.Stdin = strings.NewReader(pkg.PromptMd)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	configureHeadlessCommand(cmd)
	runErr := cmd.Run()

	exitCode := 0
	if runErr != nil {
		exitCode = -1
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	runMeta := map[string]any{
		"agentId":        agentID,
		"projectSlug":    projectSlug,
		"zoneName":       workspace.ZonePractice,
		"mode":           "headless-single-shot",
		"runtimeId":      rt.ID,
		"runtimeBin":     rt.Bin,
		"permissionMode": NormalizePermissionMode(""),
		"exitCode":       exitCode,
		"startedAt":      startedAt,
		"finishedAt":     time.Now().UTC(),
	}
	raw, _ := json.MarshalIndent(runMeta, "", "  ")
	_ = os.WriteFile(filepath.Join(runDir, "run.json"), raw, 0o644)
	if runErr != nil {
		return fmt.Errorf("headless claude evaluation: %w", runErr)
	}
	return nil
}

func headlessRuntime(claudeBin string, runtime *agentruntime.Runtime) agentruntime.Runtime {
	if runtime != nil {
		return *runtime
	}
	def, _ := agentruntime.DefinitionByID(agentruntime.RuntimeClaude)
	if claudeBin == "" {
		claudeBin = def.DefaultBin
	}
	return agentruntime.Runtime{Definition: def, Bin: claudeBin, Available: true}
}

func headlessArgs(rt agentruntime.Runtime, projectRoot, prompt string) []string {
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		return []string{"-p", "--permission-mode", NormalizePermissionMode("")}
	case agentruntime.RuntimeCodex:
		return []string{"exec", "--yolo", "-", "--cd", projectRoot}
	case agentruntime.RuntimeHermes:
		return []string{"-z", prompt}
	case agentruntime.RuntimeTrae:
		return []string{"run", prompt, "--working-dir", projectRoot}
	default:
		return nil
	}
}

func headlessUsesStdin(rt agentruntime.Runtime) bool {
	return rt.ID == agentruntime.RuntimeClaude || rt.ID == agentruntime.RuntimeCodex
}

func headlessShellCommand(rt agentruntime.Runtime, model string) string {
	switch rt.ID {
	case agentruntime.RuntimeClaude:
		args := []string{shQuote(rt.Bin), "-p", "--permission-mode", shQuote(NormalizePermissionMode(""))}
		if model != "" {
			args = append(args, "--model", shQuote(model))
		}
		return strings.Join(args, " ")
	case agentruntime.RuntimeCodex:
		return shQuote(rt.Bin) + " exec --yolo - --cd ."
	default:
		return shQuote(rt.Bin)
	}
}
