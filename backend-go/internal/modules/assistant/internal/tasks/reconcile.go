package assistanttask

import (
	"fmt"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"os"
	"path/filepath"
	"time"
)

func (d *Dispatcher) reconcileLostRuns() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("running")
		for _, task := range tasks {
			d.reconcileRunningTask(store, task)
		}
	}
}

// reconcileLateResults repairs the durable boundary between a visible CLI and
// LLL. A terminal window may outlive the process handle observed by a previous
// server instance and write its final manifest later. It also repairs an
// artifact-only result whose prior promotion failed after the manifest had
// already been validated. A valid sealed result is therefore committed
// idempotently instead of remaining stranded inside the attempt workspace.
func (d *Dispatcher) reconcileLateResults() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("failed")
		for _, task := range tasks {
			if task.Failure == nil || (task.Failure.Code != "executor-state-lost" && task.Failure.Code != "invalid-result-manifest" && task.Failure.Code != "commit-failed" && task.Failure.Code != "artifact-recovery-failed") || len(task.AttemptIDs) == 0 {
				continue
			}
			runID := task.AttemptIDs[len(task.AttemptIDs)-1]
			projectRoot, rootErr := workspace.ProjectRootForSlug(task.ProjectSlug)
			if rootErr != nil {
				continue
			}
			attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
			resultPath := filepath.Join(attemptDir, "workspace", "result-manifest.json")
			if !stableResultManifest(attemptDir) {
				continue
			}
			var journal commitJournal
			if readJSON(filepath.Join(attemptDir, "commit.json"), &journal) == nil && journal.State == "completed" && task.Failure.Code != "commit-failed" && task.Failure.Code != "artifact-recovery-failed" {
				continue
			}
			workDir := filepath.Join(attemptDir, "workspace")
			manifest, bases, loadErr := loadSealedInputs(task, projectRoot, workDir)
			if loadErr != nil {
				continue
			}
			result, readErr := readResult(resultPath, task.ID, runID)
			if readErr != nil || validateResult(workDir, task, manifest, result) != nil {
				continue
			}
			if task.Failure.Code == "commit-failed" && !isArtifactOnlyResult(task, result) {
				continue
			}
			if task.Failure.Code == "artifact-recovery-failed" {
				taskResult, artifactIDs, ok := promotedArtifactResult(projectRoot, task, runID, result)
				if !ok {
					continue
				}
				recovered, updateErr := store.Update(task.ID, func(current *Task) error {
					current.Status, current.Phase, current.Result, current.Failure = "succeeded", "", taskResult, nil
					return nil
				})
				if updateErr == nil {
					for _, artifactID := range artifactIDs {
						d.emitGenerated(task.ProjectSlug, task.ID, artifactID)
					}
					d.emit(recovered)
					d.Notify()
				}
				continue
			}
			_, _ = store.Update(task.ID, func(current *Task) error {
				current.Status, current.Phase, current.Result, current.Failure = "running", "committing", nil, nil
				return nil
			})
			d.emitCurrent(store, task.ID)
			status, taskResult, failure := d.commitResult(projectRoot, task, runID, workDir, manifest, bases, result)
			if task.Failure.Code == "commit-failed" && status == "failed" && failure != nil && failure.Code == "commit-failed" {
				failure.Code = "artifact-recovery-failed"
				failure.Message = "助教成果再次提交失败，尚未写入项目资产库"
			}
			d.finish(store, task.ID, status, taskResult, failure)
		}
	}
}

func isArtifactOnlyResult(task Task, result resultManifest) bool {
	if task.Type != "produce-material" || len(result.Deliverables) == 0 || result.SourceUpdate != nil {
		return false
	}
	for _, key := range []string{"intro", "body", "practice"} {
		update, ok := result.AssetUpdates[key]
		if !ok || update.Status != "unchanged" {
			return false
		}
	}
	return true
}

