package assistanttask

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (d *Dispatcher) commitResult(projectRoot string, task Task, runID, workDir string, manifest inputManifest, bases map[string]AssetSnapshot, result resultManifest) (string, *Result, *Failure) {
	journalPath := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "commit.json")
	journal := commitJournal{SchemaVersion: 1, TaskID: task.ID, RunID: runID, State: "publishing", UpdatedAt: time.Now().UTC()}
	_ = writeJSON(journalPath, journal)
	taskResult := &Result{Summary: result.Summary, AssetUpdates: map[string]string{}}
	failed, updated := 0, 0
	failedOutputs := make([]string, 0)
	for _, key := range d.deps.AssetKeys() {
		update, ok := result.AssetUpdates[key]
		if !ok || update.Status == "unchanged" {
			_ = d.deps.AdvanceAsset(task.ProjectSlug, key, task.ConversationCutoffSeq)
			taskResult.AssetUpdates[key] = "unchanged"
			continue
		}
		if update.Status == "failed" {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			failedOutputs = append(failedOutputs, "“"+key+"”资产候选被任务标记为失败")
			continue
		}
		candidatePath, ok := safeJoin(workDir, update.Candidate)
		if !ok {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			failedOutputs = append(failedOutputs, "“"+key+"”资产候选路径无效")
			continue
		}
		candidate, err := os.ReadFile(candidatePath)
		if err != nil {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			failedOutputs = append(failedOutputs, "“"+key+"”资产候选无法读取")
			continue
		}
		base := bases[key]
		status, payload, err := d.deps.CommitAsset(task.ProjectSlug, AssetCommitInput{Key: key, BaseVersionID: base.VersionID, BaseContent: base.Content, CandidateContent: string(candidate), TaskID: task.ID, RunID: runID, FromSeq: base.ConversationCursor + 1, ThroughSeq: task.ConversationCutoffSeq, SourceRevisionIDs: sourceRevisionIDs(manifest), ChangeSummary: result.Summary})
		if err != nil || status != "updated" {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			failedOutputs = append(failedOutputs, "“"+key+"”资产未能提交")
		} else {
			taskResult.AssetUpdates[key] = "updated"
			updated++
			d.emitAsset(task.ProjectSlug, payload)
		}
	}
	for _, deliverable := range result.Deliverables {
		artifactID, err := d.commitGeneratedArtifact(projectRoot, task, runID, workDir, manifest, deliverable)
		if err != nil {
			failed++
			failedOutputs = append(failedOutputs, "教学成果“"+deliverable.Key+"”未能写入资产库")
			continue
		}
		taskResult.Deliverables = append(taskResult.Deliverables, artifactID)
		updated++
		d.emitGenerated(task.ProjectSlug, task.ID, artifactID)
	}
	if result.SourceUpdate != nil && task.Type == "source-processing" {
		files, media := map[string][]byte{}, map[string]string{}
		for _, file := range result.SourceUpdate.Files {
			path, ok := safeJoin(workDir, file.Path)
			if !ok || !strings.Contains(filepath.ToSlash(file.Path), "source-updates/") {
				failed++
				failedOutputs = append(failedOutputs, "资料解析成果路径无效")
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				failed++
				failedOutputs = append(failedOutputs, "资料解析成果无法读取")
				continue
			}
			files[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = data
			media[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = file.MediaType
		}
		payload, err := d.deps.CommitSourceDerived(task.ProjectSlug, result.SourceUpdate.SourceID, result.SourceUpdate.RevisionID, task.ID, files, media)
		if err != nil {
			failed++
			failedOutputs = append(failedOutputs, "资料解析成果未能写入来源库")
		} else {
			d.emitSource(task.ProjectSlug, payload)
			updated++
		}
	}
	if failed > 0 && updated > 0 {
		failure := &Failure{Code: "partial-publish", Message: "部分助教成果已发布，另有输出未能写入项目", Retryable: false, Suggestion: commitFailureSuggestion(failedOutputs)}
		writeCompletedCommit(journalPath, journal, "partial", taskResult, failure)
		return "partial", taskResult, failure
	}
	if failed > 0 {
		failure := &Failure{Code: "publish-failed", Message: "助教产出未能发布到项目", Retryable: false, Suggestion: commitFailureSuggestion(failedOutputs)}
		writeCompletedCommit(journalPath, journal, "failed", taskResult, failure)
		return "failed", taskResult, failure
	}
	writeCompletedCommit(journalPath, journal, "succeeded", taskResult, nil)
	return "succeeded", taskResult, nil
}

func commitFailureSuggestion(failedOutputs []string) string {
	if len(failedOutputs) == 0 {
		return "建议在 CLI 中检查保存的 prompt、结果清单与相对路径。"
	}
	return "未发布的输出：" + strings.Join(failedOutputs, "；") + "。可在 CLI 中检查本次 attempt 的 result-manifest.json 和输出目录。"
}

func writeCompletedCommit(path string, journal commitJournal, status string, result *Result, failure *Failure) {
	journal.State, journal.Status, journal.Result, journal.Failure, journal.UpdatedAt = "completed", status, result, failure, time.Now().UTC()
	_ = writeJSON(path, journal)
}

func (d *Dispatcher) fail(store *Store, taskID, code, message string, retryable bool) {
	d.finish(store, taskID, "failed", nil, &Failure{Code: code, Message: message, Retryable: retryable, Suggestion: "如要继续，建议在可见 CLI 中检查本次 attempt；需要重新发起时再回到教师对话确认。"})
}
func (d *Dispatcher) finish(store *Store, taskID, status string, result *Result, failure *Failure) {
	task, err := store.Update(taskID, func(t *Task) error {
		t.Status, t.Phase, t.Result, t.Failure, t.Lease = status, "", result, failure, nil
		return nil
	})
	if err == nil {
		d.emit(task)
	}
	d.Notify()
}
func (d *Dispatcher) emitCurrent(store *Store, taskID string) {
	if task, err := store.Get(taskID); err == nil {
		d.emit(task)
	}
}
func (d *Dispatcher) emit(task Task) {
	if d.events != nil {
		d.events.Emit("assistant-task-updated", map[string]any{"projectSlug": task.ProjectSlug, "task": task})
	}
}
func (d *Dispatcher) emitAsset(slug string, asset any) {
	if d.events != nil && asset != nil {
		d.events.Emit("learning-asset-updated", map[string]any{"projectSlug": slug, "asset": asset})
	}
}

func (d *Dispatcher) emitGenerated(slug, taskID, artifactID string) {
	if d.events != nil {
		d.events.Emit("generated-artifact-updated", map[string]any{"projectSlug": slug, "taskId": taskID, "artifactId": artifactID})
	}
}

func (d *Dispatcher) emitSource(slug string, source any) {
	if d.events != nil && source != nil {
		d.events.Emit("source-updated", map[string]any{"projectSlug": slug, "source": source})
	}
}
