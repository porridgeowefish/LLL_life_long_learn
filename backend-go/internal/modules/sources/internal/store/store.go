package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

const (
	SchemaVersion        = 1
	DefaultFileLimit     = int64(100 << 20)
	DefaultUnitByteLimit = int64(1 << 30)
)

type Source struct {
	SchemaVersion      int        `json:"schemaVersion"`
	SourceID           string     `json:"sourceId"`
	DisplayName        string     `json:"displayName"`
	Status             string     `json:"status"`
	CurrentRevisionID  string     `json:"currentRevisionId"`
	TombstonedAt       *time.Time `json:"tombstonedAt,omitempty"`
	PermanentlyDeleted *time.Time `json:"permanentlyDeletedAt,omitempty"`
	FailureCode        string     `json:"failureCode,omitempty"`
	OriginalSHA256     string     `json:"originalSha256,omitempty"`
	OriginalBytes      int64      `json:"originalBytes,omitempty"`
	OriginalMediaType  string     `json:"originalMediaType,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// ParseDisposition decides whether source-processing may inspect a regular
// file. Archives and executable containers remain local opaque bytes.
func ParseDisposition(filename, mediaType string) (disposition, reason string) {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	blocked := map[string]string{
		".zip": "archive", ".rar": "archive", ".7z": "archive", ".tar": "archive", ".gz": "archive", ".bz2": "archive", ".xz": "archive", ".tgz": "archive",
		".exe": "executable", ".dll": "executable", ".msi": "executable", ".com": "executable", ".scr": "executable", ".apk": "executable", ".dmg": "executable", ".iso": "executable", ".jar": "executable", ".bat": "executable", ".cmd": "executable", ".ps1": "executable",
	}
	if reason, ok := blocked[extension]; ok {
		return "opaque", reason
	}
	allowed := map[string]bool{
		".pdf": true, ".doc": true, ".docx": true, ".odt": true, ".rtf": true, ".xls": true, ".xlsx": true, ".ods": true, ".csv": true, ".tsv": true, ".ppt": true, ".pptx": true, ".odp": true,
		".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true, ".tif": true, ".tiff": true,
		".txt": true, ".md": true, ".markdown": true, ".rst": true, ".json": true, ".jsonl": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true, ".html": true, ".htm": true, ".css": true,
		".go": true, ".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".java": true, ".c": true, ".h": true, ".cpp": true, ".hpp": true, ".rs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true, ".kts": true, ".sql": true, ".r": true, ".m": true, ".scala": true, ".lua": true,
	}
	if allowed[extension] {
		return "parse", "supported"
	}
	return "opaque", "unsupported"
}

type FileRef struct {
	Key       string `json:"key,omitempty"`
	Path      string `json:"path"`
	Filename  string `json:"filename,omitempty"`
	MediaType string `json:"mediaType"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
}

type Privacy struct {
	CloudDisclosureAccepted bool       `json:"cloudDisclosureAccepted"`
	AcceptedAt              *time.Time `json:"acceptedAt,omitempty"`
}

