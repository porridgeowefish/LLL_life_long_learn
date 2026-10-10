package assistanttask

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	assetstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	preferencestore "github.com/xmz14/lll/backend-go/internal/modules/preferences"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

func newTestDispatcher(execution ExecutionService, events EventEmitter) *Dispatcher {
	d := NewDispatcher(execution, events)
	d.Configure(Dependencies{
		ConversationSnapshot: func(slug string, through uint64) ([]byte, error) {
			store, err := teacher.NewConversation(slug)
			if err != nil {
				return nil, err
			}
			projection, err := store.SnapshotThrough(through)
			if err != nil {
				return nil, err
			}
			return json.MarshalIndent(projection, "", "  ")
		},
		PreferencesSnapshot: func() (string, []byte, error) {
			snapshot, err := preferencestore.Read()
			return preferencestore.Filename, []byte(snapshot.Content), err
		},
		AssetKeys: assetstore.CoreKeys,
		ReadAsset: func(slug, key string) (AssetSnapshot, error) {
			store, err := assetstore.New(slug)
			if err != nil {
				return AssetSnapshot{}, err
			}
			asset, err := store.Get(key)
			return AssetSnapshot{VersionID: asset.Meta.CurrentVersionID, ConversationCursor: asset.Meta.ConversationCursor, Content: asset.Content}, err
		},
		AdvanceAsset: func(slug, key string, through uint64) error {
			store, err := assetstore.New(slug)
			if err != nil {
				return err
			}
			_, err = store.AdvanceCursor(key, through)
			return err
		},
		CommitAsset: func(slug string, input AssetCommitInput) (string, any, error) {
			store, err := assetstore.New(slug)
			if err != nil {
				return "", nil, err
			}
			result, err := store.CommitCandidate(assetstore.CandidateInput{Key: input.Key, BaseVersionID: input.BaseVersionID, BaseContent: input.BaseContent, CandidateContent: input.CandidateContent, TaskID: input.TaskID, RunID: input.RunID, FromSeq: input.FromSeq, ThroughSeq: input.ThroughSeq, SourceRevisionIDs: input.SourceRevisionIDs, ChangeSummary: input.ChangeSummary})
			return result.Status, result.Asset.Meta, err
		},
		SealSource: func(slug, sourceID, destination string) (SourceSnapshot, error) {
			store, err := sourcestore.New(slug)
			if err != nil {
				return SourceSnapshot{}, err
			}
			_, revision, err := store.Get(sourceID)
			if err != nil {
				return SourceSnapshot{}, err
			}
			root, err := workspace.ProjectRootForSlug(slug)
			if err != nil {
				return SourceSnapshot{}, err
			}
			if err := copyTree(filepath.Join(root, "sources", sourceID, "revisions", revision.RevisionID), filepath.Join(destination, revision.RevisionID)); err != nil {
				return SourceSnapshot{}, err
			}
			return SourceSnapshot{RevisionID: revision.RevisionID, SHA256: revision.Original.SHA256}, nil
		},
	})
	return d
}

