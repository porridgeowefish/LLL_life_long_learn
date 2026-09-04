package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Negative fixtures: the rule engine must reject exactly these shapes.

func TestCheckGoEdgeRules(t *testing.T) {
	cases := []struct {
		name     string
		importer string
		imported string
		wantRule string // "" = allowed
	}{
		// R1 transport → module-private: REJECTED
		{"transport to module private", repoModule + "/backend-go/internal/transport/http", repoModule + "/backend-go/internal/modules/teacher/internal/gateway", "R1-transport-imports-module-private"},
		// transport → module facade: allowed
		{"transport to module facade", repoModule + "/backend-go/internal/transport/http", repoModule + "/backend-go/internal/modules/teacher", ""},
		// transport → platform: allowed
		{"transport to platform", repoModule + "/backend-go/internal/transport/http", repoModule + "/backend-go/internal/platform/httpx", ""},
		// R2 cross-module private: REJECTED even from another module facade
		{"cross module private", repoModule + "/backend-go/internal/modules/assistant", repoModule + "/backend-go/internal/modules/assets/internal/store", "R2-cross-module-private"},
		{"cross module private nested", repoModule + "/backend-go/internal/modules/assistant/internal/dispatch", repoModule + "/backend-go/internal/modules/teacher/internal/service", "R2-cross-module-private"},
		// same module internal: allowed
		{"same module internal", repoModule + "/backend-go/internal/modules/assistant/internal/dispatch", repoModule + "/backend-go/internal/modules/assistant/internal/tasks", ""},
		// module → other module facade: allowed
		{"module to other facade", repoModule + "/backend-go/internal/modules/teacher", repoModule + "/backend-go/internal/modules/assets", ""},
		// R3 platform → business: REJECTED
		{"platform to modules", repoModule + "/backend-go/internal/platform/filesystem", repoModule + "/backend-go/internal/modules/projects", "R3-platform-imports-business"},
		{"platform to transport", repoModule + "/backend-go/internal/platform/config", repoModule + "/backend-go/internal/transport/http", "R3-platform-imports-business"},
		{"platform to app", repoModule + "/backend-go/internal/platform/events", repoModule + "/backend-go/internal/app/bootstrap", "R3-platform-imports-business"},
		// platform → stdlib-ish other platform: allowed
		{"platform to platform", repoModule + "/backend-go/internal/platform/config", repoModule + "/backend-go/internal/platform/filesystem", ""},
		// compatibility (outside modules/) reading a facade: allowed by these rules
		{"compatibility to facade", repoModule + "/backend-go/internal/compatibility", repoModule + "/backend-go/internal/modules/assets", ""},
		// app bootstrap is the composition root: allowed everywhere below modules
		{"bootstrap to facade", repoModule + "/backend-go/internal/app/bootstrap", repoModule + "/backend-go/internal/modules/teacher", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := checkGoEdge(tc.importer, tc.imported)
			if tc.wantRule == "" {
				if v != nil {
					t.Fatalf("expected allowed, got violation %+v", v)
				}
				return
			}
			if v == nil {
				t.Fatalf("expected rule %s violation, got none", tc.wantRule)
			}
			if v.Rule != tc.wantRule {
				t.Fatalf("rule = %s, want %s", v.Rule, tc.wantRule)
			}
		})
	}
}

func TestCheckFrontendRule(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// learning reaching projects internals directly: REJECTED
	write("learning/TeacherView.tsx", "import { X } from '@/features/projects/components/ProjectCard';\n")
	// learning reaching projects index: allowed
	write("learning/AssetsView.tsx", "import { Y } from '@/features/projects';\n")
	// within-feature deep import: allowed
	write("learning/BodyAnnotations.tsx", "import { Z } from './TeacherView';\n")
	// relative hop into another feature: REJECTED
	write("learning/SourcesView.tsx", "import { W } from '../projects/components/ProjectCard';\n")
	// shared reach: allowed (not under features/)
	write("learning/GeneratedMaterials.tsx", "import { V } from '@/shared/MarkdownView';\n")
	// tests are exempt
	write("learning/TeacherView.test.tsx", "import { U } from '@/features/projects/components/ProjectCard';\n")

	violations, err := checkFrontend(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		for _, v := range violations {
			t.Logf("got: %+v", v)
		}
		t.Fatalf("expected 2 violations, got %d", len(violations))
	}
	for _, v := range violations {
		if v.Rule != "R4-frontend-cross-feature-internal" {
			t.Fatalf("rule = %s", v.Rule)
		}
	}

	// Missing features dir: rule vacuous, no error.
	if vs, err := checkFrontend(filepath.Join(dir, "nope")); err != nil || vs != nil {
		t.Fatalf("missing dir: violations=%v err=%v", vs, err)
	}
}

