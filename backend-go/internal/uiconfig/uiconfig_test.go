package uiconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSavePreservesOtherConfigAndLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	if err := os.WriteFile(path, []byte(`{"agentRuntime":"codex"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := UseConfigPathForTest(path)
	defer restore()
	if err := Save(Config{Theme: ThemeMountainMist}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"agentRuntime": "codex"`) {
		t.Fatalf("other config was lost: %s", data)
	}
	got, err := Load()
	if err != nil || got.Theme != ThemeMountainMist {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
}

func TestSaveRejectsUnknownTheme(t *testing.T) {
	if err := Save(Config{Theme: "neon"}); err == nil {
		t.Fatal("expected unknown theme error")
	}
}
