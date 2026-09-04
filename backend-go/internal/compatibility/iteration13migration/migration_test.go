package iteration13migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

func TestMigrationBacksUpAndCreatesCanonicalWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "topic")
	if err := os.WriteFile(filepath.Join(project, "explain", "output.md"), []byte("# 正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy, _ := json.Marshal([]map[string]any{
		{
			"id": "legacy-note", "quoteSnapshot": "旧批注", "charStart": 1, "charEnd": 4,
			"state": "open", "createdAt": "2026-08-01T00:00:00Z",
			"ask": map[string]any{"messages": []map[string]any{
				{"id": "old-msg", "role": "user", "content": "旧问题", "createdAt": "2026-08-01T00:01:00Z"},
			}},
		},
	})
	if err := os.WriteFile(filepath.Join(project, "explain", "confusions.json"), legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Migrate("topic"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"unit.json", "conversation/conversation.json", "assets/body/current.md", "migrations/iteration-13/backup/explain/output.md"} {
		if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(path))); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	content, _ := os.ReadFile(filepath.Join(project, "assets", "body", "current.md"))
	if string(content) != "# 正文" {
		t.Fatalf("body not migrated: %q", content)
	}
	if err := Migrate("topic"); err != nil {
		t.Fatalf("migration should be idempotent: %v", err)
	}
	annotationBytes, err := os.ReadFile(filepath.Join(project, "assets", "body", "annotations.jsonl"))
	if err != nil || !strings.Contains(string(annotationBytes), "legacy-note") || !strings.Contains(string(annotationBytes), "旧问题") {
		t.Fatalf("legacy annotations not migrated: %v %s", err, annotationBytes)
	}
}

func TestMigrationResumesIncompleteJournalEvenWhenUnitExists(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("resume", "恢复迁移", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "resume")
	if err := os.WriteFile(filepath.Join(project, "explain", "output.md"), []byte("# 恢复正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory, err := inventoryActiveLegacy(project)
	if err != nil {
		t.Fatal(err)
	}
	migrationDir := filepath.Join(project, "migrations", "iteration-13")
	backupDir := filepath.Join(migrationDir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range inventory {
		if err := copyRegular(filepath.Join(project, filepath.FromSlash(file.Path)), filepath.Join(backupDir, filepath.FromSlash(file.Path))); err != nil {
			t.Fatal(err)
		}
	}
	record := Record{SchemaVersion: 1, Status: "failed", From: "legacy-five-zone", Inventory: inventory, InventoryHash: inventoryHash(inventory), BackupPath: "migrations/iteration-13/backup"}
	if err := writeJSON(filepath.Join(migrationDir, "migration.json"), record); err != nil {
		t.Fatal(err)
	}
	if _, err := teacher.NewConversation("resume"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate("resume"); err != nil {
		t.Fatal(err)
	}
	var completed Record
	if err := readJSON(filepath.Join(migrationDir, "migration.json"), &completed); err != nil || completed.Status != "completed" {
		t.Fatalf("incomplete migration did not resume: %#v %v", completed, err)
	}
	body, err := os.ReadFile(filepath.Join(project, "assets", "body", "current.md"))
	if err != nil || string(body) != "# 恢复正文" {
		t.Fatalf("resumed body mismatch: %q %v", body, err)
	}
}

func TestMigrationIgnoresRetiredZoneFiles(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("retired", "历史目录", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "retired")
	files := map[string]string{
		"summary/legacy.md": "历史总结",
		"extend/legacy.md":  "历史拓展",
	}
	for path, content := range files {
		absolute := filepath.Join(project, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Migrate("retired"); err != nil {
		t.Fatal(err)
	}
	for path, want := range files {
		got, err := os.ReadFile(filepath.Join(project, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Fatalf("historical file %s changed: %q, %v", path, got, err)
		}
	}
	var record Record
	if err := readJSON(filepath.Join(project, "migrations", "iteration-13", "migration.json"), &record); err != nil {
		t.Fatal(err)
	}
	for _, item := range record.Inventory {
		if strings.HasPrefix(item.Path, "summary/") || strings.HasPrefix(item.Path, "extend/") {
			t.Fatalf("retired path remained in migration inventory: %s", item.Path)
		}
	}
}