func promotedArtifactResult(projectRoot string, task Task, runID string, result resultManifest) (*Result, []string, bool) {
	if !isArtifactOnlyResult(task, result) {
		return nil, nil, false
	}
	taskResult := &Result{Summary: result.Summary, AssetUpdates: map[string]string{}}
	for key, update := range result.AssetUpdates {
		taskResult.AssetUpdates[key] = update.Status
	}
	for _, deliverable := range result.Deliverables {
		artifactID := artifactIDFor(task.ID, runID, deliverable.Key)
		if !artifactOwnedBy(filepath.Join(projectRoot, "assets", "generated", artifactID), task.ID, runID) {
			return nil, nil, false
		}
		taskResult.Deliverables = append(taskResult.Deliverables, artifactID)
	}
	return taskResult, append([]string(nil), taskResult.Deliverables...), true
}

func (d *Dispatcher) reconcilePartialDeliverables() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("partial")
		for _, task := range tasks {
			if task.Failure == nil || task.Failure.Code != "partial-commit" || task.Result == nil || len(task.AttemptIDs) == 0 {
				continue
			}
			runID := task.AttemptIDs[len(task.AttemptIDs)-1]
			projectRoot, rootErr := workspace.ProjectRootForSlug(task.ProjectSlug)
			if rootErr != nil {
				continue
			}
			workDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "workspace")
			manifest, _, loadErr := loadSealedInputs(task, projectRoot, workDir)
			result, readErr := readResult(filepath.Join(workDir, "result-manifest.json"), task.ID, runID)
			if loadErr != nil || readErr != nil || validateResult(workDir, task, manifest, result) != nil || result.SourceUpdate != nil {
				continue
			}
			cleanAssets := true
			for _, status := range task.Result.AssetUpdates {
				if status == "failed" {
					cleanAssets = false
				}
			}
			if !cleanAssets {
				continue
			}
			deliverables := append([]string(nil), task.Result.Deliverables...)
			allCommitted := true
			for _, declared := range result.Deliverables {
				artifactID, commitErr := d.commitGeneratedArtifact(projectRoot, task, runID, workDir, manifest, declared)
				if commitErr != nil {
					allCommitted = false
					continue
				}
				if !containsString(deliverables, artifactID) {
					deliverables = append(deliverables, artifactID)
				}
			}
			if !allCommitted {
				continue
			}
			recovered, updateErr := store.Update(task.ID, func(current *Task) error {
				current.Status, current.Failure = "succeeded", nil
				current.Result.Deliverables = deliverables
				return nil
			})
			if updateErr == nil {
				d.emit(recovered)
			}
		}
	}
}

func (d *Dispatcher) reconcileRunningTask(store *Store, task Task) {
	if len(task.AttemptIDs) == 0 {
		d.fail(store, task.ID, "executor-state-lost", "运行中的任务没有可恢复的 attempt", false)
		return
	}
	runID := task.AttemptIDs[len(task.AttemptIDs)-1]
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		d.fail(store, task.ID, "workspace-unavailable", "无法打开学习单元工作区", false)
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	var finished exitRecord
	if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
		go d.recoverAttempt(store, task, runID, finished.ExitCode)
		return
	}

	if stableResultManifest(attemptDir) {
		go d.recoverAttempt(store, task, runID, 0)
		return
	}
	var executor executorRecord
	if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) == nil && processMatches(executor.PID, executor.StartedAt) {
		_, _ = store.Update(task.ID, func(current *Task) error {
			if current.Lease == nil {
				current.Lease = &Lease{RunID: runID, AcquiredAt: executor.StartedAt}
			}
			current.Lease.DispatcherInstance = d.instance
			current.Lease.LastHeartbeatAt = time.Now().UTC()
			return nil
		})
		go d.monitorRecovered(store, task, runID, executor.PID)
		return
	}
	var launch launchRecord
	if readJSON(filepath.Join(attemptDir, "launch.json"), &launch) == nil && launch.SchemaVersion == 1 && launch.TaskID == task.ID && launch.RunID == runID {
		deadline := launch.CreatedAt.Add(launchRecordGrace)
		if time.Now().UTC().Before(deadline) {
			go d.monitorLaunching(store, task, runID, deadline)
			return
		}
	}
	d.fail(store, task.ID, "executor-state-lost", "应用重启后确认原 CLI 进程已经结束，且没有可提交结果", false)
}

