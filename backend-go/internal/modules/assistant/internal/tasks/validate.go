package assistanttask

import (
	"encoding/xml"
	"os"
	"path/filepath"

	"fmt"
	"io"

	"encoding/json"
	"strings"

	"errors"
)

func readResult(path, taskID, runID string) (resultManifest, error) {
	var result resultManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if result.SchemaVersion != 1 || result.TaskID != taskID || result.RunID != runID || strings.TrimSpace(result.Summary) == "" || len([]byte(result.Summary)) > 16<<10 {
		return result, errors.New("result identity or summary is invalid")
	}
	return result, nil
}

func validateResult(workDir string, task Task, manifest inputManifest, result resultManifest) error {
	core := map[string]bool{"intro": true, "body": true, "practice": true}
	if len(result.AssetUpdates) != len(core) {
		return errors.New("every core asset must have an explicit result")
	}
	for key, update := range result.AssetUpdates {
		if !core[key] {
			return fmt.Errorf("unknown core asset: %s", key)
		}
		candidate := strings.TrimSpace(update.Candidate)
		switch update.Status {
		case "updated":
			expected := filepath.ToSlash(filepath.Join("asset-updates", key, "current.md"))
			if filepath.ToSlash(candidate) != expected {
				return fmt.Errorf("invalid candidate for %s", key)
			}
			path, ok := safeJoin(workDir, candidate)
			if !ok || !regularFile(path) {
				return fmt.Errorf("missing candidate for %s", key)
			}
		case "unchanged":
			if candidate != "" {
				return fmt.Errorf("unchanged asset %s declares a candidate", key)
			}
		case "failed":
			if candidate != "" || strings.TrimSpace(update.Code) == "" || len(update.Code) > 128 {
				return fmt.Errorf("invalid failed asset result for %s", key)
			}
		default:
			return fmt.Errorf("invalid asset status for %s", key)
		}
	}
	if task.Type == "consolidate" {
		if result.AssetUpdates["intro"].Status != "updated" || result.AssetUpdates["body"].Status != "updated" {
			return errors.New("consolidation must update intro and body")
		}
		if task.PracticeRequested {
			if result.AssetUpdates["practice"].Status != "updated" {
				return errors.New("consolidation with requested questions must update practice")
			}
		} else if result.AssetUpdates["practice"].Status != "unchanged" {
			return errors.New("default consolidation must leave practice unchanged")
		}
	}
	seenKeys := map[string]bool{}
	for _, deliverable := range result.Deliverables {
		if !safeOutputKey(deliverable.Key) || seenKeys[deliverable.Key] {
			return errors.New("invalid or duplicate deliverable key")
		}
		seenKeys[deliverable.Key] = true
		expected := filepath.ToSlash(filepath.Join("deliverables", deliverable.Key, "artifact.json"))
		if filepath.ToSlash(deliverable.Descriptor) != expected {
			return fmt.Errorf("descriptor does not match deliverable key %s", deliverable.Key)
		}
		if _, err := validateArtifactDescriptor(workDir, deliverable); err != nil {
			return err
		}
	}
	if result.SourceUpdate != nil {
		if task.Type != "source-processing" || result.SourceUpdate.SourceID == "" || result.SourceUpdate.RevisionID == "" {
			return errors.New("source update is not allowed for this task")
		}
		allowed := false
		for _, source := range manifest.Sources {
			if source.SourceID == result.SourceUpdate.SourceID && source.RevisionID == result.SourceUpdate.RevisionID {
				allowed = true
			}
		}
		if !allowed {
			return errors.New("source update is outside sealed input")
		}
		seen := map[string]bool{}
		prefix := filepath.ToSlash(filepath.Join("source-updates", result.SourceUpdate.RevisionID)) + "/"
		canonicalPath := prefix + "content.md"
		if len(result.SourceUpdate.Files) != 1 {
			return errors.New("source-processing must produce exactly one canonical markdown file")
		}
		for _, file := range result.SourceUpdate.Files {
			clean := filepath.ToSlash(file.Path)
			if file.Key != "content" || clean != canonicalPath || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(file.MediaType)), "text/markdown") || seen[file.Key] || !strings.HasPrefix(clean, prefix) {
				return errors.New("invalid source-derived file declaration")
			}
			seen[file.Key] = true
			path, ok := safeJoin(workDir, file.Path)
			if !ok || !regularFile(path) {
				return errors.New("missing source-derived file")
			}
		}
	} else if task.Type == "source-processing" {
		return errors.New("source-processing task omitted source update")
	}
	return nil
}

