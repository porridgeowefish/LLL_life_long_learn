package assistanttask

import (
	"encoding/json"

	"errors"

	"crypto/sha256"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"

	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"io"

	"strings"
)

func loadSealedInputs(task Task, projectRoot, workDir string) (inputManifest, map[string]AssetSnapshot, error) {
	var manifest inputManifest
	if err := readJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), &manifest); err != nil {
		return manifest, nil, err
	}
	if manifest.SchemaVersion != 1 || manifest.TaskID != task.ID {
		return manifest, nil, errors.New("sealed input identity mismatch")
	}
	bases := map[string]AssetSnapshot{}
	for key, entry := range manifest.Assets {
		content, err := os.ReadFile(filepath.Join(workDir, "inputs", "assets", key, "current.md"))
		if err != nil || hashBytes(content) != entry.SHA256 {
			return manifest, nil, errors.New("sealed asset input mismatch")
		}
		bases[key] = AssetSnapshot{VersionID: entry.VersionID, ConversationCursor: entry.Cursor, Content: string(content)}
	}
	return manifest, bases, nil
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func safeJoin(root, rel string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", false
	}
	path := filepath.Join(root, clean)
	back, err := filepath.Rel(root, path)
	return path, err == nil && !strings.HasPrefix(back, "..")
}
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func sourceRevisionIDs(manifest inputManifest) []string {
	out := make([]string, 0, len(manifest.Sources))
	for _, source := range manifest.Sources {
		out = append(out, source.RevisionID)
	}
	return out
}

func artifactIDFor(taskID, runID, key string) string {
	sum := sha256.Sum256([]byte(taskID + "\x00" + runID + "\x00" + key))
	return "artifact_" + hex.EncodeToString(sum[:16])
}

func artifactOwnedBy(dir, taskID, runID string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "artifact.json"))
	if err != nil {
		return false
	}
	var artifact struct {
		Provenance struct {
			TaskID string `json:"taskId"`
			RunID  string `json:"runId"`
		} `json:"provenance"`
	}
	return json.Unmarshal(data, &artifact) == nil && artifact.Provenance.TaskID == taskID && artifact.Provenance.RunID == runID
}

func (d *Dispatcher) commitGeneratedArtifact(projectRoot string, task Task, runID, workDir string, manifest inputManifest, deliverable struct {
	Key        string `json:"key"`
	Descriptor string `json:"descriptor"`
}) (string, error) {
	descriptorPath, ok := safeJoin(workDir, deliverable.Descriptor)
	if !ok || !strings.HasPrefix(filepath.Clean(descriptorPath), filepath.Join(workDir, "deliverables")+string(os.PathSeparator)) {
		return "", errors.New("artifact descriptor is outside deliverables")
	}
	artifactID := artifactIDFor(task.ID, runID, deliverable.Key)
	target := filepath.Join(projectRoot, "assets", "generated", artifactID)
	if artifactOwnedBy(target, task.ID, runID) {
		return artifactID, nil
	}
	validated, err := validateArtifactDescriptor(workDir, deliverable)
	if err != nil {
		return "", err
	}
	staging := target + ".staging-" + runID
	_ = os.RemoveAll(staging)
	if err := stageArtifactPackage(workDir, filepath.Dir(descriptorPath), staging, validated); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	var artifact map[string]any
	if data, readErr := os.ReadFile(filepath.Join(staging, "artifact.json")); readErr != nil || json.Unmarshal(data, &artifact) != nil {
		_ = os.RemoveAll(staging)
		return "", errors.New("staged artifact descriptor is invalid")
	}
	artifact["schemaVersion"] = 1
	artifact["artifactId"] = artifactID
	artifact["provenance"] = map[string]any{"taskId": task.ID, "runId": runID, "conversationRange": map[string]uint64{"throughSeq": task.ConversationCutoffSeq}, "sourceRevisionIds": sourceRevisionIDs(manifest)}
	artifact["createdAt"] = time.Now().UTC()
	if err := writeJSON(filepath.Join(staging, "artifact.json"), artifact); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	if err := os.Rename(staging, target); err != nil {
		_ = os.RemoveAll(staging)
		if !artifactOwnedBy(target, task.ID, runID) {
			return "", err
		}
	}
	return artifactID, nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if errors.Is(err, os.ErrExist) {
			return err
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
	})
}

func stageArtifactPackage(workDir, descriptorDir, destination string, descriptor artifactDescriptor) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, file := range descriptor.Files {
		source, packageRel, ok := resolveArtifactFile(workDir, descriptorDir, file.Path)
		if !ok {
			return errors.New("invalid staged artifact path")
		}
		target, ok := safeJoin(destination, packageRel)
		if !ok {
			return errors.New("invalid artifact package path")
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := workspace.AtomicWriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(destination, "artifact.json"), descriptor)
}
