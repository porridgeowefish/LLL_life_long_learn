package assetstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const SchemaVersion = 1

var coreKeys = []string{"intro", "body", "practice"}

type Meta struct {
	SchemaVersion      int       `json:"schemaVersion"`
	AssetID            string    `json:"assetId"`
	Key                string    `json:"key"`
	Title              string    `json:"title"`
	CurrentVersionID   string    `json:"currentVersionId"`
	EditRevision       uint64    `json:"editRevision"`
	ConversationCursor uint64    `json:"conversationCursor"`
	ContentHash        string    `json:"contentHash"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Version struct {
	SchemaVersion     int               `json:"schemaVersion"`
	VersionID         string            `json:"versionId"`
	AssetID           string            `json:"assetId"`
	ParentVersionID   string            `json:"parentVersionId,omitempty"`
	Author            string            `json:"author"`
	TaskID            string            `json:"taskId,omitempty"`
	RunID             string            `json:"runId,omitempty"`
	BaseVersionID     string            `json:"baseVersionId,omitempty"`
	ConversationRange map[string]uint64 `json:"conversationRange,omitempty"`
	SourceRevisionIDs []string          `json:"sourceRevisionIds,omitempty"`
	ContentHash       string            `json:"contentHash"`
	ChangeSummary     string            `json:"changeSummary,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
}

type Asset struct {
	Meta        Meta   `json:"meta"`
	Content     string `json:"content"`
	ContentKind string `json:"contentKind"`
}

type ConflictError struct{ CurrentRevision uint64 }

func (e *ConflictError) Error() string { return "asset edit conflict" }

type Store struct {
	root string
	mu   *sync.Mutex
}

var locks sync.Map

func New(slug string) (*Store, error) {
	if !workspace.ValidateSlug(slug) {
		return nil, errors.New("invalid project slug")
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		return nil, err
	}
	if state.ProjectType != workspace.ProjectTypeSystemLearning {
		return nil, errors.New("assets require system-learning project")
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	v, _ := locks.LoadOrStore(root, &sync.Mutex{})
	s := &Store{root: root, mu: v.(*sync.Mutex)}
	if err := s.ensure(); err != nil {
		return nil, err
	}
	return s, nil
}

func CoreKeys() []string { return append([]string(nil), coreKeys...) }

func validKey(key string) bool {
	for _, candidate := range coreKeys {
		if key == candidate {
			return true
		}
	}
	return false
}

func (s *Store) ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range coreKeys {
		dir := filepath.Join(s.root, "assets", key)
		if err := os.MkdirAll(filepath.Join(dir, "versions"), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, "asset.json")); err == nil {
			continue
		}
		content := s.legacyContent(key)
		if err := s.createInitialLocked(key, content, "migration"); err != nil {
			return err
		}
	}
	return os.MkdirAll(filepath.Join(s.root, "assets", "generated"), 0o755)
}

func (s *Store) legacyContent(key string) string {
	switch key {
	case "intro":
		return readText(filepath.Join(s.root, "intro", "output.md"))
	case "body":
		if text := readText(filepath.Join(s.root, "explain", "output.md")); strings.TrimSpace(text) != "" {
			return text
		}
		return s.legacyExplainPages()
	case "practice":
		if text := readText(filepath.Join(s.root, "practice", "output.md")); strings.TrimSpace(text) != "" {
			return text
		}
		if raw := readText(filepath.Join(s.root, "practice", "tasks.json")); strings.TrimSpace(raw) != "" {
			return "# 练习\n\n```json\n" + strings.TrimSpace(raw) + "\n```\n"
		}
	}
	return ""
}

