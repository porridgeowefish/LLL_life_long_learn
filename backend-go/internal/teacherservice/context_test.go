package teacherservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestSoftScopeAppendixIsGuidance(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("mechanics", "经典力学", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	projectRoot, _ := workspace.ProjectRootForSlug("mechanics")
	scope := `{"schemaVersion":1,"status":"ready","title":"经典力学","chapterTitle":"力与运动","goal":"理解牛顿定律","inScope":["受力分析"],"outOfScope":[],"prerequisites":[],"ownedConcepts":["惯性"],"reusedConcepts":[],"source":{"type":"discipline-map","mapSlug":"physics","topicId":"mechanics"},"updatedAt":"2026-08-31T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(projectRoot, "learning-scope.json"), []byte(scope), 0o644); err != nil {
		t.Fatal(err)
	}
	appendix := softScopeAppendix("mechanics")
	for _, want := range []string{"经典力学", "力与运动", "理解牛顿定律", "不是拒答或拆分对话的硬边界"} {
		if !strings.Contains(appendix, want) {
			t.Fatalf("scope appendix missing %q: %s", want, appendix)
		}
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
