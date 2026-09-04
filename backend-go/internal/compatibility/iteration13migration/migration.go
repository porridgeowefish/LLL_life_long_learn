package iteration13migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	assets "github.com/xmz14/lll/backend-go/internal/modules/assets"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

type FileInventory struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}
type Record struct {
	SchemaVersion int             `json:"schemaVersion"`
	Status        string          `json:"status"`
	From          string          `json:"from"`
	Inventory     []FileInventory `json:"inventory"`
	InventoryHash string          `json:"inventoryHash"`
	BackupPath    string          `json:"backupPath"`
	FailureCode   string          `json:"failureCode,omitempty"`
	PreparedAt    time.Time       `json:"preparedAt"`
	CompletedAt   *time.Time      `json:"completedAt,omitempty"`
}

type Preflight struct {
	Ready          bool
	FailedProjects []string
}

func RunAll() Preflight {
	result := Preflight{Ready: true}
	projects, err := workspace.IndexAll()
	if err != nil {
		return Preflight{Ready: false, FailedProjects: []string{"workspace-index"}}
	}
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		if err := Migrate(project.Slug); err != nil {
			result.Ready = false
			result.FailedProjects = append(result.FailedProjects, project.Slug)
		}
	}
	return result
}

func Migrate(slug string) error {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return err
	}
	migrationDir := filepath.Join(root, "migrations", "iteration-13")
	recordPath := filepath.Join(migrationDir, "migration.json")
	unitExists := false
	if _, statErr := os.Stat(filepath.Join(root, "unit.json")); statErr == nil {
		unitExists = true
	}
	var record Record
	recordErr := readJSON(recordPath, &record)
	if unitExists && errors.Is(recordErr, os.ErrNotExist) {
		annotations, openErr := assets.NewAnnotations(slug)
		if openErr != nil {
			return openErr
		}
		return annotations.ImportLegacy()
	}
	if recordErr == nil && record.SchemaVersion == 1 && record.Status == "completed" {
		annotations, openErr := assets.NewAnnotations(slug)
		if openErr != nil {
			return openErr
		}
		return annotations.ImportLegacy()
	}
	backupDir := filepath.Join(migrationDir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	if recordErr != nil {
		if !errors.Is(recordErr, os.ErrNotExist) {
			return recordErr
		}
		inventory, inventoryErr := inventoryLegacy(root)
		if inventoryErr != nil {
			return inventoryErr
		}
		now := time.Now().UTC()
		record = Record{SchemaVersion: 1, Status: "prepared", From: "legacy-five-zone", Inventory: inventory, InventoryHash: inventoryHash(inventory), BackupPath: "migrations/iteration-13/backup", PreparedAt: now}
		if err := writeJSON(recordPath, record); err != nil {
			return err
		}
		if err := appendJournal(filepath.Join(migrationDir, "journal.jsonl"), "prepared", map[string]any{"inventoryHash": record.InventoryHash}); err != nil {
			return fail(migrationDir, record, "journal-write-failed", err)
		}
	} else {
		if record.SchemaVersion != 1 || record.InventoryHash != inventoryHash(record.Inventory) {
			return errors.New("invalid iteration-13 migration record")
		}
		_ = appendJournal(filepath.Join(migrationDir, "journal.jsonl"), "resumed", map[string]any{"previousStatus": record.Status})
	}
	for _, file := range record.Inventory {
		source := filepath.Join(root, filepath.FromSlash(file.Path))
		target := filepath.Join(backupDir, filepath.FromSlash(file.Path))
		if err := copyRegular(source, target); err != nil {
			return fail(migrationDir, record, "backup-failed", err)
		}
	}
	record.Status = "converting"
	if err := writeJSON(recordPath, record); err != nil {
		return err
	}
	if _, err := teacher.NewConversation(slug); err != nil {
		return fail(migrationDir, record, "conversation-conversion-failed", err)
	}
	if _, err := assets.New(slug); err != nil {
		return fail(migrationDir, record, "asset-conversion-failed", err)
	}
	if _, err := sourcestore.New(slug); err != nil {
		return fail(migrationDir, record, "source-conversion-failed", err)
	}
	if _, err := assistant.NewTaskStore(slug); err != nil {
		return fail(migrationDir, record, "task-conversion-failed", err)
	}
	annotations, err := assets.NewAnnotations(slug)
	if err != nil {
		return fail(migrationDir, record, "annotation-conversion-failed", err)
	}
	if err := annotations.ImportLegacy(); err != nil {
		return fail(migrationDir, record, "annotation-conversion-failed", err)
	}
	completed := time.Now().UTC()
	record.Status, record.CompletedAt = "completed", &completed
	if err := appendJournal(filepath.Join(migrationDir, "journal.jsonl"), "completed", map[string]any{"completedAt": completed}); err != nil {
		return err
	}
	return writeJSON(recordPath, record)
}

func inventoryLegacy(root string) ([]FileInventory, error) {
	legacy := map[string]bool{"intro": true, "explain": true, "practice": true, "extend": true, "summary": true}
	var files []FileInventory
	for dir := range legacy {
		base := filepath.Join(root, dir)
		walkErr := filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) {
					return nil
				}
				return walkErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, FileInventory{Path: filepath.ToSlash(rel), Bytes: info.Size(), SHA256: hash(data)})
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
			return nil, walkErr
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func inventoryHash(files []FileInventory) string { data, _ := json.Marshal(files); return hash(data) }
func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func copyRegular(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(target)
		if readErr != nil {
			return readErr
		}
		sourceData, readErr := os.ReadFile(source)
		if readErr != nil {
			return readErr
		}
		if hash(existing) == hash(sourceData) {
			return nil
		}
		return errors.New("backup target differs")
	}
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

func appendJournal(path, step string, detail any) error {
	entry := map[string]any{"schemaVersion": 1, "step": step, "at": time.Now().UTC(), "detail": detail}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(data, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func fail(dir string, record Record, code string, cause error) error {
	record.Status, record.FailureCode = "failed", code
	_ = writeJSON(filepath.Join(dir, "migration.json"), record)
	_ = appendJournal(filepath.Join(dir, "journal.jsonl"), "failed", map[string]any{"code": code})
	return fmt.Errorf("%s: %w", code, cause)
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(path, append(data, '\n'), 0o644)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