func (s *Store) legacyExplainPages() string {
	var manifest struct {
		Title string `json:"title"`
		Pages []struct {
			Title string `json:"title"`
			File  string `json:"file"`
			Order int    `json:"order"`
		} `json:"pages"`
	}
	raw, err := os.ReadFile(filepath.Join(s.root, "explain", "manifest.json"))
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		return ""
	}
	sort.SliceStable(manifest.Pages, func(i, j int) bool { return manifest.Pages[i].Order < manifest.Pages[j].Order })
	var out strings.Builder
	if strings.TrimSpace(manifest.Title) != "" {
		out.WriteString("# " + strings.TrimSpace(manifest.Title) + "\n\n")
	}
	for _, page := range manifest.Pages {
		clean := filepath.Clean(filepath.FromSlash(page.File))
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			continue
		}
		content := readText(filepath.Join(s.root, "explain", clean))
		if strings.TrimSpace(content) == "" {
			continue
		}
		if strings.TrimSpace(page.Title) != "" && !strings.HasPrefix(strings.TrimSpace(content), "#") {
			out.WriteString("## " + strings.TrimSpace(page.Title) + "\n\n")
		}
		out.WriteString(strings.TrimSpace(content) + "\n\n")
	}
	return out.String()
}

func (s *Store) createInitialLocked(key, content, author string) error {
	dir := filepath.Join(s.root, "assets", key)
	now := time.Now().UTC()
	assetID := idgen.New("asset")
	versionID := idgen.New("aver")
	hash := contentHash(content)
	meta := Meta{SchemaVersion: SchemaVersion, AssetID: assetID, Key: key, Title: titleFor(key), CurrentVersionID: versionID, EditRevision: 1, ContentHash: hash, UpdatedAt: now}
	version := Version{SchemaVersion: SchemaVersion, VersionID: versionID, AssetID: assetID, Author: author, ContentHash: hash, ChangeSummary: "初始化教学资产", CreatedAt: now}
	return writeAssetFiles(dir, content, meta, version)
}

func (s *Store) List() ([]Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Meta, 0, len(coreKeys))
	for _, key := range coreKeys {
		meta, err := readMeta(filepath.Join(s.root, "assets", key, "asset.json"))
		if err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	return out, nil
}

