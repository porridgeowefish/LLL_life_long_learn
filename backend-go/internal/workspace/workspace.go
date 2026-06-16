package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/paths"
)

// ZoneName is the canonical name of one of the five learning zones.
type ZoneName string

const (
	ZoneIntro    ZoneName = "Intro"
	ZoneExplain  ZoneName = "Explain"
	ZonePractice ZoneName = "Practice"
	ZoneExtend   ZoneName = "Extend"
	ZoneSummary  ZoneName = "Summary"
)

// AllZones enumerates the five fixed zones in canonical order.
var AllZones = []ZoneName{ZoneIntro, ZoneExplain, ZonePractice, ZoneExtend, ZoneSummary}

// validSlugPattern is the strict slug validator for URL/path parameters.
// Allows lowercase ASCII letters/digits/hyphens AND CJK / other lowercase
// Unicode letters (so Chinese project slugs like "场论-笔记" work).
// Rejects uppercase ASCII to keep URLs predictable.
var validSlugPattern = regexp.MustCompile(`^[a-z0-9\p{Ll}\p{Lo}][a-z0-9\p{Ll}\p{Lo}\-]*$`)

// zoneFilename is the file each zone writes to (slice-relevant subset).
var zoneFilenames = map[ZoneName]string{
	ZoneIntro:    "output.md",
	ZoneExplain:  "output.md",
	ZonePractice: "tasks.json",
	ZoneExtend:   "prompts.md",
	ZoneSummary:  "summary.md",
}

// ProjectState is the persistent per-project state.
type ProjectState struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Slug            string        `json:"slug"`
	ParentProjectID string        `json:"parentProjectId,omitempty"`
	Status          string        `json:"status"`
	ActiveZone      ZoneName      `json:"activeZone"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	ChildProjectIDs []string      `json:"childProjectIds,omitempty"`
	LastArtifacts   []ArtifactRef `json:"lastArtifacts,omitempty"`
	GeneratedZones  []ZoneName    `json:"-"`
}

// ArtifactRef links a session/turn to a curated zone file.
type ArtifactRef struct {
	ZoneName  ZoneName  `json:"zoneName"`
	Filename  string    `json:"filename"`
	SessionID string    `json:"sessionId,omitempty"`
	RunDirRel string    `json:"runDirRel,omitempty"`
	WrittenAt time.Time `json:"writtenAt"`
}

// PredecessorFile represents one input to a zone's prompt.
type PredecessorFile struct {
	ZoneName ZoneName `json:"zoneName"`
	Path     string   `json:"path"`    // absolute path
	RelPath  string   `json:"relPath"` // relative to project root
	Exists   bool     `json:"exists"`
}

// ProjectMeta is the lightweight summary used in the tree listing.
type ProjectMeta struct {
	ID              string `json:"id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	ParentProjectID string `json:"parentProjectId,omitempty"`
	HasSubprojects  bool   `json:"hasSubprojects"`
}

// ValidateSlug returns true if s is a valid project slug.
// Used for URL/path parameter validation to prevent path traversal.
func ValidateSlug(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	return validSlugPattern.MatchString(s)
}

// ValidateZoneName returns true if z is one of the five zones.
func ValidateZoneName(z string) bool {
	for _, valid := range AllZones {
		if string(valid) == z {
			return true
		}
	}
	return false
}

