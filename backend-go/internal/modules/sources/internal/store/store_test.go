package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	s, err := New("topic")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAddAndDeleteSource(t *testing.T) {
	s := newTestStore(t)
	data := []byte("教材")
	source, revision, err := s.Add("教材", "../book.md", "text/markdown", int64(len(data)), bytes.NewReader(data), true)
	if err != nil {
		t.Fatal(err)
	}
	if revision.Original.Filename != "book.md" || source.Status != "stored" {
		t.Fatalf("unexpected source: %#v %#v", source, revision)
	}
	path := filepath.Join(s.root, source.SourceID, "revisions", revision.RevisionID, filepath.FromSlash(revision.Original.Path))
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, data) {
		t.Fatalf("original not retained: %q %v", got, err)
	}
	if _, err := s.PermanentlyDelete(source.SourceID, true); err == nil {
		t.Fatal("active reference should block deletion")
	}
	deleted, err := s.PermanentlyDelete(source.SourceID, false)
	if err != nil || deleted.Status != "deleted" {
		t.Fatalf("delete failed: %#v %v", deleted, err)
	}
	if deleted.OriginalSHA256 != revision.Original.SHA256 || deleted.OriginalBytes != int64(len(data)) || deleted.OriginalMediaType != "text/markdown" {
		t.Fatalf("permanent deletion lost non-content evidence: %#v", deleted)
	}
}

func TestParseDispositionKeepsArchivesAndExecutablesOpaque(t *testing.T) {
	for _, name := range []string{"book.zip", "installer.exe", "script.ps1", "unknown.blob"} {
		if disposition, _ := ParseDisposition(name, "application/octet-stream"); disposition != "opaque" {
			t.Fatalf("unsafe or unknown file became parseable: %s", name)
		}
	}
	for _, name := range []string{"book.pdf", "notes.md", "table.xlsx", "source.py", "figure.png"} {
		if disposition, _ := ParseDisposition(name, ""); disposition != "parse" {
			t.Fatalf("supported file stayed opaque: %s", name)
		}
	}
}

func TestRejectsDeclaredSizeMismatch(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.Add("bad", "bad.txt", "text/plain", 2, bytes.NewReader([]byte("three")), false)
	if err == nil {
		t.Fatal("expected byte mismatch")
	}
}

func TestCommitDerivedAcceptsOnlyCanonicalMarkdownContent(t *testing.T) {
	s := newTestStore(t)
	data := []byte("source")
	source, revision, err := s.Add("教材", "book.pdf", "application/pdf", int64(len(data)), bytes.NewReader(data), true)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.CommitDerived(source.SourceID, revision.RevisionID, map[string][]byte{"notes.md": []byte("extra")}, map[string]string{"notes.md": "text/markdown"}); err == nil {
		t.Fatal("non-canonical derived content must be rejected")
	}
	updated, err := s.CommitDerived(source.SourceID, revision.RevisionID, map[string][]byte{"content.md": []byte("# 提取内容")}, map[string]string{"content.md": "text/markdown; charset=utf-8"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.DerivedFiles) != 1 || updated.DerivedFiles[0].Key != "content" || updated.DerivedFiles[0].Path != "derived/content.md" {
		t.Fatalf("expected exactly canonical content.md, got %#v", updated.DerivedFiles)
	}
	if _, err := s.SetStatus(source.SourceID, "ready", "", "task_parse"); err != nil {
		t.Fatal(err)
	}
	content, err := s.ReadContent(source.SourceID)
	if err != nil || content != "# 提取内容" {
		t.Fatalf("ready source content was not readable: %q %v", content, err)
	}
}
