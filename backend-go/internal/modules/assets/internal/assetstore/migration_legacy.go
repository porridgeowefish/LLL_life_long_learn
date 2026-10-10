package assetstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func migrationText(filename string) (string, bool, error) {
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

func migrationSource(root, key string) (string, error) {
	canonical, exists, err := migrationText(filepath.Join(root, "assets", key, "current.md"))
	if err != nil {
		return "", err
	}
	if exists {
		legacy, err := legacyMigrationVersion(filepath.Join(root, "assets", key))
		if err != nil {
			return "", err
		}
		if key != "body" || !legacy || canonical == "" {
			return canonical, nil
		}
	}
	if key == "intro" {
		text, _, err := migrationText(filepath.Join(root, "intro", "output.md"))
		return text, err
	}
	manifestText, hasManifest, err := migrationText(filepath.Join(root, "explain", "manifest.json"))
	if err != nil {
		return "", err
	}
	if !hasManifest {
		if exists {
			return canonical, nil
		}
		text, _, err := migrationText(filepath.Join(root, "explain", "output.md"))
		return text, err
	}
	var manifest struct {
		Pages []struct {
			Title string `json:"title"`
			File  string `json:"file"`
			Order int    `json:"order"`
		} `json:"pages"`
	}
	if err := json.Unmarshal([]byte(manifestText), &manifest); err != nil {
		return "", fmt.Errorf("decode Explain manifest: %w", err)
	}
	sort.SliceStable(manifest.Pages, func(i, j int) bool { return manifest.Pages[i].Order < manifest.Pages[j].Order })
	var out strings.Builder
	for i, page := range manifest.Pages {
		rel := filepath.Clean(filepath.FromSlash(page.File))
		if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", errors.New("invalid Explain page path")
		}
		if err := rejectLinks(root, filepath.Join("explain", rel)); err != nil {
			return "", err
		}
		text, present, err := migrationText(filepath.Join(root, "explain", rel))
		if err != nil {
			return "", err
		}
		if !present {
			return "", errors.New("Explain page missing")
		}
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(strings.TrimPrefix(text, "\ufeff"))
		if !strings.HasSuffix(text, "\n") {
			out.WriteByte('\n')
		}
	}
	if len(manifest.Pages) == 0 && exists {
		return canonical, nil
	}
	if exists && canonical != out.String() {
		// Earlier migration kept a readable manifest/summary in current.md;
		// retain it as well as the actual old reader pages, never substitute.
		return joinReview(canonical, out.String()), nil
	}
	return out.String(), nil
}

func migrationPractice(root string) (string, error) {
	if text, exists, err := migrationText(filepath.Join(root, "assets", "practice", "current.md")); err != nil {
		return "", err
	} else if exists {
		legacy, err := legacyMigrationVersion(filepath.Join(root, "assets", "practice"))
		if err != nil {
			return "", err
		}
		if !legacy {
			return text, nil
		}
		if _, exists, err := migrationText(filepath.Join(root, "practice", "tasks.json")); err != nil {
			return "", err
		} else if !exists {
			return text, nil
		}
	}
	if text, exists, err := migrationText(filepath.Join(root, "practice", "output.md")); err != nil || exists {
		return text, err
	}
	raw, exists, err := migrationText(filepath.Join(root, "practice", "tasks.json"))
	if err != nil || !exists {
		return "", err
	}
	var tasks struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(raw), &tasks); err != nil {
		return "", err
	}
	answers := map[string]any{}
	if key, exists, err := migrationText(filepath.Join(root, "practice", "answer-key.json")); err != nil {
		return "", err
	} else if exists {
		if err := json.Unmarshal([]byte(key), &answers); err != nil {
			return "", err
		}
	}
	var out strings.Builder
	out.WriteString("# 练习题\n\n")
	for i, task := range tasks.Tasks {
		fmt.Fprintf(&out, "## 练习 %d\n\n", i+1)
		writePracticeValue(&out, task)
		id, _ := task["id"].(string)
		if collection, ok := answers["answers"].(map[string]any); ok {
			if answer, ok := collection[id]; ok {
				out.WriteString("\n### 参考答案与解析\n\n")
				writePracticeValue(&out, answer)
			}
		}
	}
	// Preserve any answer schema beyond the indexed answers instead of dropping it.
	if len(answers) > 0 {
		out.WriteString("\n## 答案资料\n\n")
		writePracticeValue(&out, answers)
	}
	return out.String(), nil
}

func legacyMigrationVersion(dir string) (bool, error) {
	meta, err := readMeta(filepath.Join(dir, "asset.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var version Version
	if err := readJSON(filepath.Join(dir, "versions", meta.CurrentVersionID, "version.json"), &version); err != nil {
		return false, err
	}
	return version.Author == "migration", nil
}

func writePracticeValue(out *strings.Builder, value any) {
	labels := map[string]string{"question": "题目", "prompt": "题目", "options": "选项", "answer": "参考答案", "correctAnswer": "参考答案", "explanation": "解析", "hint": "提示", "type": "题型", "content": "内容", "id": "编号", "answers": "答案"}
	switch v := value.(type) {
	case string:
		out.WriteString(v)
		out.WriteString("\n\n")
	case []any:
		for i, item := range v {
			fmt.Fprintf(out, "%d. ", i+1)
			writePracticeValue(out, item)
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			label := labels[key]
			if label == "" {
				label = key
			}
			out.WriteString("**" + label + "**\n\n")
			writePracticeValue(out, v[key])
		}
	default:
		fmt.Fprintf(out, "%v\n\n", v)
	}
}