// projectRoot returns the absolute path to a project's folder.
// Refuses slugs that traverse outside PROJECTS_ROOT.
func projectRoot(slug string) (string, error) {
	if !ValidateSlug(slug) {
		return "", fmt.Errorf("invalid slug: %q", slug)
	}
	root := filepath.Join(activeProjectsRoot(), slug)
	if _, err := os.Stat(filepath.Join(root, "state.json")); err == nil {
		return filepath.Abs(root)
	}
	if nested, ok := findNestedProjectRoot(activeProjectsRoot(), slug); ok {
		return filepath.Abs(nested)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	projectsAbs, err := filepath.Abs(activeProjectsRoot())
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(projectsAbs, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("slug escapes projects root: %q", slug)
	}
	return abs, nil
}

// findNestedProjectRoot locates an existing child project by leaf slug.
func findNestedProjectRoot(root, slug string) (string, bool) {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" || !d.IsDir() || d.Name() != slug || path == filepath.Join(root, slug) {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(path, "state.json")); statErr == nil {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found, found != ""
}

// subprojectRoot returns the path to a child project under a parent.
func subprojectRoot(parentSlug, childSlug string) (string, error) {
	if !ValidateSlug(childSlug) {
		return "", fmt.Errorf("invalid child slug: %q", childSlug)
	}
	parent, err := projectRoot(parentSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, "subprojects", childSlug), nil
}

// activeProjectsRoot returns the override (test) or the real PROJECTS_ROOT.
func activeProjectsRoot() string {
	if projectsRootOverride != "" {
		return projectsRootOverride
	}
	return paths.PROJECTS_ROOT
}

// ProjectExists reports whether a project state file is on disk.
func ProjectExists(slug string) (bool, error) {
	root, err := projectRoot(slug)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(root, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// CreateProjectSkeleton writes the canonical folder tree plus initial
// state.json, project.md, and memory files. Returns os.ErrExist-equivalent
// (via *SlugConflictError) if the project already exists.
// CreateProjectSkeleton creates a project with empty background fields.
// Kept for backward compatibility with tests and subproject handler.
func CreateProjectSkeleton(slug, title string, parentSlug string) error {
	return CreateProjectSkeletonWithInput(slug, title, parentSlug, ProjectInput{})
}

// CreateProjectSkeletonWithInput is the rich-form creator: writes project.md
// seeded with the user's why / current / target / standard fields.
func CreateProjectSkeletonWithInput(slug, title string, parentSlug string, in ProjectInput) error {
	var root string
	var err error
	if parentSlug != "" {
		root, err = subprojectRoot(parentSlug, slug)
	} else {
		root, err = projectRoot(slug)
	}
	if err != nil {
		return err
	}

	// Check for collision.
	if exists, err := os.Stat(filepath.Join(root, "state.json")); err == nil && !exists.IsDir() {
		return &SlugConflictError{Slug: slug}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Folder tree.
	dirs := []string{
		"", "memory", "intro", "explain", "practice", "extend", "summary",
		"progress", "runs", "runs/_index", "assets", "subprojects",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	parsed, _ := time.Parse(time.RFC3339, now)

	state := &ProjectState{
		ID:         slug,
		Title:      title,
		Slug:       slug,
		Status:     "active",
		ActiveZone: ZoneIntro,
		CreatedAt:  parsed,
		UpdatedAt:  parsed,
	}
	if parentSlug != "" {
		state.ParentProjectID = parentSlug
	}

	// state.json (atomic)
	if err := WriteProjectState(slug, state, parentSlug); err != nil {
		return err
	}

	// project.md
	projectMd := fmt.Sprintf("# %s\n\n%s", title, defaultProjectMdBodyForInput(in))
	if err := AtomicWriteFile(filepath.Join(root, "project.md"), []byte(projectMd), 0o644); err != nil {
		return err
	}

	// memory/project-memory.md
	memMd := "# Project Memory\n\n_What is already understood, where confusion remains, which examples worked._\n"
	if err := AtomicWriteFile(filepath.Join(root, "memory", "project-memory.md"), []byte(memMd), 0o644); err != nil {
		return err
	}

	// memory/project-state.json (empty initial state)
	memState := map[string]any{
		"understood":  []string{},
		"confusion":   []string{},
		"knownGaps":   []string{},
		"lastUpdated": parsed,
	}
	memBytes, _ := json.MarshalIndent(memState, "", "  ")
	if err := AtomicWriteFile(filepath.Join(root, "memory", "project-state.json"), memBytes, 0o644); err != nil {
		return err
	}

	// summary/summary.md — empty learner-owned file
	if err := AtomicWriteFile(filepath.Join(root, "summary", "summary.md"), []byte(""), 0o644); err != nil {
		return err
	}

	// If subproject, register in parent's state.
	if parentSlug != "" {
		parent, err := ReadProjectState(parentSlug)
		if err != nil {
			return fmt.Errorf("read parent state: %w", err)
		}
		parent.ChildProjectIDs = appendUnique(parent.ChildProjectIDs, slug)
		parent.UpdatedAt = parsed
		if err := WriteProjectState(parentSlug, parent, ""); err != nil {
			return fmt.Errorf("update parent state: %w", err)
		}
	}

	return nil
}

// CreateSubprojectWithInput applies the same learner background contract to a
// child project as CreateProjectSkeletonWithInput does to a root project.
func CreateSubprojectWithInput(parentSlug, slug, title string, in ProjectInput) error {
	return CreateProjectSkeletonWithInput(slug, title, parentSlug, in)
}

// ReadProjectState reads and decodes a project's state.json.
// For subprojects, pass the full slug path joined as "parent/child"
// — but for now this slice only supports top-level reads here.
// Callers needing subprojects should use IndexAll or ResolveByPath.
func ReadProjectState(slug string) (*ProjectState, error) {
	root, err := projectRoot(slug)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "state.json"))
	if err != nil {
		return nil, err
	}
	var s ProjectState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode state.json: %w", err)
	}
	s.GeneratedZones = detectGeneratedZones(root)
	return &s, nil
}

func detectGeneratedZones(root string) []ZoneName {
	var generated []ZoneName
	if nonEmptyFile(filepath.Join(root, "intro", "output.md")) ||
		validJSONObject(filepath.Join(root, "intro", "assessment.json")) {
		generated = append(generated, ZoneIntro)
	}
	if validExplainManifest(filepath.Join(root, "explain", "manifest.json")) ||
		nonEmptyFile(filepath.Join(root, "explain", "output.md")) {
		generated = append(generated, ZoneExplain)
	}
	if validTaskSet(filepath.Join(root, "practice", "tasks.json")) {
		generated = append(generated, ZonePractice)
	}
	if nonEmptyFile(filepath.Join(root, "extend", "relation-notes.md")) ||
		nonEmptyFile(filepath.Join(root, "extend", "prompts.md")) {
		generated = append(generated, ZoneExtend)
	}
	if nonEmptyFile(filepath.Join(root, "summary", "review-pack.md")) ||
		validFlashcardsFile(filepath.Join(root, "summary", "flashcards.json")) ||
		nonEmptyFile(filepath.Join(root, "summary", "summary.md")) {
		generated = append(generated, ZoneSummary)
	}
	return generated
}

func nonEmptyFile(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && len(strings.TrimSpace(string(data))) > 0
}

func validJSONObject(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return false
	}
	var value map[string]any
	return json.Unmarshal(data, &value) == nil
}

func validFlashcardsFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return false
	}
	data = []byte(stripJSONFence(strings.TrimSpace(strings.TrimPrefix(string(data), "\ufeff"))))
	var versioned struct {
		Cards      []json.RawMessage `json:"cards"`
		Flashcards []json.RawMessage `json:"flashcards"`
		Items      []json.RawMessage `json:"items"`
	}
	if json.Unmarshal(data, &versioned) == nil &&
		(len(versioned.Cards) > 0 || len(versioned.Flashcards) > 0 || len(versioned.Items) > 0) {
		return true
	}
	var legacy []json.RawMessage
	return json.Unmarshal(data, &legacy) == nil && len(legacy) > 0
}

