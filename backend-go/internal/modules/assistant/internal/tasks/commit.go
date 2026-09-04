package assistanttask

import (
	"os"
	"path/filepath"

	"strings"

	"encoding/json"

	"time"
)

func (d *Dispatcher) commitResult(projectRoot string, task Task, runID, workDir string, manifest inputManifest, bases map[string]AssetSnapshot, result resultManifest) (string, *Result, *Failure) {
	resultBytes, _ := json.Marshal(result)
	journalPath := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "commit.json")
	journal := commitJournal{SchemaVersion: 1, TaskID: task.ID, RunID: runID, State: "committing", ResultHash: hashBytes(resultBytes), UpdatedAt: time.Now().UTC()}
	_ = writeJSON(journalPath, journal)
	taskResult := &Result{Summary: result.Summary, AssetUpdates: map[string]string{}}
	failed, updated := 0, 0
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
			continue
		}
		candidatePath, ok := safeJoin(workDir, update.Candidate)
		if !ok {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			continue
		}
		candidate, err := os.ReadFile(candidatePath)
		if err != nil {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			continue
		}
		base := bases[key]
		status, payload, err := d.deps.CommitAsset(task.ProjectSlug, AssetCommitInput{Key: key, BaseVersionID: base.VersionID, BaseContent: base.Content, CandidateContent: string(candidate), TaskID: task.ID, RunID: runID, FromSeq: base.ConversationCursor + 1, ThroughSeq: task.ConversationCutoffSeq, SourceRevisionIDs: sourceRevisionIDs(manifest), ChangeSummary: result.Summary})
		if err != nil || status != "updated" {
			taskResult.AssetUpdates[key] = "failed"
			failed++
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
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				failed++
				continue
			}
			files[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = data
			media[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = file.MediaType
		}
		payload, err := d.deps.CommitSourceDerived(task.ProjectSlug, result.SourceUpdate.SourceID, result.SourceUpdate.RevisionID, task.ID, files, media)
		if err != nil {
			failed++
		} else {
			d.emitSource(task.ProjectSlug, payload)
			updated++
		}
	}
	if failed > 0 && updated > 0 {
		failure := &Failure{Code: "partial-commit", Message: "部分成果已提交，另有输出未通过校验", Retryable: false, Suggestion: "可在 CLI 中查看本次 attempt 的 result-manifest.json 和输出目录。"}
		writeCompletedCommit(journalPath, journal, "partial", taskResult, failure)
		return "partial", taskResult, failure
	}
	if failed > 0 {
		failure := &Failure{Code: "commit-failed", Message: "助教产出未通过提交校验", Retryable: false, Suggestion: "建议在 CLI 中检查保存的 prompt、结果清单与相对路径。"}
		writeCompletedCommit(journalPath, journal, "failed", taskResult, failure)
		return "failed", taskResult, failure
	}
	writeCompletedCommit(journalPath, journal, "succeeded", taskResult, nil)
	return "succeeded", taskResult, nil
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
