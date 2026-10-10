package assistanttask

import (
	"strings"
	"testing"
)

func TestConsolidationPromptRequiresTeachingManuscriptNotTranscript(t *testing.T) {
	prompt := buildTaskPrompt(Task{ID: "task_test", Type: "consolidate", Objective: "沉淀本轮教学稿", PracticeRequested: false}, "run_test")
	for _, want := range []string{"assets/review", "为什么值得学习", "独立教学稿", "批判性思维", "不是对话逐字稿", "practice 必须 unchanged", "不得创建助教成果", "不得写入教师笔记"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("consolidation prompt missing %q: %s", want, prompt)
		}
	}
}