func stripJSONFence(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return text
	}
	end := len(lines)
	if strings.HasPrefix(strings.TrimSpace(lines[end-1]), "```") {
		end--
	}
	return strings.TrimSpace(strings.Join(lines[1:end], "\n"))
}

func validExplainManifest(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var manifest struct {
		Pages []json.RawMessage `json:"pages"`
	}
	return json.Unmarshal(data, &manifest) == nil && len(manifest.Pages) > 0
}

func validTaskSet(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var taskSet struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	if json.Unmarshal(data, &taskSet) == nil && len(taskSet.Tasks) > 0 {
		return true
	}
	var legacy []json.RawMessage
	return json.Unmarshal(data, &legacy) == nil && len(legacy) > 0
}

// WriteProjectState persists state.json atomically. parentSlug is non-empty
// when writing a subproject's state so the file lands in subprojects/<child>/.
func WriteProjectState(slug string, state *ProjectState, parentSlug string) error {
	var root string
	var err error
	if parentSlug != "" {
		root, err = subprojectRoot(parentSlug, slug)
	} else {
		root, err = projectRoot(slug)
	}
	if err != nil {
		return err
	}
	state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(root, "state.json"), data, 0o644)
}

// IndexAll walks PROJECTS_ROOT recursively and returns every project
// (top-level + nested subprojects) as a flat list with parent IDs populated.
func IndexAll() ([]ProjectMeta, error) {
	var out []ProjectMeta
	if err := walkProjects(activeProjectsRoot(), "", &out); err != nil {
		return nil, err
	}
	// Stable sort by slug for deterministic listing.
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}

