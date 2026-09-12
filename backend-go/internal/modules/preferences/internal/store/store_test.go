package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

func TestReadCreatesAndWriteUpdatesSingleGlobalFile(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

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

func TestEnsureRestoresPreferencesWhenWorkspaceFileDisappears(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := "# 我的偏好\n\n保留真实案例。\n"
	if _, err := Write(want); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(Path()); err != nil {
		t.Fatal(err)
	}

	restored, err := Ensure()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Content != want {
		t.Fatalf("preferences were not restored: %q", restored.Content)
	}
	data, err := os.ReadFile(Path())
	if err != nil || string(data) != want {
		t.Fatalf("canonical preferences were not recovered: %q, %v", data, err)
	}
}

func TestEnsureSeedsRecoveryBackupForAnExistingPreferenceFile(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := "# 已有偏好\n\n不要因升级丢失。\n"
	if err := os.WriteFile(Path(), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(Path()); err != nil {
		t.Fatal(err)
	}

	restored, err := Ensure()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Content != want {
		t.Fatalf("existing preferences were not backed up: %q", restored.Content)
	}
}

func TestBackupExistingKeepsPreferencesReadableAfterWorkspaceFileLoss(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := "# 启动时备份\n\n保留案例偏好。\n"
	if err := os.WriteFile(Path(), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := BackupExisting(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(Path()); err != nil {
		t.Fatal(err)
	}

	recovered, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Exists || recovered.Content != want {
		t.Fatalf("startup backup was not readable: %#v", recovered)
	}
}

func TestBackupExistingRestoresNonEmptyBackupAfterUnexpectedTruncation(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := "# 已保存偏好\n\n不要被空文件覆盖。\n"
	if _, err := Write(want); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := BackupExisting(); err != nil {
		t.Fatal(err)
	}
	recovered, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Content != want {
		t.Fatalf("non-empty backup was overwritten: %q", recovered.Content)
	}
}

func TestEnsureRestoresNonEmptyBackupAfterUnexpectedTruncation(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := "# 已保存偏好\n\n打开编辑器时恢复。\n"
	if _, err := Write(want); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	recovered, err := Ensure()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Content != want {
		t.Fatalf("empty canonical file won over recovery backup: %q", recovered.Content)
	}
}

func TestWriteRejectsOversizedContent(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Write(strings.Repeat("x", MaxBytes+1)); err == nil {
		t.Fatal("expected size error")
	}
}
