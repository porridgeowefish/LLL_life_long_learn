package assetstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

type MigrationResult struct {
	ProjectSlug  string
	State        string
	SourceBytes  int64
	ReviewBytes  int64
	RemovedFiles int
}

type reviewMigration struct {
	SchemaVersion int       `json:"schemaVersion"`
	State         string    `json:"state"`
	ReviewHash    string    `json:"reviewHash"`
	SourceBytes   int64     `json:"sourceBytes"`
	ReviewBytes   int64     `json:"reviewBytes"`
	RemovedFiles  int       `json:"removedFiles"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func MigrateReview(slug string) (MigrationResult, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return MigrationResult{}, err
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		return MigrationResult{}, err
	}
	if state.ProjectType != workspace.ProjectTypeSystemLearning {
		return MigrationResult{ProjectSlug: slug, State: "completed"}, nil
	}
	value, _ := locks.LoadOrStore(root, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	recordPath := filepath.Join(root, "migrations", "iteration-18.json")
	record := reviewMigration{SchemaVersion: 1, State: "prepared"}
	if err := readJSON(recordPath, &record); err != nil && !errors.Is(err, os.ErrNotExist) {
		return MigrationResult{}, err
	}
	if record.SchemaVersion != 1 {
		return MigrationResult{}, errors.New("unsupported review migration record")
	}
	result := func() MigrationResult {
		return MigrationResult{ProjectSlug: slug, State: record.State, SourceBytes: record.SourceBytes, ReviewBytes: record.ReviewBytes, RemovedFiles: record.RemovedFiles}
	}
	if record.State == "completed" {
		return result(), nil
	}
	if err := rejectLinks(root, "assets"); err != nil {
		return result(), err
	}
	reviewDir := filepath.Join(root, "assets", "review")
	if record.State == "prepared" {
		intro, err := migrationSource(root, "intro")
		if err != nil {
			return result(), err
		}
		body, err := migrationSource(root, "body")
		if err != nil {
			return result(), err
		}
		content := joinReview(intro, body)
		record.SourceBytes = int64(len(intro) + len(body))
		if _, err := os.Stat(filepath.Join(reviewDir, "asset.json")); err == nil {
			// A restart may occur after the directory rename but before journal write.
			meta, err := readMeta(filepath.Join(reviewDir, "asset.json"))
			if err != nil {
				return result(), err
			}
			existing, err := os.ReadFile(filepath.Join(reviewDir, "current.md"))
			if err != nil {
				return result(), err
			}
			if contentHash(string(existing)) != meta.ContentHash {
				return result(), errors.New("existing review hash does not match metadata")
			}
			// Only the exact migration output proves a completed rename. A different
			// review must never authorize deleting unmerged source text.
			if string(existing) != content && record.SourceBytes > 0 {
				return result(), errors.New("existing review conflicts with unmerged intro/body; sources retained")
			}
			content = string(existing)
		} else if !errors.Is(err, os.ErrNotExist) {
			return result(), err
		} else {
			stage := filepath.Join(root, "assets", ".review-migration")
			if err := safeRemove(root, stage); err != nil {
				return result(), err
			}
			store := &Store{root: root, mu: mu}
			if err := store.createInitialAt(stage, "review", content, "migration"); err != nil {
				return result(), err
			}
			staged, err := os.ReadFile(filepath.Join(stage, "current.md"))
			if err != nil || string(staged) != content {
				return result(), errors.New("staged review write verification failed")
			}
			if err := os.Rename(stage, reviewDir); err != nil {
				return result(), fmt.Errorf("commit review directory: %w", err)
			}
		}
		record.ReviewHash = contentHash(content)
		record.ReviewBytes = int64(len(content))
		record.State = "review-committed"
		record.UpdatedAt = time.Now().UTC()
		if err := writeJSON(recordPath, record); err != nil {
			return result(), err
		}
	}
	if record.State != "review-committed" {
		return result(), errors.New("unknown review migration stage")
	}
	current, err := os.ReadFile(filepath.Join(reviewDir, "current.md"))
	if err != nil {
		return result(), err
	}
	if contentHash(string(current)) != record.ReviewHash {
		return result(), errors.New("committed review changed before migration cleanup")
	}
	if err := migrateReviewAnnotations(root); err != nil {
		return result(), err
	}
	practiceDir := filepath.Join(root, "assets", "practice")
	if _, err := os.Stat(filepath.Join(practiceDir, "asset.json")); errors.Is(err, os.ErrNotExist) {
		content, err := migrationPractice(root)
		if err != nil {
			return result(), err
		}
		store := &Store{root: root, mu: mu}
		if err := store.createInitialAt(practiceDir, "practice", content, "migration"); err != nil {
			return result(), err
		}
	} else if err != nil {
		return result(), err
	} else {
		content, err := migrationPractice(root)
		if err != nil {
			return result(), err
		}
		meta, err := readMeta(filepath.Join(practiceDir, "asset.json"))
		if err != nil {
			return result(), err
		}
		current, err := os.ReadFile(filepath.Join(practiceDir, "current.md"))
		if err != nil {
			return result(), err
		}
		if string(current) != content {
			store := &Store{root: root, mu: mu}
			if _, err := store.commitLocked("practice", meta, content, "migration", "", "", meta.CurrentVersionID, nil, nil, "将旧练习转换为 Markdown", meta.ConversationCursor); err != nil {
				return result(), err
			}
		}
	}
	// Make all later stores use the new keys; notes is not an assistant candidate.
	unitPath := filepath.Join(root, "unit.json")
	var unit map[string]any
	if err := readJSON(unitPath, &unit); err == nil {
		unit["activeAssetKeys"] = []string{"review", "practice", "notes"}
		if err := writeJSON(unitPath, unit); err != nil {
			return result(), err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return result(), err
	}
	for _, key := range []string{"intro", "body", "generated"} {
		target := filepath.Join(root, "assets", key)
		count, err := countRegularFiles(target)
		if err != nil {
			return result(), err
		}
		if err := safeRemove(root, target); err != nil {
			return result(), err
		}
		record.RemovedFiles += count
	}
	record.State = "completed"
	record.UpdatedAt = time.Now().UTC()
	if err := writeJSON(recordPath, record); err != nil {
		return result(), err
	}
	return result(), nil
}

func joinReview(intro, body string) string {
	intro = strings.TrimPrefix(intro, "\ufeff")
	body = strings.TrimPrefix(body, "\ufeff")
	if intro == "" {
		return body
	}
	if body == "" {
		return intro
	}
	separator := "\n---\n\n"
	if !strings.HasSuffix(intro, "\n") {
		separator = "\n" + separator
	}
	return intro + separator + body
}

func (s *Store) createInitialAt(dir, key, content, author string) error {
	// Reuse initial metadata generation without changing canonical store paths.
	now := time.Now().UTC()
	assetID, versionID := idgen.New("asset"), idgen.New("aver")
	hash := contentHash(content)
	meta := Meta{SchemaVersion: SchemaVersion, AssetID: assetID, Key: key, Title: titleFor(key), CurrentVersionID: versionID, EditRevision: 1, ContentHash: hash, UpdatedAt: now}
	version := Version{SchemaVersion: SchemaVersion, VersionID: versionID, AssetID: assetID, Author: author, ContentHash: hash, ChangeSummary: "迁移教学资产", CreatedAt: now}
	return writeAssetFiles(dir, content, meta, version)
}

func rejectLinks(root, relative string) error {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	target := filepath.Join(root, relative)
	return filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("migration refuses linked path: %s", relative)
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(canonicalRoot, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("migration path leaves project")
		}
		return nil
	})
}

func safeRemove(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("invalid migration deletion target")
	}
	if err := rejectLinks(root, rel); err != nil {
		return err
	}
	return os.RemoveAll(target)
}

func countRegularFiles(root string) (int, error) {
	count := 0
	err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			count++
		}
		return nil
	})
	return count, err
}

func migrateReviewAnnotations(root string) error {
	source := filepath.Join(root, "assets", "body", "annotations.jsonl")
	raw, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	target := filepath.Join(root, "assets", "review", "annotations.jsonl")
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	meta, err := readMeta(filepath.Join(root, "assets", "review", "asset.json"))
	if err != nil {
		return err
	}
	var out strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("decode old annotation: %w", err)
		}
		if annotation, ok := event["annotation"].(map[string]any); ok {
			annotation["assetId"] = meta.AssetID
			annotation["legacySource"] = true
			annotation["detached"] = true
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return err
		}
		out.Write(encoded)
		out.WriteByte('\n')
	}
	return workspace.AtomicWriteFile(target, []byte(out.String()), 0644)
}
