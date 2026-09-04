package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/paths"
)

func TestReadCreatesAndWriteUpdatesSingleGlobalFile(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })

	first, err := Ensure()
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != filepath.Join(paths.WORKSPACE, Filename) || !strings.Contains(first.Content, "全局学习偏好") {
		t.Fatalf("unexpected initial snapshot: %#v", first)
	}
	if _, err := Write("# 我的偏好\n\n先给结论。\n"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(first.Path)
	if err != nil || string(data) != "# 我的偏好\n\n先给结论。\n" {
		t.Fatalf("unexpected file: %q, %v", data, err)
	}
}

func TestWriteRejectsOversizedContent(t *testing.T) {
	if _, err := Write(strings.Repeat("x", MaxBytes+1)); err == nil {
		t.Fatal("expected size error")
	}
}
