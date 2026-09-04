package assistanttask

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSourceProcessingRequiresOneCanonicalMarkdownOutput(t *testing.T) {
	workDir := t.TempDir()
	revisionID := "srev_test"
	if err := os.MkdirAll(filepath.Join(workDir, "source-updates", revisionID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "source-updates", revisionID, "notes.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := inputManifest{}
	manifest.Sources = append(manifest.Sources, struct{ SourceID, RevisionID, Path, SHA256 string }{SourceID: "source_test", RevisionID: revisionID})
	result := resultManifest{
		AssetUpdates: map[string]struct {
			Status    string `json:"status"`
			Candidate string `json:"candidate"`
			Code      string `json:"code"`
		}{"intro": {Status: "unchanged"}, "body": {Status: "unchanged"}, "practice": {Status: "unchanged"}},
		SourceUpdate: &struct {
			SourceID   string `json:"sourceId"`
			RevisionID string `json:"revisionId"`
			Files      []struct {
				Key       string `json:"key"`
				Path      string `json:"path"`
				MediaType string `json:"mediaType"`
			} `json:"files"`
		}{SourceID: "source_test", RevisionID: revisionID, Files: []struct {
			Key       string `json:"key"`
			Path      string `json:"path"`
			MediaType string `json:"mediaType"`
		}{{Key: "notes", Path: "source-updates/" + revisionID + "/notes.md", MediaType: "text/markdown"}}},
	}

	if err := validateResult(workDir, Task{Type: "source-processing"}, manifest, result); err == nil {
		t.Fatal("non-canonical source output must be rejected")
	}
}

func TestValidateConsolidationRequiresIntroAndBodyUpdates(t *testing.T) {
	result := resultManifest{AssetUpdates: map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	}{"intro": {Status: "unchanged"}, "body": {Status: "unchanged"}, "practice": {Status: "unchanged"}}}
	if err := validateResult(t.TempDir(), Task{Type: "consolidate"}, inputManifest{}, result); err == nil {
		t.Fatal("consolidation must require intro and body updates")
	}
}

func TestConsolidationPromptRequiresTeachingManuscriptNotTranscript(t *testing.T) {
	prompt := buildTaskPrompt(Task{ID: "task_test", Type: "consolidate", Objective: "沉淀本轮教学稿", PracticeRequested: false}, "run_test")
	for _, want := range []string{"assets/intro", "为什么值得学习", "独立教学稿", "批判性思维", "不是对话逐字稿", "practice 必须 unchanged"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("consolidation prompt missing %q: %s", want, prompt)
		}
	}
}

func TestSourceProcessingPromptRequiresCanonicalContentMarkdown(t *testing.T) {
	prompt := buildTaskPrompt(Task{ID: "task_test", Type: "source-processing", Objective: "解析资料"}, "run_test")
	for _, want := range []string{"content.md", "text/markdown", "不得生成多份派生资料"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("source processing prompt missing %q: %s", want, prompt)
		}
	}
}