func TestRunningVisibleTerminalCommitsStableManifestBeforeTerminalExit(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("live-terminal", "可见终端", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, _ := New("live-terminal")
	task, _, err := store.Create(CreateInput{Type: "consolidate", Objective: "整理资产", Origin: Origin{OperationID: "op_live"}})
	if err != nil {
		t.Fatal(err)
	}
	runID := "run_live"
	task, _ = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase, current.AttemptIDs = "running", "executing", []string{runID}
		return nil
	})
	projectRoot, _ := workspace.ProjectRootForSlug("live-terminal")
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	workDir := filepath.Join(attemptDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workDir, "asset-updates", "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := newTestDispatcher(nil, nil)
	defer close(d.stop)
	manifest, _, err := d.sealInputs(task, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "asset-updates", "review", "current.md"), []byte("终端仍打开时已经完成的正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := resultManifest{SchemaVersion: 1, TaskID: task.ID, RunID: runID, Summary: "成果已经写完", AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"review": {Status: "updated", Candidate: "asset-updates/review/current.md"}, "practice": {Status: "unchanged"}}}
	resultPath := filepath.Join(workDir, "result-manifest.json")
	if err := writeJSON(resultPath, result); err != nil {
		t.Fatal(err)
	}
	stableAt := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(resultPath, stableAt, stableAt); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(attemptDir, "executor.json"), executorRecord{SchemaVersion: 1, PID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}

	d.reconcileRunningTask(store, task)
	// Reconciliation is asynchronous and now commits two required learning
	// assets (intro and body).  This assertion is about eventual recovery while
	// the terminal remains alive, not a sub-second latency budget.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := store.Get(task.ID)
		if got.Status == "succeeded" {
			asset, _ := assetstore.New("live-terminal")
			body, _ := asset.Get("review")
			if body.Content != "终端仍打开时已经完成的正文" {
				t.Fatalf("committed body mismatch: %q", body.Content)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := store.Get(task.ID)
	t.Fatalf("stable result was left behind while terminal stayed alive: status=%s phase=%s", got.Status, got.Phase)
}

type recordingTaskEmitter struct{ names []string }

func (e *recordingTaskEmitter) Emit(name string, _ any) { e.names = append(e.names, name) }

func TestDispatcherEmitsResultProjectionEvents(t *testing.T) {
	emitter := &recordingTaskEmitter{}
	dispatcher := newTestDispatcher(nil, emitter)
	dispatcher.emitAsset("topic", map[string]any{"key": "review"})
	if len(emitter.names) != 1 || emitter.names[0] != "learning-asset-updated" {
		t.Fatalf("unexpected projection events: %#v", emitter.names)
	}
}

func TestDispatcherReportsOnlyPublishedSuccessfulWorkAsLearningActivity(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("activity", "学习活动", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, err := New("activity")
	if err != nil {
		t.Fatal(err)
	}
	task, _, err := store.Create(CreateInput{Type: "consolidate", Objective: "整理闭包笔记", Origin: Origin{OperationID: "op_activity"}})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := newTestDispatcher(nil, nil)
	var published []Task
	dispatcher.deps.OnPublished = func(task Task) { published = append(published, task) }

	dispatcher.finish(store, task.ID, "succeeded", &Result{AssetUpdates: map[string]string{"review": "updated"}}, nil)
	if len(published) != 1 || published[0].ID != task.ID {
		t.Fatalf("published task was not reported: %#v", published)
	}

	second, _, err := store.Create(CreateInput{Type: "consolidate", Objective: "核查闭包笔记", Origin: Origin{OperationID: "op_no_output"}})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher.finish(store, second.ID, "succeeded", &Result{}, nil)
	if len(published) != 1 {
		t.Fatalf("task without published output was reported: %#v", published)
	}
}

func TestLateAssistantResultPublishesWithoutTaskSpecificOutputChecks(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("late", "迟到结果", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, _ := New("late")
	task, _, err := store.Create(CreateInput{Type: "consolidate", Objective: "生成材料", Origin: Origin{OperationID: "op_late"}})
	if err != nil {
		t.Fatal(err)
	}
	runID := "run_late"
	task, _ = store.Update(task.ID, func(current *Task) error {
		current.Status, current.AttemptIDs = "running", []string{runID}
		return nil
	})
	projectRoot, _ := workspace.ProjectRootForSlug("late")
	workDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "workspace")
	for _, dir := range []string{"inputs", "asset-updates/review"} {
		if err := os.MkdirAll(filepath.Join(workDir, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	d := newTestDispatcher(nil, nil)
	manifest, _, err := d.sealInputs(task, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "asset-updates", "review", "current.md"), []byte("助教迟到的正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := resultManifest{SchemaVersion: 1, TaskID: task.ID, RunID: runID, Summary: "迟到但完整的成果", AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"review": {Status: "updated", Candidate: "asset-updates/review/current.md"}}}
	resultPath := filepath.Join(workDir, "result-manifest.json")
	if err := writeJSON(resultPath, result); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(resultPath, old, old); err != nil {
		t.Fatal(err)
	}
	_, _ = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase = "failed", ""
		current.Failure = &Failure{Code: "executor-state-lost", Message: "tracker ended"}
		return nil
	})
	d.reconcileLateResults()
	got, _ := store.Get(task.ID)
	if got.Status != "succeeded" || got.Result == nil || got.Result.AssetUpdates["review"] != "updated" {
		t.Fatalf("late result was not recovered: %#v", got)
	}
	asset, _ := assetstore.New("late")
	body, _ := asset.Get("review")
	if body.Content != "助教迟到的正文" {
		t.Fatalf("late asset was not committed: %q", body.Content)
	}
}

func TestSealInputsUsesCompleteConversationAtApprovalCutoff(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("cutoff", "截止", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	conversation, _ := teacher.NewConversation("cutoff")
	var cutoff uint64
	for i := 0; i < 510; i++ {
		_, event, err := conversation.AppendMessage("learner", "completed", "", []teacher.Block{{Type: "markdown", Source: "message"}})
		if err != nil {
			t.Fatal(err)
		}
		if i == 504 {
			cutoff = event.Seq
		}
	}
	workDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dispatcher := newTestDispatcher(nil, nil)
	manifest, _, err := dispatcher.sealInputs(Task{ID: "task_cutoff", ProjectSlug: "cutoff", ConversationCutoffSeq: cutoff}, workDir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(workDir, "inputs", "conversation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var projection teacher.Projection
	if err := json.Unmarshal(data, &projection); err != nil {
		t.Fatal(err)
	}
	if manifest.Conversation.ThroughSeq != cutoff || len(projection.Messages) != 505 {
		t.Fatalf("sealed snapshot mismatch: through=%d messages=%d", manifest.Conversation.ThroughSeq, len(projection.Messages))
	}
}

func TestFreshLaunchRecordPreventsImmediateRestartFailure(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("launch", "启动", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	store, _ := New("launch")
	task, _, _ := store.Create(CreateInput{Type: "consolidate", Objective: "验证", Origin: Origin{OperationID: "op_launch"}})
	runID := "run_launching"
	if err := prepareLaunchRecord(task, runID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase, current.AttemptIDs = "running", "preparing", []string{runID}
		return nil
	})
	dispatcher := newTestDispatcher(nil, nil)
	dispatcher.reconcileRunningTask(store, task)
	got, _ := store.Get(task.ID)
	if got.Status != "running" {
		t.Fatalf("fresh launch was failed during executor startup window: %#v", got)
	}
	close(dispatcher.stop)
}

func TestRecoverAttemptFinishesCompletedCommitJournal(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, err := New("topic")
	if err != nil {
		t.Fatal(err)
	}
	task, _, err := store.Create(CreateInput{Type: "consolidate", Objective: "验证", Origin: Origin{OperationID: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	runID := "run_recovered"
	task, err = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase = "running", "publishing"
		current.AttemptIDs = []string{runID}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	projectRoot, _ := workspace.ProjectRootForSlug("topic")
	journalPath := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "commit.json")
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(journalPath, commitJournal{SchemaVersion: 1, TaskID: task.ID, RunID: runID, State: "completed", Status: "succeeded", Result: &Result{Summary: "已恢复"}, UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	d := newTestDispatcher(nil, nil)
	d.recoverAttempt(store, task, runID, 0)
	got, _ := store.Get(task.ID)
	if got.Status != "succeeded" || got.Result == nil || got.Result.Summary != "已恢复" {
		t.Fatalf("completed journal not recovered: %#v", got)
	}
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if errors.Is(err, os.ErrExist) {
			return err
		}
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	})
}