func walkProjects(dir string, parentID string, out *[]ProjectMeta) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "subprojects" || name == "runs" || name == "assets" || name == "memory" ||
			name == "intro" || name == "explain" || name == "practice" || name == "extend" || name == "summary" {
			continue
		}
		childDir := filepath.Join(dir, name)
		statePath := filepath.Join(childDir, "state.json")
		if _, err := os.Stat(statePath); err != nil {
			continue
		}
		data, err := os.ReadFile(statePath)
		if err != nil {
			continue
		}
		var s ProjectState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		hasSub := false
		if _, err := os.Stat(filepath.Join(childDir, "subprojects")); err == nil {
			if subs, err := os.ReadDir(filepath.Join(childDir, "subprojects")); err == nil {
				for _, se := range subs {
					if se.IsDir() {
						if _, err := os.Stat(filepath.Join(childDir, "subprojects", se.Name(), "state.json")); err == nil {
							hasSub = true
							break
						}
					}
				}
			}
		}
		*out = append(*out, ProjectMeta{
			ID:              s.ID,
			Slug:            s.Slug,
			Title:           s.Title,
			ParentProjectID: parentID,
			HasSubprojects:  hasSub,
		})
		// Recurse into subprojects/.
		_ = walkProjects(filepath.Join(childDir, "subprojects"), s.ID, out)
	}
	return nil
}

// ResolvePredecessorFiles returns the input file paths for a given zone
// according to the dependency graph:
//
//	Intro -> Explain
//	Explain -> Practice, Extend
//	Practice -> Extend
//	Intro + Explain + Practice + Extend -> Summary
func ResolvePredecessorFiles(slug string, zone ZoneName) ([]PredecessorFile, error) {
	root, err := projectRoot(slug)
	if err != nil {
		return nil, err
	}
	var preds []ZoneName
	switch zone {
	case ZoneIntro:
		preds = nil
	case ZoneExplain:
		preds = []ZoneName{ZoneIntro}
	case ZonePractice:
		preds = []ZoneName{ZoneExplain}
	case ZoneExtend:
		preds = []ZoneName{ZoneExplain, ZonePractice}
	case ZoneSummary:
		preds = []ZoneName{ZoneIntro, ZoneExplain, ZonePractice, ZoneExtend}
	default:
		return nil, fmt.Errorf("unknown zone: %s", zone)
	}
	var out []PredecessorFile
	for _, p := range preds {
		filename := zoneFilenames[p]
		absPath := filepath.Join(root, strings.ToLower(string(p)), filename)
		relPath := filepath.ToSlash(filepath.Join(strings.ToLower(string(p)), filename))
		_, err := os.Stat(absPath)
		exists := err == nil
		// For Summary zone, summary.md is a learner-owned output, not a predecessor.
		if p == ZoneSummary {
			continue
		}
		// Skip predecessors whose file doesn't exist (pending).
		if !exists && p != ZoneIntro {
			// Still include the path so prompt-assembly can list pending predecessors,
			// but only Intro is mandatory-empty.
		}
		out = append(out, PredecessorFile{
			ZoneName: p,
			Path:     absPath,
			RelPath:  relPath,
			Exists:   exists,
		})
	}
	return out, nil
}

// SafeWriteArtifact writes content into a zone folder atomically.
// Refuses to write to summary/summary.md; route through SafeWriteSummary instead.
func SafeWriteArtifact(slug string, zone ZoneName, filename string, content []byte) error {
	if !ValidateZoneName(string(zone)) {
		return fmt.Errorf("invalid zone: %s", zone)
	}
	if zone == ZoneSummary && filename == "summary.md" {
		return errors.New("SafeWriteArtifact refuses summary/summary.md; call SafeWriteSummary instead")
	}
	if !validFilename(filename) {
		return fmt.Errorf("invalid filename: %q", filename)
	}
	root, err := projectRoot(slug)
	if err != nil {
		return err
	}
	target := filepath.Join(root, strings.ToLower(string(zone)), filename)
	return AtomicWriteFile(target, content, 0o644)
}

// SafeWriteSummary writes summary/summary.md. Refuses to overwrite a non-empty
// existing file unless force is true. This is the learner-protection chokepoint.
func SafeWriteSummary(slug string, content []byte, force bool) error {
	root, err := projectRoot(slug)
	if err != nil {
		return err
	}
	target := filepath.Join(root, "summary", "summary.md")
	if !force {
		if existing, err := os.ReadFile(target); err == nil && len(strings.TrimSpace(string(existing))) > 0 {
			return errors.New("summary/summary.md is learner-owned; pass force=true to overwrite")
		}
	}
	return AtomicWriteFile(target, content, 0o644)
}

// ReadArtifact reads a zone file. Returns os.ErrNotExist-equivalent if missing.
func ReadArtifact(slug string, zone ZoneName, filename string) ([]byte, error) {
	if !ValidateZoneName(string(zone)) {
		return nil, fmt.Errorf("invalid zone: %s", zone)
	}
	if !validFilename(filename) {
		return nil, fmt.Errorf("invalid filename: %q", filename)
	}
	root, err := projectRoot(slug)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(root, strings.ToLower(string(zone)), filename))
}

