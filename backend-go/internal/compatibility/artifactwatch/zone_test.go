package artifactwatch

import "testing"

func TestParseZonePath(t *testing.T) {
	cases := []struct {
		rel      string
		wantSlug string
		wantZone string
		wantOK   bool
	}{
		{"myproj/explain/pages/01.md", "myproj", "explain", true},
		{"myproj/explain/manifest.json", "myproj", "explain", true},
		{"abc/intro/output.md", "abc", "intro", true},
		{"abc/summary/summary.md", "", "", false},
		{"physics/overview.md", "physics", "overview", true},
		{"physics/learning-plan.json", "physics", "learning-plan", true},
		{"physics/discipline-topics.json", "physics", "discipline-topics", true},
		{"physics/learning-scope.json", "physics", "learning-scope", true},
		// ignored structural folders before a zone -> not a zone artifact
		{"abc/runs/2026-x/explain/result.md", "", "", false},
		{"abc/memory/note.md", "", "", false},
		{"abc/progress/summary.json", "", "", false},
		// no zone at all
		{"README.md", "", "", false},
		// windows backslashes
		{"myproj\\explain\\pages\\01.md", "myproj", "explain", true},
		{"physics\\overview.md", "physics", "overview", true},
		{"physics\\learning-plan.json", "physics", "learning-plan", true},
		{"physics\\discipline-topics.json", "physics", "discipline-topics", true},
	}
	for _, c := range cases {
		slug, zone, ok := parseZonePath(c.rel)
		if slug != c.wantSlug || zone != c.wantZone || ok != c.wantOK {
			t.Errorf("parseZonePath(%q) = (%q,%q,%t), want (%q,%q,%t)",
				c.rel, slug, zone, ok, c.wantSlug, c.wantZone, c.wantOK)
		}
	}
}
