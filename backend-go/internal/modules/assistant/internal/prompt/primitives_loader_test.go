package promptassembly

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
)

func writePrimFile(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, "primitives", name+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPrimitive_BasicAndCached(t *testing.T) {
	dir := t.TempDir()
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	ClearPrimitiveCacheForTest()

	writePrimFile(t, dir, "alpha", "alpha body v1")

	got, err := LoadPrimitive("alpha")
	if err != nil {
		t.Fatalf("LoadPrimitive: %v", err)
	}
	if got != "alpha body v1" {
		t.Errorf("got %q, want %q", got, "alpha body v1")
	}

	// Mutate the file; cached value should still be returned.
	writePrimFile(t, dir, "alpha", "alpha body v2")
	got2, _ := LoadPrimitive("alpha")
	if got2 != "alpha body v1" {
		t.Errorf("cache miss: got %q, want cached %q", got2, "alpha body v1")
	}
}

func TestLoadPrimitive_MissingReturnsError(t *testing.T) {
	dir := t.TempDir()
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	ClearPrimitiveCacheForTest()

	_, err := LoadPrimitive("definitely_missing")
	if err == nil {
		t.Fatal("expected error for missing primitive, got nil")
	}
}

func TestExpandPrimitives_RequiredAndOptional(t *testing.T) {
	dir := t.TempDir()
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	ClearPrimitiveCacheForTest()

	writePrimFile(t, dir, "req_a", "REQ_A body")
	writePrimFile(t, dir, "req_b", "REQ_B body")
	writePrimFile(t, dir, "opt_a", "OPT_A body")

	out, err := ExpandPrimitives([]string{"req_a", "req_b"}, []string{"opt_a"})
	if err != nil {
		t.Fatalf("ExpandPrimitives: %v", err)
	}
	if !strings.Contains(out, "REQ_A body") {
		t.Errorf("missing REQ_A body in output:\n%s", out)
	}
	if !strings.Contains(out, "REQ_B body") {
		t.Errorf("missing REQ_B body in output:\n%s", out)
	}
	if !strings.Contains(out, "OPT_A body") {
		t.Errorf("missing OPT_A body in output:\n%s", out)
	}
	if !strings.Contains(out, "### Required primitives") {
		t.Errorf("missing required section header in output:\n%s", out)
	}
	if !strings.Contains(out, "### Optional primitive references") {
		t.Errorf("missing optional section header in output:\n%s", out)
	}
	if !strings.Contains(out, "Do not create a page or section merely because a mechanism is listed here.") {
		t.Errorf("missing optional activation guard in output:\n%s", out)
	}
}

func TestExpandPrimitives_MissingRequiredFails(t *testing.T) {
	dir := t.TempDir()
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	ClearPrimitiveCacheForTest()

	writePrimFile(t, dir, "present", "body")

	_, err := ExpandPrimitives([]string{"present", "absent"}, nil)
	if err == nil {
		t.Fatal("expected error when a required primitive is missing")
	}
}

func TestExpandPrimitives_MissingOptionalWarnsButSucceeds(t *testing.T) {
	dir := t.TempDir()
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	ClearPrimitiveCacheForTest()

	writePrimFile(t, dir, "present", "body")

	out, err := ExpandPrimitives([]string{"present"}, []string{"absent"})
	if err != nil {
		t.Fatalf("missing optional must not error, got: %v", err)
	}
	if !strings.Contains(out, "body") {
		t.Errorf("present body missing from output:\n%s", out)
	}
}
