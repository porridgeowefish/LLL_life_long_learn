package assistanttask

import (
	"strings"
	"testing"
)

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