func TestAllowlistSuppresses(t *testing.T) {
	// The engine-level suppression is what matters; point the repo scan at a
	// scratch tree whose graph compiles (R1/R2 shapes are also compile-time
	// illegal in Go, so a compilable R3 edge proves the pipeline end-to-end).
	repo := t.TempDir()
	for _, dir := range []string{
		filepath.Join("backend-go", "cmd", "lll"),
		filepath.Join("backend-go", "internal", "modules", "projects"),
		filepath.Join("backend-go", "internal", "platform", "filesystem"),
	} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeGo := func(rel, body string) {
		if err := os.WriteFile(filepath.Join(repo, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeGo("go.mod", "module "+repoModule+"\n\ngo 1.25\n")
	writeGo(filepath.Join("backend-go", "cmd", "lll", "main.go"), "package main\nfunc main() {}\n")
	writeGo(filepath.Join("backend-go", "internal", "modules", "projects", "projects.go"), "package projects\n")
	writeGo(filepath.Join("backend-go", "internal", "platform", "filesystem", "fs.go"),
		"package filesystem\nimport _ \""+repoModule+"/backend-go/internal/modules/projects\"\n")

	importer := repoModule + "/backend-go/internal/platform/filesystem"
	target := repoModule + "/backend-go/internal/modules/projects"

	// Without the allowlist the violation is reported.
	violations, _, err := check(repo, "", false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range violations {
		if v.Rule == "R3-platform-imports-business" && v.Importer == importer && v.Imported == target {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected R3 violation before allowlisting, got %+v", violations)
	}

	// With the allowlist it is suppressed.
	allowFile := filepath.Join(t.TempDir(), "allow.txt")
	if err := os.WriteFile(allowFile, []byte("# temporary\n"+importer+" -> "+target+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, _, err = check(repo, allowFile, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		if v.Rule == "R3-platform-imports-business" && v.Importer == importer && v.Imported == target {
			t.Fatalf("allowlisted violation was still reported: %+v", v)
		}
	}
}

func TestListGoPackagesUsesDirectImportsOnly(t *testing.T) {
	repo := t.TempDir()
	for _, dir := range []string{
		filepath.Join("backend-go", "internal", "transport", "http"),
		filepath.Join("backend-go", "internal", "modules", "assets"),
		filepath.Join("backend-go", "internal", "modules", "assets", "internal", "store"),
	} {
		if err := os.MkdirAll(filepath.Join(repo, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeGo := func(rel, body string) {
		if err := os.WriteFile(filepath.Join(repo, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeGo("go.mod", "module "+repoModule+"\n\ngo 1.25\n")
	writeGo(filepath.Join("backend-go", "internal", "modules", "assets", "internal", "store", "store.go"), "package store\n")
	writeGo(filepath.Join("backend-go", "internal", "modules", "assets", "api.go"), "package assets\nimport _ \""+repoModule+"/backend-go/internal/modules/assets/internal/store\"\n")
	writeGo(filepath.Join("backend-go", "internal", "transport", "http", "router.go"), "package http\nimport _ \""+repoModule+"/backend-go/internal/modules/assets\"\n")

	packages, err := listGoPackages(repo, "./backend-go/...")
	if err != nil {
		t.Fatal(err)
	}
	transport := repoModule + "/backend-go/internal/transport/http"
	privateStore := repoModule + "/backend-go/internal/modules/assets/internal/store"
	for _, imported := range packages[transport] {
		if imported == privateStore {
			t.Fatalf("transport inherited facade's transitive dependency: %s", imported)
		}
	}
}
