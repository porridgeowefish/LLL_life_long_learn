package agentregistry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

// withTempAgents redirects paths.AGENTS_ROOT to a temp dir for the test.
func withTempAgents(t *testing.T, fill func(dir string)) func() {
	t.Helper()
	dir := t.TempDir()
	old := agentsRootOverride
	agentsRootOverride = dir
	fill(dir)
	return func() { agentsRootOverride = old }
}

// writePrimitive creates agents/primitives/<name>.md inside dir.
func writePrimitive(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, "primitives", name+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeAgent(t *testing.T, dir, id string, zones []workspace.ZoneName, charter string) {
	t.Helper()
	regDir := filepath.Join(dir, "registry")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chPath := filepath.Join(dir, "charters", id+".md")
	if err := os.MkdirAll(filepath.Dir(chPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chPath, []byte(charter), 0o644); err != nil {
		t.Fatal(err)
	}
	a := Agent{
		ID:                   id,
		Name:                 id,
		UserStory:            "test user story for " + id,
		AllowedZones:         zones,
		CharterPath:          chPath,
		DefaultOutputTargets: []OutputTarget{{ZoneName: zones[0], Filename: "output.md"}},
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, id+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRegistry_LoadValidAgent(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		writeAgent(t, dir, "explain", []workspace.ZoneName{workspace.ZoneExplain}, "# Explain charter\n")
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := r.Get("explain")
	if !ok {
		t.Fatal("explain agent missing")
	}
	if got.CharterText == "" {
		t.Error("charter text not loaded")
	}
	if got.UserStory == "" {
		t.Error("userStory not retained")
	}
}

func TestRegistry_RejectsUnknownZone(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		writeAgent(t, dir, "bad", []workspace.ZoneName{"Bogus"}, "")
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err == nil {
		t.Error("expected error for unknown zone, got nil")
	}
}

func TestRegistry_ListSorted(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		writeAgent(t, dir, "zeta", []workspace.ZoneName{workspace.ZoneIntro}, "")
		writeAgent(t, dir, "alpha", []workspace.ZoneName{workspace.ZoneIntro}, "")
		writeAgent(t, dir, "middle", []workspace.ZoneName{workspace.ZoneExplain}, "")
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err != nil {
		t.Fatal(err)
	}
	got := r.List()
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0].ID != "alpha" || got[2].ID != "zeta" {
		t.Errorf("not sorted: %+v", got)
	}
}

func TestRegistry_RejectsEmptyUserStory(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		// Hand-write a registry entry with no userStory.
		regDir := filepath.Join(dir, "registry")
		_ = os.MkdirAll(regDir, 0o755)
		a := Agent{
			ID:           "nouserstory",
			Name:         "nouserstory",
			AllowedZones: []workspace.ZoneName{workspace.ZoneExplain},
			CharterPath:  filepath.Join(dir, "charters", "nouserstory.md"),
		}
		_ = os.MkdirAll(filepath.Dir(a.CharterPath), 0o755)
		_ = os.WriteFile(a.CharterPath, []byte("x"), 0o644)
		data, _ := json.Marshal(a)
		_ = os.WriteFile(filepath.Join(regDir, "nouserstory.json"), data, 0o644)
	})
	defer cleanup()

	r := New()
	err := r.Load()
	if err == nil {
		t.Fatal("expected error for empty userStory, got nil")
	}
}

func TestRegistry_RejectsTooManyRequiredPrimitives(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		regDir := filepath.Join(dir, "registry")
		_ = os.MkdirAll(regDir, 0o755)
		chPath := filepath.Join(dir, "charters", "toomany.md")
		_ = os.MkdirAll(filepath.Dir(chPath), 0o755)
		_ = os.WriteFile(chPath, []byte("x"), 0o644)
		a := Agent{
			ID:           "toomany",
			Name:         "toomany",
			UserStory:    "has too many primitives",
			AllowedZones: []workspace.ZoneName{workspace.ZoneExplain},
			CharterPath:  chPath,
			Primitives: Primitives{
				Required: []string{"a", "b", "c", "d", "e", "f", "g"}, // 7 > 6
			},
		}
		data, _ := json.Marshal(a)
		_ = os.WriteFile(filepath.Join(regDir, "toomany.json"), data, 0o644)
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err == nil {
		t.Fatal("expected error for too many required primitives, got nil")
	}
}

func TestRegistry_RejectsInvalidPrimitiveName(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		regDir := filepath.Join(dir, "registry")
		_ = os.MkdirAll(regDir, 0o755)
		chPath := filepath.Join(dir, "charters", "badname.md")
		_ = os.MkdirAll(filepath.Dir(chPath), 0o755)
		_ = os.WriteFile(chPath, []byte("x"), 0o644)
		a := Agent{
			ID:           "badname",
			Name:         "badname",
			UserStory:    "has an invalid primitive name",
			AllowedZones: []workspace.ZoneName{workspace.ZoneExplain},
			CharterPath:  chPath,
			Primitives: Primitives{
				Required: []string{"UPPERCASE"}, // invalid
			},
		}
		data, _ := json.Marshal(a)
		_ = os.WriteFile(filepath.Join(regDir, "badname.json"), data, 0o644)
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err == nil {
		t.Fatal("expected error for invalid primitive name, got nil")
	}
}

func TestRegistry_AcceptsValidPrimitives(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		writePrimitive(t, dir, "mece_decompose", "body for mece")
		writePrimitive(t, dir, "first_principles", "body for fp")
		regDir := filepath.Join(dir, "registry")
		_ = os.MkdirAll(regDir, 0o755)
		chPath := filepath.Join(dir, "charters", "good.md")
		_ = os.MkdirAll(filepath.Dir(chPath), 0o755)
		_ = os.WriteFile(chPath, []byte("x"), 0o644)
		a := Agent{
			ID:           "good",
			Name:         "good",
			UserStory:    "well-formed agent",
			AllowedZones: []workspace.ZoneName{workspace.ZoneExplain},
			CharterPath:  chPath,
			Primitives: Primitives{
				Required: []string{"mece_decompose", "first_principles"},
				Optional: []string{"analogy"},
			},
		}
		data, _ := json.Marshal(a)
		_ = os.WriteFile(filepath.Join(regDir, "good.json"), data, 0o644)
	})
	defer cleanup()

	r := New()
	if err := r.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := r.Get("good")
	if !ok {
		t.Fatal("agent missing")
	}
	if len(got.Primitives.Required) != 2 {
		t.Errorf("required primitives len=%d, want 2", len(got.Primitives.Required))
	}
}

func TestRegistry_PrimitiveExists(t *testing.T) {
	cleanup := withTempAgents(t, func(dir string) {
		writePrimitive(t, dir, "exists", "body")
	})
	defer cleanup()

	if !PrimitiveExists("exists") {
		t.Error("PrimitiveExists(exists) = false, want true")
	}
	if PrimitiveExists("nope") {
		t.Error("PrimitiveExists(nope) = true, want false")
	}
	if PrimitiveExists("UPPERCASE") {
		t.Error("PrimitiveExists(UPPERCASE) = true, want false (invalid name)")
	}
}