func (d *Dispatcher) monitorLaunching(store *Store, task Task, runID string, deadline time.Time) {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			var finished exitRecord
			if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
				d.recoverAttempt(store, task, runID, finished.ExitCode)
				return
			}
			if stableResultManifest(attemptDir) {
				d.recoverAttempt(store, task, runID, 0)
				return
			}
			var executor executorRecord
			if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) == nil && processMatches(executor.PID, executor.StartedAt) {
				d.monitorRecovered(store, task, runID, executor.PID)
				return
			}
			if !time.Now().UTC().Before(deadline) {
				d.fail(store, task.ID, "executor-state-lost", "CLI 启动记录存在，但在宽限期内没有出现可观察执行器", false)
				return
			}
		}
	}
}

func (d *Dispatcher) monitorRecovered(store *Store, task Task, runID string, pid int) {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			var finished exitRecord
			if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
				d.recoverAttempt(store, task, runID, finished.ExitCode)
				return
			}
			if stableResultManifest(attemptDir) {
				d.recoverAttempt(store, task, runID, 0)
				return
			}
			var executor executorRecord
			if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) != nil || !processMatches(pid, executor.StartedAt) {
				if _, err := os.Stat(filepath.Join(attemptDir, "workspace", "result-manifest.json")); err == nil {
					d.recoverAttempt(store, task, runID, 0)
				} else {
					d.fail(store, task.ID, "executor-state-lost", "恢复观察的 CLI 进程已经结束，且没有写出结果", false)
				}
				return
			}
			_, _ = store.Update(task.ID, func(current *Task) error {
				if current.Lease != nil {
					current.Lease.LastHeartbeatAt = time.Now().UTC()
				}
				return nil
			})
		}
	}
}

func (d *Dispatcher) recoverAttempt(store *Store, task Task, runID string, exitCode int) {
	current, err := store.Get(task.ID)
	if err != nil || IsTerminal(current.Status) {
		return
	}
	if exitCode == 130 || exitCode == -1073741510 {
		d.finish(store, task.ID, "cancelled", nil, &Failure{Code: "terminal-interrupted", Message: "任务已在 CLI 终端中止", Retryable: false, Suggestion: "如需重试，请在教师对话中重新确认任务，或在 CLI 中手动执行保存的 prompt.md。"})
		return
	}
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		d.fail(store, task.ID, "workspace-unavailable", "无法打开学习单元工作区", false)
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	workDir := filepath.Join(attemptDir, "workspace")
	var committed commitJournal
	if readJSON(filepath.Join(attemptDir, "commit.json"), &committed) == nil && committed.SchemaVersion == 1 && committed.TaskID == task.ID && committed.RunID == runID && committed.State == "completed" && IsTerminal(committed.Status) {
		d.finish(store, task.ID, committed.Status, committed.Result, committed.Failure)
		return
	}
	manifest, bases, err := loadSealedInputs(task, projectRoot, workDir)
	if err != nil {
		d.fail(store, task.ID, "input-recovery-failed", "无法恢复助教任务的封存输入", false)
		return
	}
	_, _ = store.Update(task.ID, func(current *Task) error { current.Phase = "validating"; return nil })
	d.emitCurrent(store, task.ID)
	result, err := readResult(filepath.Join(workDir, "result-manifest.json"), task.ID, runID)
	if err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "CLI 未产生有效的 result-manifest.json", false)
		return
	}
	if err := validateResult(workDir, task, manifest, result); err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "恢复的 CLI 结果清单或声明产物未通过工作区校验", false)
		return
	}
	if exitCode != 0 {
		d.fail(store, task.ID, "executor-exit-failed", fmt.Sprintf("CLI 以状态 %d 结束", exitCode), false)
		return
	}
	_, _ = store.Update(task.ID, func(current *Task) error { current.Phase = "committing"; return nil })
	d.emitCurrent(store, task.ID)
	status, taskResult, failure := d.commitResult(projectRoot, task, runID, workDir, manifest, bases, result)
	d.finish(store, task.ID, status, taskResult, failure)
}
