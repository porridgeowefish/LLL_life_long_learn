package assetstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestMigratesLegacyAndProtectsLearnerEdit(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "topic")
	if err := os.WriteFile(filepath.Join(project, "explain", "output.md"), []byte("# 原正文\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New("topic")
	if err != nil {
		t.Fatal(err)
	}
	body, err := s.Get("body")
	if err != nil || body.Content != "# 原正文\n" {
		t.Fatalf("migration failed: %#v %v", body, err)
	}
	edited, err := s.UpdateLearner("body", body.Meta.EditRevision, "# 用户正文\n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateLearner("body", body.Meta.EditRevision, "stale")
	if _, ok := err.(*ConflictError); !ok {
		t.Fatalf("expected conflict, got %v", err)
	}
	if edited.Meta.EditRevision != body.Meta.EditRevision+1 {
		t.Fatal("revision did not advance")
	}
}

func TestMigratedStructuredAssetsKeepTheirPresentationContracts(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("structured", "结构化课程", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "structured")
	if err := os.WriteFile(filepath.Join(project, "explain", "output.md"), []byte("请阅读 explain/manifest.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "explain", "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "explain", "pages", "one.md"), []byte("# 第一课"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": 1, "pages": []map[string]any{{"id": "p1", "file": "pages/one.md"}}})
	if err := os.WriteFile(filepath.Join(project, "explain", "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	tasks, _ := json.Marshal(map[string]any{"schemaVersion": 1, "tasks": []map[string]any{{"id": "q1", "question": "为什么？", "type": "short-answer"}}})
	if err := os.WriteFile(filepath.Join(project, "practice", "tasks.json"), tasks, 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := New("structured")
	if err != nil {
		t.Fatal(err)
	}
	body, err := store.Get("body")
	if err != nil || body.ContentKind != "explain-pages" {
		t.Fatalf("body should retain explain page presentation, got %#v %v", body, err)
	}
	practice, err := store.Get("practice")
	if err != nil || practice.ContentKind != "practice-set" {
		t.Fatalf("practice should retain interactive practice presentation, got %#v %v", practice, err)
	}

	edited, err := store.UpdateLearner("body", body.Meta.EditRevision, "# 学习者重写的正文")
	if err != nil {
		t.Fatal(err)
	}
	if edited.ContentKind != "markdown" {
		t.Fatalf("a canonical learner edit must replace the migration presentation, got %q", edited.ContentKind)
	}
}

func TestCandidateOverlapPreservesLearner(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	s, _ := New("topic")
	base, _ := s.Get("body")
	_, _ = s.UpdateLearner("body", base.Meta.EditRevision, "learner rewrite")
	result, err := s.CommitCandidate(CandidateInput{Key: "body", BaseVersionID: base.Meta.CurrentVersionID, BaseContent: base.Content, CandidateContent: "assistant rewrite", TaskID: "task", ThroughSeq: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != "edit-conflict" {
		t.Fatalf("expected edit conflict, got %#v", result)
	}
	current, _ := s.Get("body")
	if current.Content != "learner rewrite" {
		t.Fatalf("learner content overwritten: %q", current.Content)
	}
}

func TestCandidateRecoveryRepairsCurrentAndMetaFromExistingVersion(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("recovery", "恢复", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	store, _ := New("recovery")
	base, _ := store.Get("body")
	input := CandidateInput{Key: "body", BaseVersionID: base.Meta.CurrentVersionID, BaseContent: base.Content, CandidateContent: "assistant candidate", TaskID: "task_recovery", RunID: "run_recovery", ThroughSeq: 17}
	first, err := store.CommitCandidate(input)
	if err != nil || first.Asset == nil {
		t.Fatalf("first commit failed: %#v %v", first, err)
	}
	versionID := first.Asset.Meta.CurrentVersionID
	assetDir := filepath.Join(root, "recovery", "assets", "body")
	if err := os.WriteFile(filepath.Join(assetDir, "current.md"), []byte(base.Content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(assetDir, "asset.json"), base.Meta); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.CommitCandidate(input)
	if err != nil || recovered.Asset == nil {
		t.Fatalf("recovery failed: %#v %v", recovered, err)
	}
	if recovered.Asset.Content != "assistant candidate" || recovered.Asset.Meta.CurrentVersionID != versionID || recovered.Asset.Meta.ConversationCursor != 17 {
		t.Fatalf("recovery returned inconsistent asset: %#v", recovered.Asset)
	}
	current, _ := store.Get("body")
	if current.Content != "assistant candidate" || current.Meta.CurrentVersionID != versionID {
		t.Fatalf("canonical asset was not repaired: %#v", current)
	}
}
