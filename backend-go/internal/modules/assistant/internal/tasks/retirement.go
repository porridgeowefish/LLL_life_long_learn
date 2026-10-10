package assistanttask

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

// RetireLegacyTasks runs before the asset cutover and before any dispatcher.
// A live or recently launched CLI blocks the entire project without mutations.
func RetireLegacyTasks(slug string) error {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return err
	}
	base := filepath.Join(root, "assistant-tasks")
	var tasks []string
	err = filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) && path == base {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("linked assistant task directory blocks cutover")
		}
		if info.IsDir() {
			return nil
		}
		switch info.Name() {
		case "task.json":
			tasks = append(tasks, path)
		case "executor.json":
			var record executorRecord
			if err := readJSON(path, &record); err != nil {
				return err
			}
			if processMatches(record.PID, record.StartedAt) {
				return errors.New("old assistant CLI is still running")
			}
		case "launch.json":
			var record launchRecord
			if err := readJSON(path, &record); err != nil {
				return err
			}
			if time.Since(record.CreatedAt) < launchRecordGrace {
				var ended exitRecord
				if readJSON(filepath.Join(filepath.Dir(path), "exit.json"), &ended) != nil {
					return errors.New("old assistant CLI may still be launching")
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Persist cancellation first: recovery must never publish a retired attempt.
	for _, path := range tasks {
		var task Task
		if err := readJSON(path, &task); err != nil {
			return err
		}
		if !IsTerminal(task.Status) || !ValidType(task.Type) || task.Status == "failed" {
			task.Status, task.Phase, task.Lease = "cancelled", "", nil
			task.Failure = &Failure{Code: "iteration-18-retired", Message: "旧任务已退役，请在教师对话重新确认复习稿或练习题任务"}
		}
		task.Result = nil
		task.UpdatedAt = time.Now().UTC()
		if err := writeJSON(path, task); err != nil {
			return err
		}
		attempts := filepath.Join(filepath.Dir(path), "attempts")
		err := filepath.Walk(attempts, func(path string, info os.FileInfo, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() && (info.Name() == "deliverables" || info.Name() == "asset-updates") {
				relative, err := filepath.Rel(base, path)
				if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
					return errors.New("unsafe retired task path")
				}
				if err := os.RemoveAll(path); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			if !info.IsDir() && (info.Name() == "result-manifest.json" || info.Name() == "commit.json") {
				return os.Remove(path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
