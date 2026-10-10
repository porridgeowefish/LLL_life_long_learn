package annotationstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/modules/assets/internal/assetstore"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func TestReviewAnnotationMigrationKeepsOldQuote(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	oldDir := filepath.Join(root, "topic", "assets", "body")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{"schemaVersion":1,"seq":1,"eventId":"aevt_old","type":"annotation-created","annotationId":"ann_old","at":"2026-09-01T00:00:00Z","annotation":{"schemaVersion":1,"annotationId":"ann_old","assetId":"asset_old","assetVersionId":"aver_old","quoteSnapshot":"原有引用","status":"open","createdAt":"2026-09-01T00:00:00Z","updatedAt":"2026-09-01T00:00:00Z"}}` + "\n"
	if err := os.WriteFile(filepath.Join(oldDir, "annotations.jsonl"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "current.md"), []byte("原正文"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := assetstore.MigrateReview("topic"); err != nil {
		t.Fatal(err)
	}
	store, err := New("topic")
	if err != nil {
		t.Fatal(err)
	}
	annotation, err := store.Get("ann_old")
	if err != nil {
		t.Fatal(err)
	}
	var resource map[string]any
	encoded, _ := json.Marshal(annotation)
	_ = json.Unmarshal(encoded, &resource)
	if annotation.QuoteSnapshot != "原有引用" || annotation.AssetVersionID != "aver_old" || !annotation.Detached || resource["legacySource"] != true {
		t.Fatalf("old citation lost: %#v", resource)
	}
}

func TestAppendOnlyAnnotationThreadRecovers(t *testing.T) {
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
	created, err := store.Create(CreateInput{QuoteSnapshot: "闭包保存词法环境", Anchors: Anchors{Start: 4, End: 12, Prefix: "前", Suffix: "后"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendMessage(created.AnnotationID, Message{Role: "learner", Content: "为什么？"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendMessage(created.AnnotationID, Message{Role: "assistant", Content: "因为函数捕获了环境。"}); err != nil {
		t.Fatal(err)
	}
	reopened, _ := New("topic")
	got, err := reopened.Get(created.AnnotationID)
	if err != nil || got.AssetVersionID == "" || len(got.Ask.Messages) != 2 || got.Ask.Messages[0].Role != "learner" {
		t.Fatalf("unexpected projection: %#v %v", got, err)
	}
	path := filepath.Join(root, "topic", "assets", "review", "annotations.jsonl")
	if data, _ := os.ReadFile(path); len(data) == 0 {
		t.Fatal("canonical JSONL was not written")
	}
}

func TestDeleteIsSemanticEvent(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	store, _ := New("topic")
	created, _ := store.Create(CreateInput{QuoteSnapshot: "文本"})
	if err := store.Delete(created.AnnotationID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(created.AnnotationID); !os.IsNotExist(err) {
		t.Fatalf("deleted annotation still readable: %v", err)
	}
	all, _ := store.List(true)
	if len(all) != 1 || all[0].Status != StateDeleted {
		t.Fatalf("deletion history lost: %#v", all)
	}
}
