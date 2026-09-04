package teacherservice

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	askaiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
	conversationstore "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/conversation"
)

func TestSoftScopeAppendixIsGuidance(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("mechanics", "经典力学", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	projectRoot, _ := workspace.ProjectRootForSlug("mechanics")
	scope := `{"schemaVersion":1,"status":"ready","title":"经典力学","chapterTitle":"力与运动","goal":"理解牛顿定律","teachingOutline":"从受力、运动状态与牛顿定律之间的因果关系建立模型，不展开具体仿真软件。","inScope":["受力分析"],"outOfScope":[],"prerequisites":[],"ownedConcepts":["惯性"],"reusedConcepts":[],"source":{"type":"discipline-map","mapSlug":"physics","topicId":"mechanics"},"updatedAt":"2026-08-31T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(projectRoot, "learning-scope.json"), []byte(scope), 0o644); err != nil {
		t.Fatal(err)
	}
	appendix := softScopeAppendix("mechanics")
	for _, want := range []string{"经典力学", "力与运动", "理解牛顿定律", "从受力、运动状态与牛顿定律之间的因果关系建立模型", "不是拒答或拆分对话的硬边界"} {
		if !strings.Contains(appendix, want) {
			t.Fatalf("scope appendix missing %q: %s", want, appendix)
		}
	}
}

func TestSelectedReadySourceContributesCanonicalTextToTeacherContext(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "闭包", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	sources, err := sourcestore.New("topic")
	if err != nil {
		t.Fatal(err)
	}
	source, revision, err := sources.Add("闭包讲义", "notes.txt", "text/plain", 5, bytes.NewBufferString("notes"), true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sources.CommitDerived(source.SourceID, revision.RevisionID, map[string][]byte{"content.md": []byte("闭包捕获定义时的词法环境。")}, map[string]string{"content.md": "text/markdown"}); err != nil {
		t.Fatal(err)
	}
	if _, err := sources.SetStatus(source.SourceID, "ready", "", "task_parse"); err != nil {
		t.Fatal(err)
	}
	messages := toProviderMessages("topic", []conversationstore.SequencedMessage{{Message: conversationstore.Message{Role: "learner", Blocks: []conversationstore.Block{{Type: "attachment", ArtifactRef: source.SourceID}}}}})
	if len(messages) != 1 || !strings.Contains(messages[0].Content, "闭包捕获定义时的词法环境") {
		t.Fatalf("selected source text was not cited: %#v", messages)
	}
}

func TestContextBudgetsRespectSmallerProviderWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	restore := askaiconfig.UseConfigPathForTest(path)
	defer restore()
	config := `{"askAiProviders":{"default":"small","providers":[{"id":"small","kind":"openai","baseURL":"http://example.invalid","apiKey":"x","model":"small","contextWindowTokens":32768}],"bindings":{"teacher":{"providerId":"small","model":"small"}}}}`
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	threshold, recent := contextBudgets()
	if threshold != 24576 || recent != 12288 {
		t.Fatalf("unexpected budgets threshold=%d recent=%d", threshold, recent)
	}
}
