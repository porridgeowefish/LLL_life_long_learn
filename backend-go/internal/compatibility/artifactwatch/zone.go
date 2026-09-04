package artifactwatch

import "strings"

// zones is the set of folder names that hold generated artifacts.
var zones = map[string]bool{
	"intro":    true,
	"explain":  true,
	"practice": true,
}

// ignoreFolders are structural project folders that are NOT zones. If one
// appears before a zone segment, the path is inside that structural folder
// (e.g. runs/<ts>/explain/result.md) and must not be treated as a zone write.
var ignoreFolders = map[string]bool{
	"runs":     true,
	"memory":   true,
	"assets":   true,
	"progress": true,
}

// parseZonePath maps a path relative to the projects root to the owning project
// slug and the zone whose artifact changed. ok is false when no zone folder
// appears or when an ignored structural folder precedes the zone. Separators are
// normalized to "/", so slug uses "/" (matching the frontend slug convention).
func parseZonePath(rel string) (slug, zone string, ok bool) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	parts := strings.Split(rel, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] == "overview.md" {
		return parts[0], "overview", true
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "learning-plan.json" {
		return parts[0], "learning-plan", true
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "discipline-topics.json" {
		return parts[0], "discipline-topics", true
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "learning-scope.json" {
		return parts[0], "learning-scope", true
	}
	for i, seg := range parts {
		if ignoreFolders[seg] {
			return "", "", false
		}
		if zones[seg] {
			return strings.Join(parts[:i], "/"), seg, true
		}
	}
	return "", "", false
}