func (s *Store) Get(key string) (Asset, error) {
	if !validKey(key) {
		return Asset{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, "assets", key)
	meta, err := readMeta(filepath.Join(dir, "asset.json"))
	if err != nil {
		return Asset{}, err
	}
	contentBytes, err := os.ReadFile(filepath.Join(dir, "current.md"))
	if err != nil {
		return Asset{}, err
	}
	content := string(contentBytes)
	if contentHash(content) != meta.ContentHash {
		updated, err := s.commitLocked(key, meta, content, "learner", "", "", meta.CurrentVersionID, nil, nil, "导入外部文件编辑", meta.ConversationCursor)
		if err != nil {
			return Asset{}, err
		}
		meta = updated
	}
	return s.assetLocked(key, meta, content), nil
}

func (s *Store) UpdateLearner(key string, baseRevision uint64, content string) (Asset, error) {
	if !validKey(key) {
		return Asset{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, "assets", key)
	meta, err := readMeta(filepath.Join(dir, "asset.json"))
	if err != nil {
		return Asset{}, err
	}
	if meta.EditRevision != baseRevision {
		return Asset{}, &ConflictError{CurrentRevision: meta.EditRevision}
	}
	meta, err = s.commitLocked(key, meta, content, "learner", "", "", meta.CurrentVersionID, nil, nil, "学习者编辑", meta.ConversationCursor)
	if err != nil {
		return Asset{}, err
	}
	return s.assetLocked(key, meta, content), nil
}

type CandidateInput struct {
	Key               string
	BaseVersionID     string
	BaseContent       string
	CandidateContent  string
	TaskID            string
	RunID             string
	FromSeq           uint64
	ThroughSeq        uint64
	SourceRevisionIDs []string
	ChangeSummary     string
}

type CandidateResult struct {
	Status string `json:"status"`
	Asset  *Asset `json:"asset,omitempty"`
	Code   string `json:"code,omitempty"`
}

func (s *Store) CommitCandidate(in CandidateInput) (CandidateResult, error) {
	if !validKey(in.Key) {
		return CandidateResult{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, "assets", in.Key)
	meta, err := readMeta(filepath.Join(dir, "asset.json"))
	if err != nil {
		return CandidateResult{}, err
	}
	// A dispatcher may restart after the immutable version was written but
	// before the task record reached terminal state. Treat the same task/run as
	// an idempotent commit instead of creating a duplicate version.
	versionDirs, _ := os.ReadDir(filepath.Join(dir, "versions"))
	for _, entry := range versionDirs {
		if !entry.IsDir() {
			continue
		}
		var version Version
		if readJSON(filepath.Join(dir, "versions", entry.Name(), "version.json"), &version) == nil && version.TaskID == in.TaskID && version.RunID == in.RunID {
			content := readText(filepath.Join(dir, "versions", version.VersionID, "content.md"))
			if version.AssetID != meta.AssetID || contentHash(content) != version.ContentHash {
				return CandidateResult{}, errors.New("existing assistant version is incomplete")
			}
			if meta.CurrentVersionID != version.VersionID && meta.CurrentVersionID != version.ParentVersionID {
				return CandidateResult{Status: "failed", Code: "edit-conflict"}, nil
			}
			if meta.CurrentVersionID == version.ParentVersionID {
				meta.EditRevision++
			}
			meta.CurrentVersionID = version.VersionID
			meta.ContentHash = version.ContentHash
			if through := version.ConversationRange["throughSeq"]; through > meta.ConversationCursor {
				meta.ConversationCursor = through
			}
			meta.UpdatedAt = version.CreatedAt
			if err := workspace.AtomicWriteFile(filepath.Join(dir, "current.md"), []byte(content), 0o644); err != nil {
				return CandidateResult{}, err
			}
			if err := writeJSON(filepath.Join(dir, "asset.json"), meta); err != nil {
				return CandidateResult{}, err
			}
			asset := s.assetLocked(in.Key, meta, content)
			return CandidateResult{Status: "updated", Asset: &asset}, nil
		}
	}
	current := readText(filepath.Join(dir, "current.md"))
	merged, ok := mergeThreeWay(in.BaseContent, current, in.CandidateContent)
	if !ok {
		return CandidateResult{Status: "failed", Code: "edit-conflict"}, nil
	}
	rangeMap := map[string]uint64{"fromSeq": in.FromSeq, "throughSeq": in.ThroughSeq}
	meta, err = s.commitLocked(in.Key, meta, merged, "assistant", in.TaskID, in.RunID, in.BaseVersionID, rangeMap, in.SourceRevisionIDs, in.ChangeSummary, in.ThroughSeq)
	if err != nil {
		return CandidateResult{}, err
	}
	committed := s.assetLocked(in.Key, meta, merged)
	asset := &committed
	return CandidateResult{Status: "updated", Asset: asset}, nil
}

// assetLocked projects the canonical editable asset into the appropriate
// learner-facing renderer. Migration versions may still be backed by the
// legacy structured protocols; later learner or assistant versions are plain
// canonical Markdown and must not be shadowed by stale legacy files.
func (s *Store) assetLocked(key string, meta Meta, content string) Asset {
	kind := "markdown"
	var current Version
	versionPath := filepath.Join(s.root, "assets", key, "versions", meta.CurrentVersionID, "version.json")
	if readJSON(versionPath, &current) == nil && current.Author == "migration" {
		switch key {
		case "body":
			if s.hasExplainPageContract() {
				kind = "explain-pages"
			}
		case "practice":
			if s.hasPracticeContract() {
				kind = "practice-set"
			}
		}
	}
	return Asset{Meta: meta, Content: content, ContentKind: kind}
}

func (s *Store) hasExplainPageContract() bool {
	var manifest struct {
		Pages []struct {
			File string `json:"file"`
		} `json:"pages"`
	}
	if readJSON(filepath.Join(s.root, "explain", "manifest.json"), &manifest) != nil || len(manifest.Pages) == 0 {
		return false
	}
	for _, page := range manifest.Pages {
		clean := filepath.Clean(filepath.FromSlash(page.File))
		if page.File == "" || filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, "..") {
			return false
		}
		info, err := os.Stat(filepath.Join(s.root, "explain", clean))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func (s *Store) hasPracticeContract() bool {
	var tasks struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	return readJSON(filepath.Join(s.root, "practice", "tasks.json"), &tasks) == nil && len(tasks.Tasks) > 0
}

func (s *Store) AdvanceCursor(key string, throughSeq uint64) (Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, "assets", key)
	meta, err := readMeta(filepath.Join(dir, "asset.json"))
	if err != nil {
		return Meta{}, err
	}
	if throughSeq > meta.ConversationCursor {
		meta.ConversationCursor = throughSeq
		meta.UpdatedAt = time.Now().UTC()
		if err := writeJSON(filepath.Join(dir, "asset.json"), meta); err != nil {
			return Meta{}, err
		}
	}
	return meta, nil
}

func (s *Store) commitLocked(key string, meta Meta, content, author, taskID, runID, baseVersionID string, conversationRange map[string]uint64, sourceRevisionIDs []string, summary string, cursor uint64) (Meta, error) {
	dir := filepath.Join(s.root, "assets", key)
	now := time.Now().UTC()
	versionID := idgen.New("aver")
	version := Version{SchemaVersion: SchemaVersion, VersionID: versionID, AssetID: meta.AssetID, ParentVersionID: meta.CurrentVersionID, Author: author, TaskID: taskID, RunID: runID, BaseVersionID: baseVersionID, ConversationRange: conversationRange, SourceRevisionIDs: sourceRevisionIDs, ContentHash: contentHash(content), ChangeSummary: summary, CreatedAt: now}
	meta.CurrentVersionID = versionID
	meta.EditRevision++
	meta.ContentHash = version.ContentHash
	meta.ConversationCursor = cursor
	meta.UpdatedAt = now
	if err := writeAssetFiles(dir, content, meta, version); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func (s *Store) Versions(key string) ([]Version, error) {
	if !validKey(key) {
		return nil, os.ErrNotExist
	}
	dir := filepath.Join(s.root, "assets", key, "versions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Version
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var v Version
		if readJSON(filepath.Join(dir, entry.Name(), "version.json"), &v) == nil {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func writeAssetFiles(dir, content string, meta Meta, version Version) error {
	versionDir := filepath.Join(dir, "versions", version.VersionID)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return err
	}
	if err := workspace.AtomicWriteFile(filepath.Join(versionDir, "content.md"), []byte(content), 0o644); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(versionDir, "version.json"), version); err != nil {
		return err
	}
	if err := workspace.AtomicWriteFile(filepath.Join(dir, "current.md"), []byte(content), 0o644); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "asset.json"), meta)
}

func mergeThreeWay(base, current, candidate string) (string, bool) {
	if current == base || current == candidate {
		return candidate, true
	}
	if candidate == base {
		return current, true
	}
	// Conservative append merge covers the common cumulative case and refuses
	// overlapping rewrites rather than risking learner content.
	if base != "" && strings.HasPrefix(current, base) && strings.HasPrefix(candidate, base) {
		left := strings.TrimPrefix(current, base)
		right := strings.TrimPrefix(candidate, base)
		if left == "" || right == "" || !strings.Contains(left, right) {
			return base + left + right, true
		}
	}
	return current, false
}

func titleFor(key string) string {
	switch key {
	case "intro":
		return "引入"
	case "body":
		return "正文"
	case "practice":
		return "练习"
	default:
		return key
	}
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func readMeta(path string) (Meta, error) {
	var meta Meta
	err := readJSON(path, &meta)
	return meta, err
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return workspace.AtomicWriteFile(path, append(data, '\n'), 0o644)
}
