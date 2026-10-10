package assistanttask

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
		if err != nil {
			return manifest, nil, errors.New("attempt asset input is unavailable")
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
func sourceRevisionIDs(manifest inputManifest) []string {
	out := make([]string, 0, len(manifest.Sources))
	for _, source := range manifest.Sources {
		out = append(out, source.RevisionID)
	}
	return out
}