func validateArtifactDescriptor(workDir string, deliverable struct {
	Key        string `json:"key"`
	Descriptor string `json:"descriptor"`
}) (artifactDescriptor, error) {
	var descriptor artifactDescriptor
	descriptorPath, ok := safeJoin(workDir, deliverable.Descriptor)
	if !ok || !regularFile(descriptorPath) {
		return descriptor, errors.New("deliverable descriptor is missing")
	}
	if err := readJSON(descriptorPath, &descriptor); err != nil {
		return descriptor, err
	}
	if descriptor.SchemaVersion != 1 || strings.TrimSpace(descriptor.Kind) == "" || strings.TrimSpace(descriptor.Title) == "" || len(descriptor.Files) == 0 {
		return descriptor, errors.New("invalid artifact descriptor identity")
	}
	dir := filepath.Dir(descriptorPath)
	declared := map[string]bool{}
	declaredSources := map[string]bool{}
	for _, file := range descriptor.Files {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(file.Path)))
		if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(file.Path)) || declared[clean] || strings.TrimSpace(file.MediaType) == "" {
			return descriptor, errors.New("invalid artifact file declaration")
		}
		path, _, ok := resolveArtifactFile(workDir, dir, clean)
		if !ok || !regularFile(path) {
			return descriptor, errors.New("declared artifact file is missing")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return descriptor, err
		}
		if file.Bytes != int64(len(data)) || file.SHA256 != hashBytes(data) {
			return descriptor, errors.New("artifact file size or hash mismatch")
		}
		if strings.EqualFold(file.MediaType, "image/svg+xml") && unsafeSVG(data) {
			return descriptor, errors.New("artifact SVG contains active content")
		}
		declared[clean] = true
		declaredSources[filepath.Clean(path)] = true
	}
	for _, entry := range descriptor.EntryPoints {
		if !declared[filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry)))] {
			return descriptor, errors.New("artifact entry point is not a declared file")
		}
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return errors.New("deliverable contains unsupported filesystem entry")
		}
		if info.IsDir() || path == descriptorPath {
			return nil
		}
		if !declaredSources[filepath.Clean(path)] {
			return errors.New("deliverable contains undeclared file")
		}
		return nil
	})
	return descriptor, err
}

// resolveArtifactFile accepts both the canonical descriptor-relative form
// (files/report.md) and the workspace-relative form that interactive CLIs
// naturally produce (deliverables/report/files/report.md). The latter may
// reference a related deliverable such as an animation; it remains confined
// to workspace/deliverables and is repackaged under the same safe path.
func resolveArtifactFile(workDir, descriptorDir, declared string) (source, packageRel string, ok bool) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(declared)))
	if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(declared)) {
		return "", "", false
	}
	if strings.HasPrefix(clean, "deliverables/") {
		source, ok = safeJoin(workDir, clean)
		return source, clean, ok
	}
	source, ok = safeJoin(descriptorDir, clean)
	return source, clean, ok
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func safeOutputKey(key string) bool {
	if key == "" || len(key) > 80 {
		return false
	}
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func unsafeSVG(data []byte) bool {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	styleDepth := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		switch value := token.(type) {
		case xml.Directive, xml.ProcInst:
			return true
		case xml.StartElement:
			name := strings.ToLower(value.Name.Local)
			if name == "script" || name == "foreignobject" || name == "iframe" || name == "object" || name == "embed" {
				return true
			}
			if name == "style" {
				styleDepth++
			}
			for _, attribute := range value.Attr {
				attrName := strings.ToLower(attribute.Name.Local)
				attrValue := strings.ToLower(strings.TrimSpace(attribute.Value))
				if strings.HasPrefix(attrName, "on") && len(attrName) > 2 {
					return true
				}
				if attrName == "style" && unsafeSVGStyle(attrValue) {
					return true
				}
				if attrName == "href" || attrName == "src" {
					if strings.HasPrefix(attrValue, "#") || strings.HasPrefix(attrValue, "data:image/png") || strings.HasPrefix(attrValue, "data:image/jpeg") || strings.HasPrefix(attrValue, "data:image/jpg") {
						continue
					}
					if attrValue != "" {
						return true
					}
				}
			}
		case xml.CharData:
			if styleDepth > 0 && unsafeSVGStyle(string(value)) {
				return true
			}
		case xml.EndElement:
			if strings.EqualFold(value.Name.Local, "style") && styleDepth > 0 {
				styleDepth--
			}
		}
	}
}

func unsafeSVGStyle(value string) bool {
	compact := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "\t", "")
	if strings.Contains(compact, "javascript:") || strings.Contains(compact, "expression(") || strings.Contains(compact, "@import") {
		return true
	}
	for start := 0; ; {
		index := strings.Index(compact[start:], "url(")
		if index < 0 {
			return false
		}
		index += start
		if !strings.HasPrefix(compact[index:], "url(#") && !strings.HasPrefix(compact[index:], "url('#") && !strings.HasPrefix(compact[index:], "url(\"#") {
			return true
		}
		start = index + len("url(")
	}
}