// WriteRunFile writes a file under a project's runs/ directory.
// runDirRel is the run subfolder like "runs/2026-06-08T15-explain".
func WriteRunFile(slug, runDirRel, filename string, content []byte) error {
	if !validFilename(filename) {
		return fmt.Errorf("invalid filename: %q", filename)
	}
	root, err := projectRoot(slug)
	if err != nil {
		return err
	}
	target := filepath.Join(root, runDirRel, filename)
	if _, err := filepath.Rel(filepath.Join(root, "runs"), target); err != nil {
		return fmt.Errorf("run path escapes runs/: %w", err)
	}
	return AtomicWriteFile(target, content, 0o644)
}

// validFilename enforces a conservative filename pattern to avoid traversal.
// Allows letters, digits, dash, underscore, dot. Must not start with a dot.
func validFilename(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	if name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

func appendUnique(slice []string, s string) []string {
	for _, x := range slice {
		if x == s {
			return slice
		}
	}
	return append(slice, s)
}

// SlugConflictError indicates a project with the same slug already exists.
type SlugConflictError struct {
	Slug string
}

func (e *SlugConflictError) Error() string {
	return "slug already exists: " + e.Slug
}

// IsSlugConflict reports whether err is a *SlugConflictError.
func IsSlugConflict(err error) bool {
	var sc *SlugConflictError
	return errors.As(err, &sc)
}

func defaultProjectMdBody() string {
	return defaultProjectMdBodyForInput(ProjectInput{})
}

// ProjectInput captures the rich background fields collected at creation time.
// Why/Current/Target/Standard seed the project.md sections that follow the title.
type ProjectInput struct {
	Why      string // motivation: why this topic matters
	Current  string // current ability self-assessment
	Target   string // target ability self-assessment
	Standard string // completion criteria the learner commits to
}

func defaultProjectMdBodyForInput(in ProjectInput) string {
	why := strings.TrimSpace(in.Why)
	if why == "" {
		why = "_尚未填写：为什么这个主题值得学。_"
	}
	current := strings.TrimSpace(in.Current)
	if current == "" {
		current = "_尚未评估：当前在哪个水平。_"
	}
	target := strings.TrimSpace(in.Target)
	if target == "" {
		target = "_尚未设定：希望达到什么水平。_"
	}
	standard := strings.TrimSpace(in.Standard)
	if standard == "" {
		standard = "_尚未约定：怎样的成果算「完成」。_"
	}
	return fmt.Sprintf(`## Why this topic matters

%s

## Current ability

%s

## Target ability

%s

## Completion standard

%s

## Active Phase

Intro

## Notes

_Learner-owned. Agents will not overwrite this file without confirmation._`, why, current, target, standard)
}

// FindProjectBySlug scans IndexAll and returns the meta + the path-prefix
// (e.g. "" for top-level, "parent/child" for nested). Useful when callers
// have just a leaf slug and need to know if it's a subproject.
func FindProjectBySlug(slug string) (ProjectMeta, string, bool) {
	all, err := IndexAll()
	if err != nil {
		return ProjectMeta{}, "", false
	}
	for _, p := range all {
		if p.Slug == slug {
			path := ""
			if p.ParentProjectID != "" {
				path = filepath.Join(p.ParentProjectID, "subprojects", p.Slug)
			}
			return p, path, true
		}
	}
	return ProjectMeta{}, "", false
}

// EnsureProjectsRoot creates PROJECTS_ROOT if missing.
func EnsureProjectsRoot() error {
	return os.MkdirAll(activeProjectsRoot(), 0o755)
}

// ProjectRootForSlug returns the absolute path to a project's folder for
// external callers (memory store, launcher, etc.). Validates the slug.
func ProjectRootForSlug(slug string) (string, error) {
	return projectRoot(slug)
}

// projectsRootOverride is set by tests to redirect PROJECTS_ROOT to a temp
// directory. Production code never sets it (activeProjectsRoot falls back
// to paths.PROJECTS_ROOT when empty).
var projectsRootOverride string

// ProjectsRootForTest returns the current override (test helper).
func ProjectsRootForTest() string { return projectsRootOverride }

// SetProjectsRootForTest sets the override (test helper).
func SetProjectsRootForTest(dir string) { projectsRootOverride = dir }