type Revision struct {
	SchemaVersion int       `json:"schemaVersion"`
	RevisionID    string    `json:"revisionId"`
	SourceID      string    `json:"sourceId"`
	Original      FileRef   `json:"original"`
	DerivedFiles  []FileRef `json:"derivedFiles"`
	ParseTaskID   string    `json:"parseTaskId,omitempty"`
	Privacy       Privacy   `json:"privacy"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Store struct {
	root          string
	mu            *sync.Mutex
	fileLimit     int64
	unitByteLimit int64
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
		return nil, errors.New("sources require system-learning project")
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	v, _ := locks.LoadOrStore(root, &sync.Mutex{})
	s := &Store{root: filepath.Join(root, "sources"), mu: v.(*sync.Mutex), fileLimit: DefaultFileLimit, unitByteLimit: DefaultUnitByteLimit}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) SetLimitsForTest(file, unit int64) { s.fileLimit, s.unitByteLimit = file, unit }

func (s *Store) Add(displayName, filename, mediaType string, size int64, r io.Reader, cloudAccepted bool) (Source, Revision, error) {
	if size < 0 || size > s.fileLimit {
		return Source{}, Revision{}, fmt.Errorf("source exceeds %d byte limit", s.fileLimit)
	}
	filename = sanitizeFilename(filename)
	if filename == "" {
		return Source{}, Revision{}, errors.New("invalid source filename")
	}
	if mediaType == "" {
		mediaType = mime.TypeByExtension(filepath.Ext(filename))
	}
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	used, err := s.bytesLocked()
	if err != nil {
		return Source{}, Revision{}, err
	}
	if used+size > s.unitByteLimit {
		return Source{}, Revision{}, fmt.Errorf("unit source storage exceeds %d byte limit", s.unitByteLimit)
	}

	sourceID, revisionID := idgen.New("source"), idgen.New("srev")
	dir := filepath.Join(s.root, sourceID, "revisions", revisionID)
	originalDir := filepath.Join(dir, "original")
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		return Source{}, Revision{}, err
	}
	tmp, err := os.CreateTemp(originalDir, ".upload-*")
	if err != nil {
		return Source{}, Revision{}, err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.RemoveAll(filepath.Join(s.root, sourceID))
		}
	}()
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, s.fileLimit+1))
	if syncErr := tmp.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return Source{}, Revision{}, copyErr
	}
	if n != size || n > s.fileLimit {
		return Source{}, Revision{}, errors.New("source byte count differs from declared size")
	}
	target := filepath.Join(originalDir, filename)
	if err := os.Rename(tmpName, target); err != nil {
		return Source{}, Revision{}, err
	}
	now := time.Now().UTC()
	var acceptedAt *time.Time
	if cloudAccepted {
		acceptedAt = &now
	}
	revision := Revision{SchemaVersion: SchemaVersion, RevisionID: revisionID, SourceID: sourceID, Original: FileRef{Path: filepath.ToSlash(filepath.Join("original", filename)), Filename: filename, MediaType: mediaType, Bytes: n, SHA256: "sha256:" + hex.EncodeToString(h.Sum(nil))}, Privacy: Privacy{CloudDisclosureAccepted: cloudAccepted, AcceptedAt: acceptedAt}, CreatedAt: now}
	source := Source{SchemaVersion: SchemaVersion, SourceID: sourceID, DisplayName: strings.TrimSpace(displayName), Status: "stored", CurrentRevisionID: revisionID, CreatedAt: now, UpdatedAt: now}
	if source.DisplayName == "" {
		source.DisplayName = filename
	}
	if err := writeJSON(filepath.Join(dir, "revision.json"), revision); err != nil {
		return Source{}, Revision{}, err
	}
	if err := writeJSON(filepath.Join(s.root, sourceID, "source.json"), source); err != nil {
		return Source{}, Revision{}, err
	}
	ok = true
	return source, revision, nil
}

func (s *Store) List() ([]Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var out []Source
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var source Source
		if readJSON(filepath.Join(s.root, entry.Name(), "source.json"), &source) == nil {
			out = append(out, source)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) Get(sourceID string) (Source, Revision, error) {
	if !safeID(sourceID, "source_") {
		return Source{}, Revision{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var source Source
	if err := readJSON(filepath.Join(s.root, sourceID, "source.json"), &source); err != nil {
		return Source{}, Revision{}, err
	}
	var revision Revision
	if err := readJSON(filepath.Join(s.root, sourceID, "revisions", source.CurrentRevisionID, "revision.json"), &revision); err != nil && source.PermanentlyDeleted == nil {
		return Source{}, Revision{}, err
	}
	return source, revision, nil
}

func (s *Store) SetStatus(sourceID, status, failureCode, parseTaskID string) (Source, error) {
	allowed := map[string]bool{"stored": true, "processing": true, "ready": true, "opaque": true, "failed": true, "tombstoned": true, "deleted": true}
	if !safeID(sourceID, "source_") || !allowed[status] {
		return Source{}, errors.New("invalid source status update")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, sourceID, "source.json")
	var source Source
	if err := readJSON(path, &source); err != nil {
		return Source{}, err
	}
	source.Status, source.FailureCode, source.UpdatedAt = status, failureCode, time.Now().UTC()
	if parseTaskID != "" {
		revisionPath := filepath.Join(s.root, sourceID, "revisions", source.CurrentRevisionID, "revision.json")
		var revision Revision
		if err := readJSON(revisionPath, &revision); err != nil {
			return Source{}, err
		}
		revision.ParseTaskID = parseTaskID
		if err := writeJSON(revisionPath, revision); err != nil {
			return Source{}, err
		}
	}
	return source, writeJSON(path, source)
}

func (s *Store) CommitDerived(sourceID, revisionID string, files map[string][]byte, mediaTypes map[string]string) (Revision, error) {
	if !safeID(sourceID, "source_") || !safeID(revisionID, "srev_") {
		return Revision{}, errors.New("invalid source identity")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	revisionPath := filepath.Join(s.root, sourceID, "revisions", revisionID, "revision.json")
	var revision Revision
	if err := readJSON(revisionPath, &revision); err != nil {
		return Revision{}, err
	}
	derivedDir := filepath.Join(filepath.Dir(revisionPath), "derived")
	var refs []FileRef
	for key, data := range files {
		clean := filepath.Clean(filepath.FromSlash(key))
		if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			return Revision{}, fmt.Errorf("invalid derived path: %q", key)
		}
		path := filepath.Join(derivedDir, clean)
		if err := workspace.AtomicWriteFile(path, data, 0o644); err != nil {
			return Revision{}, err
		}
		sum := sha256.Sum256(data)
		refs = append(refs, FileRef{Key: strings.TrimSuffix(filepath.Base(clean), filepath.Ext(clean)), Path: filepath.ToSlash(filepath.Join("derived", clean)), MediaType: mediaTypes[key], Bytes: int64(len(data)), SHA256: "sha256:" + hex.EncodeToString(sum[:])})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	revision.DerivedFiles = refs
	if err := writeJSON(revisionPath, revision); err != nil {
		return Revision{}, err
	}
	return revision, nil
}

func (s *Store) Tombstone(sourceID string) (Source, error) {
	now := time.Now().UTC()
	source, err := s.SetStatus(sourceID, "tombstoned", "", "")
	if err != nil {
		return Source{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	source.TombstonedAt, source.UpdatedAt = &now, now
	return source, writeJSON(filepath.Join(s.root, sourceID, "source.json"), source)
}

func (s *Store) PermanentlyDelete(sourceID string, activeReference bool) (Source, error) {
	if activeReference {
		return Source{}, errors.New("source is sealed by an active task")
	}
	if !safeID(sourceID, "source_") {
		return Source{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, sourceID, "source.json")
	var source Source
	if err := readJSON(path, &source); err != nil {
		return Source{}, err
	}
	if source.CurrentRevisionID != "" {
		var revision Revision
		if readJSON(filepath.Join(s.root, sourceID, "revisions", source.CurrentRevisionID, "revision.json"), &revision) == nil {
			source.OriginalSHA256 = revision.Original.SHA256
			source.OriginalBytes = revision.Original.Bytes
			source.OriginalMediaType = revision.Original.MediaType
		}
	}
	revisions := filepath.Join(s.root, sourceID, "revisions")
	if err := os.RemoveAll(revisions); err != nil {
		return Source{}, err
	}
	now := time.Now().UTC()
	source.Status, source.DisplayName, source.CurrentRevisionID = "deleted", "已删除资料", ""
	source.PermanentlyDeleted, source.UpdatedAt = &now, now
	return source, writeJSON(path, source)
}

func (s *Store) bytesLocked() (int64, error) {
	var total int64
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	if name == "." || name == ".." {
		return ""
	}
	return name
}

func safeID(id, prefix string) bool {
	return strings.HasPrefix(id, prefix) && !strings.ContainsAny(id, `/\\`) && len(id) <= 96
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
		return err
	}
	return workspace.AtomicWriteFile(path, append(data, '\n'), 0o644)
}
