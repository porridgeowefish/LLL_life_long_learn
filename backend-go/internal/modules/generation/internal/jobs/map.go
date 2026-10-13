package generation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	"github.com/xmz14/lll/backend-go/internal/modules/learning"
	"github.com/xmz14/lll/backend-go/internal/modules/projects"
)

type MapProjection struct {
	MapVersionID     string           `json:"mapVersionId"`
	Format           string           `json:"format"`
	OverviewMarkdown string           `json:"overviewMarkdown"`
	Topics           learning.Catalog `json:"topics"`
	Coverage         Coverage         `json:"coverage"`
}
type mapManifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	VersionID     string    `json:"versionId"`
	JobID         string    `json:"jobId,omitempty"`
	Format        string    `json:"format"`
	OverviewHash  string    `json:"overviewHash"`
	TopicsHash    string    `json:"topicsHash"`
	Coverage      Coverage  `json:"coverage"`
	CreatedAt     time.Time `json:"createdAt"`
}
type mapPointer struct {
	SchemaVersion int    `json:"schemaVersion"`
	VersionID     string `json:"versionId"`
}

var mapLocks sync.Map

func mapLock(root string) *sync.Mutex {
	v, _ := mapLocks.LoadOrStore(root, &sync.Mutex{})
	return v.(*sync.Mutex)
}
func ReadMap(slug, version string) (MapProjection, error) {
	root, e := projects.ProjectRootForSlug(slug)
	if e != nil {
		return MapProjection{}, e
	}
	state, e := projects.ReadProjectState(slug)
	if e != nil {
		return MapProjection{}, e
	}
	if state.ProjectType != projects.ProjectTypeDisciplineMap {
		return MapProjection{}, problem("map-project-type", 400)
	}
	mu := mapLock(root)
	mu.Lock()
	defer mu.Unlock()
	if version != "" && !safeID(version) {
		return MapProjection{}, problem("invalid-map-version", 400)
	}
	return readMapLocked(root, version)
}
func readMapLocked(root, version string) (MapProjection, error) {
	if version == "" {
		var pointer mapPointer
		e := readJSON(filepath.Join(root, "map", "current.json"), &pointer)
		if errors.Is(e, os.ErrNotExist) {
			if e = importLegacyMapLocked(root); e != nil {
				return MapProjection{}, e
			}
			e = readJSON(filepath.Join(root, "map", "current.json"), &pointer)
		}
		if e != nil || pointer.SchemaVersion != 1 || !safeID(pointer.VersionID) {
			return MapProjection{}, problem("map-recovery-required", 409)
		}
		version = pointer.VersionID
	}
	dir := filepath.Join(root, "map", "versions", version)
	var m mapManifest
	if e := readJSON(filepath.Join(dir, "manifest.json"), &m); e != nil {
		return MapProjection{}, problem("map-version-unavailable", 404)
	}
	overview, e := os.ReadFile(filepath.Join(dir, "overview.md"))
	if e != nil {
		return MapProjection{}, problem("map-recovery-required", 409)
	}
	topics, e := os.ReadFile(filepath.Join(dir, "topics.json"))
	if e != nil || hash(string(overview)) != m.OverviewHash || hash(string(topics)) != m.TopicsHash || m.VersionID != version {
		return MapProjection{}, problem("map-recovery-required", 409)
	}
	var catalog learning.Catalog
	if json.Unmarshal(topics, &catalog) != nil {
		catalog = learning.EmptyCatalog()
	}
	return MapProjection{MapVersionID: version, Format: m.Format, OverviewMarkdown: string(overview), Topics: catalog, Coverage: m.Coverage}, nil
}
func importLegacyMapLocked(root string) error {
	overview, e := os.ReadFile(filepath.Join(root, "overview.md"))
	if e != nil {
		return e
	}
	topics, e := os.ReadFile(filepath.Join(root, "discipline-topics.json"))
	if errors.Is(e, os.ErrNotExist) {
		topics, _ = json.Marshal(learning.EmptyCatalog())
	} else if e != nil {
		return e
	}
	format := "legacy"
	var catalog learning.Catalog
	if json.Unmarshal(topics, &catalog) == nil && learning.ValidateCatalog(&catalog) == nil && learning.ValidateAgainstOverview(catalog, string(overview)) == nil && len(catalog.Topics) > 0 {
		format = "structured"
	}
	id := "legacy-" + hash(string(overview) + string(topics))[:24]
	if e = writeMapVersion(root, id, "", format, overview, topics, Coverage{}); e != nil {
		return e
	}
	return writeJSON(filepath.Join(root, "map", "current.json"), mapPointer{SchemaVersion: 1, VersionID: id})
}
func writeMapVersion(root, id, job, format string, overview, topics []byte, coverage Coverage) error {
	dir := filepath.Join(root, "map", "versions", id)
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	if e := projects.AtomicWriteFile(filepath.Join(dir, "overview.md"), overview, 0600); e != nil {
		return e
	}
	if e := projects.AtomicWriteFile(filepath.Join(dir, "topics.json"), topics, 0600); e != nil {
		return e
	}
	return writeJSON(filepath.Join(dir, "manifest.json"), mapManifest{SchemaVersion: 1, VersionID: id, JobID: job, Format: format, OverviewHash: hash(string(overview)), TopicsHash: hash(string(topics)), Coverage: coverage, CreatedAt: time.Now().UTC()})
}
func publishMapLocked(slug, job, base string, out MapProjection) (string, error) {
	root, e := projects.ProjectRootForSlug(slug)
	if e != nil {
		return "", e
	}
	mu := mapLock(root)
	mu.Lock()
	defer mu.Unlock()
	current, e := readMapLocked(root, "")
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	if current.MapVersionID != base {
		return "", problem("map-version-conflict", 409)
	}
	if e = validateNewMap(&out); e != nil {
		return "", e
	}
	id := idgen.New("mapv")
	out.Topics.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	topics, e := json.Marshal(out.Topics)
	if e != nil {
		return "", e
	}
	if e = writeMapVersion(root, id, job, "structured", []byte(out.OverviewMarkdown), topics, out.Coverage); e != nil {
		return "", e
	}
	if e = writeJSON(filepath.Join(root, "map", "current.json"), mapPointer{SchemaVersion: 1, VersionID: id}); e != nil {
		return "", e
	}
	return id, nil
}
func mapResultForJob(slug, job string) ([]string, error) {
	root, e := projects.ProjectRootForSlug(slug)
	if e != nil {
		return nil, e
	}
	mu := mapLock(root)
	mu.Lock()
	defer mu.Unlock()
	var ptr mapPointer
	if e = readJSON(filepath.Join(root, "map", "current.json"), &ptr); errors.Is(e, os.ErrNotExist) {
		return nil, nil
	} else if e != nil {
		return nil, e
	}
	dir := filepath.Join(root, "map", "versions", ptr.VersionID)
	var m mapManifest
	if e = readJSON(filepath.Join(dir, "manifest.json"), &m); e != nil {
		return nil, e
	}
	if m.JobID == job {
		if _, e = readMapLocked(root, ptr.VersionID); e != nil {
			return nil, e
		}
		return []string{ptr.VersionID}, nil
	}
	return nil, nil
}
func validateNewMap(out *MapProjection) error {
	if learning.ValidateCatalog(&out.Topics) != nil {
		return problem("invalid-map-output", 409)
	}
	required := []string{"这门学科研究什么", "学科边界", "主要研究领域与知识架构", "领域之间如何关联", "常用研究方法与证据", "典型应用"}
	headings := []string{}
	chapter := ""
	chapters := map[string]int{}
	topicIndex := 0
	inside := false
	for _, line := range structuralLines(out.OverviewMarkdown) {
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			headings = append(headings, title)
			inside = title == "主要研究领域与知识架构"
		}
		if strings.HasPrefix(line, "### ") {
			if !inside {
				return problem("invalid-map-output", 409)
			}
			chapter = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			if _, ok := chapters[chapter]; ok {
				return problem("invalid-map-output", 409)
			}
			chapters[chapter] = 0
		}
		if strings.HasPrefix(line, "#### ") {
			if !inside || chapter == "" || topicIndex >= len(out.Topics.Topics) {
				return problem("invalid-map-output", 409)
			}
			topic := out.Topics.Topics[topicIndex]
			if topic.Priority == "" || topic.ChapterTitle != chapter || topic.TeachingOutline == "" || topic.Title != strings.TrimSpace(strings.TrimPrefix(line, "#### ")) {
				return problem("invalid-map-output", 409)
			}
			chapters[chapter]++
			topicIndex++
		}
	}
	if len(headings) != len(required) || len(chapters) < 4 || len(chapters) > 8 || topicIndex != len(out.Topics.Topics) {
		return problem("invalid-map-output", 409)
	}
	for i, title := range required {
		if headings[i] != title {
			return problem("invalid-map-output", 409)
		}
	}
	for _, n := range chapters {
		if n < 2 || n > 6 {
			return problem("invalid-map-output", 409)
		}
	}
	return nil
}
