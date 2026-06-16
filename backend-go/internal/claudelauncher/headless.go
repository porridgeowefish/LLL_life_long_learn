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

	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// LaunchHeadless runs a single-shot background task for product workflows
// that must not expose the interactive Claude terminal to the learner.
func LaunchHeadless(
	ctx context.Context,
	projectSlug string,
	agentID string,
	claudeBin string,
	pkg *promptassembly.Package,
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
	cmd := exec.CommandContext(ctx, claudeBin, "-p", "--permission-mode", NormalizePermissionMode(""))
	cmd.Dir = projectRoot
	cmd.Stdin = strings.NewReader(pkg.PromptMd)
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
