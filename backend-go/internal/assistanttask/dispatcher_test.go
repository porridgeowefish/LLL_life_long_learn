package assistanttask

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/assetstore"
	"github.com/xmz14/lll/backend-go/internal/conversationstore"
	"github.com/xmz14/lll/backend-go/internal/sourcestore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

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
	if err := os.MkdirAll(filepath.Join(workDir, "asset-updates", "body"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(nil, nil)
	defer close(d.stop)
	manifest, _, err := d.sealInputs(task, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "asset-updates", "body", "current.md"), []byte("终端仍打开时已经完成的正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := resultManifest{SchemaVersion: 1, TaskID: task.ID, RunID: runID, Summary: "成果已经写完", AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"intro": {Status: "unchanged"}, "body": {Status: "updated", Candidate: "asset-updates/body/current.md"}, "practice": {Status: "unchanged"}}}
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
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		got, _ := store.Get(task.ID)
		if got.Status == "succeeded" {
			asset, _ := assetstore.New("live-terminal")
			body, _ := asset.Get("body")
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
	dispatcher := NewDispatcher(nil, emitter)
	dispatcher.emitGenerated("topic", "task-1", "artifact-1")
	dispatcher.emitSource("topic", sourcestore.Source{SourceID: "source-1"})
	if len(emitter.names) != 2 || emitter.names[0] != "generated-artifact-updated" || emitter.names[1] != "source-updated" {
		t.Fatalf("unexpected projection events: %#v", emitter.names)
	}
}

func TestValidateResultRequiresDeclaredHashedOutputs(t *testing.T) {
	workDir := t.TempDir()
	fileDir := filepath.Join(workDir, "deliverables", "report", "files")
	if err := os.MkdirAll(fileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("# report\n")
	if err := os.WriteFile(filepath.Join(fileDir, "report.md"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	descriptor := map[string]any{"schemaVersion": 1, "kind": "report", "title": "报告", "entryPoints": []string{"files/report.md"}, "files": []map[string]any{{"path": "files/report.md", "mediaType": "text/markdown", "sha256": hashBytes(data), "bytes": len(data)}}}
	if err := writeJSON(filepath.Join(workDir, "deliverables", "report", "artifact.json"), descriptor); err != nil {
		t.Fatal(err)
	}
	result := resultManifest{SchemaVersion: 1, TaskID: "task_x", RunID: "run_x", Summary: "完成", AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"intro": {Status: "unchanged"}, "body": {Status: "unchanged"}, "practice": {Status: "unchanged"}}}
	result.Deliverables = append(result.Deliverables, struct {
		Key        string `json:"key"`
		Descriptor string `json:"descriptor"`
	}{Key: "report", Descriptor: "deliverables/report/artifact.json"})
	if err := validateResult(workDir, Task{Type: "produce-material"}, inputManifest{}, result); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fileDir, "undeclared.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateResult(workDir, Task{Type: "produce-material"}, inputManifest{}, result); err == nil {
		t.Fatal("undeclared deliverable file was accepted")
	}
}

func TestArtifactDescriptorAcceptsWorkspaceRelativeDeclaredFiles(t *testing.T) {
	workDir := t.TempDir()
	filePath := filepath.Join(workDir, "deliverables", "report", "files", "report.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("# report\n")
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	descriptorPath := filepath.Join(workDir, "deliverables", "report", "artifact.json")
	descriptor := map[string]any{
		"schemaVersion": 1, "kind": "report", "title": "报告",
		"entryPoints": []string{"deliverables/report/files/report.md"},
		"files":       []map[string]any{{"path": "deliverables/report/files/report.md", "mediaType": "text/markdown", "sha256": hashBytes(data), "bytes": len(data)}},
	}
	if err := writeJSON(descriptorPath, descriptor); err != nil {
		t.Fatal(err)
	}
	deliverable := struct {
		Key        string `json:"key"`
		Descriptor string `json:"descriptor"`
	}{Key: "report", Descriptor: "deliverables/report/artifact.json"}
	parsed, err := validateArtifactDescriptor(workDir, deliverable)
	if err != nil {
		t.Fatalf("workspace-relative artifact path rejected: %v", err)
	}
	staged := filepath.Join(t.TempDir(), "artifact")
	if err := stageArtifactPackage(workDir, filepath.Dir(descriptorPath), staged, parsed); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(staged, "deliverables", "report", "files", "report.md")); err != nil || string(got) != string(data) {
		t.Fatalf("workspace-relative artifact was not packaged: %q %v", got, err)
	}
}

func TestLateVisibleTerminalResultIsCommittedAfterPrematureFailure(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("late", "迟到结果", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, _ := New("late")
	task, _, err := store.Create(CreateInput{Type: "produce-material", Objective: "生成材料", Origin: Origin{OperationID: "op_late"}})
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
	for _, dir := range []string{"inputs", "asset-updates/body", "deliverables"} {
		if err := os.MkdirAll(filepath.Join(workDir, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	d := NewDispatcher(nil, nil)
	manifest, _, err := d.sealInputs(task, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "asset-updates", "body", "current.md"), []byte("助教迟到的正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := resultManifest{SchemaVersion: 1, TaskID: task.ID, RunID: runID, Summary: "迟到但完整的成果", AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"intro": {Status: "unchanged"}, "body": {Status: "updated", Candidate: "asset-updates/body/current.md"}, "practice": {Status: "unchanged"}}}
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
	if got.Status != "succeeded" || got.Result == nil || got.Result.AssetUpdates["body"] != "updated" {
		t.Fatalf("late result was not recovered: %#v", got)
	}
	asset, _ := assetstore.New("late")
	body, _ := asset.Get("body")
	if body.Content != "助教迟到的正文" {
		t.Fatalf("late asset was not committed: %q", body.Content)
	}
}

func TestUnsafeSVGRejectsGeneralActiveAndExternalContent(t *testing.T) {
	unsafe := []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><circle onfocus="alert(1)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="//tracker.example/pixel.png"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url(https://tracker.example/a.svg#x)"/></svg>`,
		`<!DOCTYPE svg><svg xmlns="http://www.w3.org/2000/svg"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://tracker.example/x.css)</style></svg>`,
	}
	for _, source := range unsafe {
		if !unsafeSVG([]byte(source)) {
			t.Fatalf("unsafe SVG accepted: %s", source)
		}
	}
	safe := `<svg xmlns="http://www.w3.org/2000/svg"><defs><linearGradient id="g"/></defs><rect style="fill:url(#g)"/></svg>`
	if unsafeSVG([]byte(safe)) {
		t.Fatal("internal paint reference was rejected")
	}
	adaptive := `<svg xmlns="http://www.w3.org/2000/svg"><style>@media (prefers-color-scheme: dark) { .label { fill: #fff; } }</style><text class="label">安全图表</text></svg>`
	if unsafeSVG([]byte(adaptive)) {
		t.Fatal("safe adaptive SVG style was rejected")
	}
}

func TestSealInputsUsesCompleteConversationAtApprovalCutoff(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("cutoff", "截止", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	conversation, _ := conversationstore.New("cutoff")
	var cutoff uint64
	for i := 0; i < 510; i++ {
		_, event, err := conversation.AppendMessage("learner", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "message"}})
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
	dispatcher := NewDispatcher(nil, nil)
	manifest, _, err := dispatcher.sealInputs(Task{ID: "task_cutoff", ProjectSlug: "cutoff", ConversationCutoffSeq: cutoff}, workDir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(workDir, "inputs", "conversation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var projection conversationstore.Projection
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
	task, _, _ := store.Create(CreateInput{Type: "verify", Objective: "验证", Origin: Origin{OperationID: "op_launch"}})
	runID := "run_launching"
	if err := prepareLaunchRecord(task, runID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase, current.AttemptIDs = "running", "preparing", []string{runID}
		return nil
	})
	dispatcher := NewDispatcher(nil, nil)
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
	task, _, err := store.Create(CreateInput{Type: "verify", Objective: "验证", Origin: Origin{OperationID: "op"}})
	if err != nil {
		t.Fatal(err)
	}
	runID := "run_recovered"
	task, err = store.Update(task.ID, func(current *Task) error {
		current.Status, current.Phase = "running", "committing"
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
	d := NewDispatcher(nil, nil)
	d.recoverAttempt(store, task, runID, 0)
	got, _ := store.Get(task.ID)
	if got.Status != "succeeded" || got.Result == nil || got.Result.Summary != "已恢复" {
		t.Fatalf("completed journal not recovered: %#v", got)
	}
}
