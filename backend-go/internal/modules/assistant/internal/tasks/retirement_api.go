package assistanttask

import (
	"errors"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RetireForAPIGeneration preserves all historical/candidate files. It runs
// before any dispatcher is started and uses a distinct resumable journal.
func RetireForAPIGeneration(slug string) error {
	root, e := workspace.ProjectRootForSlug(slug)
	if e != nil {
		return e
	}
	journal := filepath.Join(root, "migrations", "iteration-22.json")
	var mark struct {
		State string `json:"state"`
	}
	if readJSON(journal, &mark) == nil && mark.State == "completed" {
		return nil
	}
	var taskPaths []string
	for _, base := range []string{filepath.Join(root, "assistant-tasks"), filepath.Join(root, "runs")} {
		e = filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) && path == base {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.New("linked legacy runtime blocks cutover")
			}
			if info.IsDir() {
				return nil
			}
			name := info.Name()
			if name == "task.json" {
				taskPaths = append(taskPaths, path)
			}
			if name == "executor.json" || strings.HasSuffix(name, ".executor.json") {
				var r executorRecord
				if e := readJSON(path, &r); e != nil {
					return e
				}
				if processAlive(r.PID) {
					if r.StartedAt.IsZero() {
						return errors.New("legacy writer identity unavailable")
					}
					if processMatches(r.PID, r.StartedAt) {
						if e := stopVerifiedExecutor(r.PID, r.StartedAt); e != nil {
							return e
						}
					}
				}
			}
			if name == "launch.json" {
				var r launchRecord
				if e := readJSON(path, &r); e != nil {
					return e
				}
				if time.Since(r.CreatedAt) < launchRecordGrace {
					var ended exitRecord
					if readJSON(filepath.Join(filepath.Dir(path), "exit.json"), &ended) != nil {
						return errors.New("legacy runtime may still be launching")
					}
				}
			}
			return nil
		})
		if e != nil {
			_ = os.MkdirAll(filepath.Dir(journal), 0755)
			_ = writeJSON(journal, map[string]any{"schemaVersion": 1, "state": "migration-blocked", "reason": "legacy-writer", "updatedAt": time.Now().UTC()})
			return e
		}
	}
	for _, path := range taskPaths {
		var task Task
		if e = readJSON(path, &task); e != nil {
			return e
		}
		if IsTerminal(task.Status) {
			continue
		}
		task.Status = "cancelled"
		task.Phase = ""
		task.Lease = nil
		task.Failure = &Failure{Code: "learning-cli-retired", Message: "旧任务已退休，请使用API生成入口"}
		task.UpdatedAt = time.Now().UTC()
		if e = writeJSON(path, task); e != nil {
			return e
		}
	}
	if e = os.MkdirAll(filepath.Dir(journal), 0755); e != nil {
		return e
	}
	return writeJSON(journal, map[string]any{"schemaVersion": 1, "state": "completed", "version": "iteration-22", "updatedAt": time.Now().UTC()})
}
