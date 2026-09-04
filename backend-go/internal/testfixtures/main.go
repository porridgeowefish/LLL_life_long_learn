// Command generate writes the synthetic iteration-14 test fixtures.
//
// Fixtures are generated THROUGH the production store code so their on-disk
// schemas cannot drift from what the stores actually read and write. The
// generator only ever writes below tests/fixtures/<family>/<slug>; it never
// touches projects/. All content is synthetic — no learner data is copied.
//
// Run: go run ./tests/fixtures/generate
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/assistanttask"
	"github.com/xmz14/lll/backend-go/internal/conversationstore"
	assetstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func main() {
	root := "tests/fixtures"
	if _, err := os.Stat(filepath.Join(root, "canonical")); err == nil {
		fmt.Println("fixtures already exist; delete tests/fixtures first to regenerate")
		return
	}

	// Redirect the workspace roots to the fixtures tree for generation.
	if err := os.MkdirAll(filepath.Join(root, "canonical"), 0o755); err != nil {
		fatal(err)
	}
	projectsRoot, err := filepath.Abs(filepath.Join(root, "canonical"))
	if err != nil {
		fatal(err)
	}
	workspace.SetProjectsRootForTest(projectsRoot)

	// --- canonical/iteration-13: one complete synthetic learning unit ---
	if err := workspace.CreateProjectSkeletonWithInput(
		"lingo-duihua", "合成对话单元", "", workspace.ProjectInput{
			ProjectType: workspace.ProjectTypeSystemLearning,
			Why:         "合成数据",
			Current:     "无基础",
			Target:      "理解基本概念",
			Standard:    "能复述定义",
		}); err != nil {
		fatal(err)
	}

	conv, err := conversationstore.New("lingo-duihua")
	if err != nil {
		fatal(err)
	}
	if _, _, err := conv.AppendMessage("teacher", "completed", "teacher-welcome-v1", []conversationstore.Block{{Type: "markdown", Source: "你好，我是合成教师。"}}); err != nil {
		fatal(err)
	}
	if _, _, err := conv.AppendMessage("learner", "completed", "op_fixture_1", []conversationstore.Block{{Type: "markdown", Source: "什么是闭包？"}}); err != nil {
		fatal(err)
	}
	if _, _, err := conv.AppendMessage("teacher", "completed", "op_fixture_2", []conversationstore.Block{{Type: "markdown", Source: "闭包是捕获了环境的函数。"}}); err != nil {
		fatal(err)
	}

	assets, err := assetstore.New("lingo-duihua")
	if err != nil {
		fatal(err)
	}
	if _, err := assets.UpdateLearner("intro", 1, "# 引言（合成）\n\n这是合成 intro 资产。"); err != nil {
		fatal(err)
	}
	if _, err := assets.UpdateLearner("body", 1, "# 正文（合成）\n\n## 闭包定义\n\n闭包 = 函数 + 环境。"); err != nil {
		fatal(err)
	}

	annotations, err := assetstore.NewAnnotations("lingo-duihua")
	if err != nil {
		fatal(err)
	}
	if _, err := annotations.Create(assetstore.CreateInput{
		QuoteSnapshot: "闭包是捕获了环境的函数",
		Anchors:       assetstore.Anchors{Start: 12, End: 25},
		Note:          "合成批注",
	}); err != nil {
		fatal(err)
	}

	sources, err := sourcestore.New("lingo-duihua")
	if err != nil {
		fatal(err)
	}
	if _, _, err := sources.Add("合成资料", "notes.txt", "text/plain", int64(len("合成资料正文内容")), strings.NewReader("合成资料正文内容"), false); err != nil {
		fatal(err)
	}

	tasks, err := assistanttask.New("lingo-duihua")
	if err != nil {
		fatal(err)
	}
	if _, _, err := tasks.Create(assistanttask.CreateInput{
		Type:                  "consolidate",
		Objective:             "整理合成对话要点",
		Origin:                assistanttask.Origin{Kind: "fixture", OperationID: "op_fixture_task"},
		ConversationCutoffSeq: 3,
	}); err != nil {
		fatal(err)
	}

	// --- legacy/zones: one pre-iteration-13 five-zone project ---
	legacyRoot := filepath.Join(root, "legacy", "zones")
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		fatal(err)
	}
	legacyAbs, err := filepath.Abs(legacyRoot)
	if err != nil {
		fatal(err)
	}
	workspace.SetProjectsRootForTest(legacyAbs)
	if err := workspace.CreateProjectSkeleton("legacy-wuqu", "合成旧版五区项目", ""); err != nil {
		fatal(err)
	}
	unit, err := workspace.ReadProjectState("legacy-wuqu")
	if err != nil {
		fatal(err)
	}
	_ = unit
	writeLegacy := func(rel, content string) {
		path := filepath.Join(legacyRoot, "legacy-wuqu", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fatal(err)
		}
	}
	writeLegacy("intro/output.md", "# 合成 Intro 产物\n\n迭代 13 之前的 Intro 输出。")
	writeLegacy("intro/assessment.json", `{"schemaVersion":1,"level":"basic","notes":"合成评估"}`)
	writeLegacy("explain/manifest.json", `{"pages":[{"file":"01.md","title":"第一页"}]}`)
	writeLegacy("explain/pages/01.md", "# 合成 Explain 页\n\n旧版讲解内容。")
	writeLegacy("explain/confusions.json", `[{"id":"cf_fix1","quoteSnapshot":"捕获了环境","charStart":3,"charEnd":9,"notes":"看不懂","state":"open","createdAt":"2026-01-01T00:00:00Z"}]`)
	writeLegacy("practice/tasks.json", `{"schemaVersion":1,"setId":"set_fix","generatedAt":"2026-01-01T00:00:00Z","tasks":[{"id":"t1","type":"short-answer","prompt":"用自己的话解释闭包","difficulty":3}]}`)
	writeLegacy("practice/answer-key.json", `{"answers":{"t1":"函数加环境"}}`)
	writeLegacy("extend/prompts.md", "# 合成 Extend 提示\n\n拓展方向。")
	writeLegacy("summary/summary.md", "# 合成总结\n\n五个区块的旧版总结。")
	writeLegacy("summary/flashcards.json", `{"cards":[{"id":"fc1","front":"闭包是什么","back":"函数+环境"}]}`)
	writeLegacy("progress/events.jsonl", `{"id":"evt1","sourceType":"practice","sourceId":"t1","difficulty":3,"outcome":"correct","delta":1,"createdAt":"2026-01-01T00:00:00Z"}`+"\n")

	// Import the legacy confusions into the canonical annotation log now, so
	// the fixture ships in its post-import steady state and later reads are
	// read-only (the first read in production performs this import once).
	legacyAnnotations, err := assetstore.NewAnnotations("legacy-wuqu")
	if err != nil {
		fatal(err)
	}
	if err := legacyAnnotations.ImportLegacy(); err != nil {
		fatal(err)
	}

	// --- legacy/memory-era: pre-retirement project memory files ---
	memoryRoot := filepath.Join(root, "legacy", "memory-era")
	if err := os.MkdirAll(filepath.Join(memoryRoot, "memory-project", "memory"), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memoryRoot, "memory-project", "memory", "project-memory.md"), []byte("# 合成项目记忆\n\n记忆时代的遗留文件，仅供兼容读取。"), 0o644); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memoryRoot, "memory-project", "memory", "project-state.json"), []byte(`{"schemaVersion":1,"items":[]}`), 0o644); err != nil {
		fatal(err)
	}

	// --- corrupt: a system-learning project with damaged canonical files ---
	corruptRoot := filepath.Join(root, "corrupt")
	if err := os.MkdirAll(corruptRoot, 0o755); err != nil {
		fatal(err)
	}
	corruptAbs, err := filepath.Abs(corruptRoot)
	if err != nil {
		fatal(err)
	}
	workspace.SetProjectsRootForTest(corruptAbs)
	if err := workspace.CreateProjectSkeletonWithInput("sunhuai-xiangmu", "损坏示例", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		fatal(err)
	}
	conv2, err := conversationstore.New("sunhuai-xiangmu")
	if err != nil {
		fatal(err)
	}
	if _, _, err := conv2.AppendMessage("learner", "completed", "op_corrupt", []conversationstore.Block{{Type: "markdown", Source: "半截对话"}}); err != nil {
		fatal(err)
	}
	// Truncate the events log mid-record so readers must exercise recovery paths.
	eventsPath := filepath.Join(corruptRoot, "sunhuai-xiangmu", "conversation", "events.jsonl")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		fatal(err)
	}
	if len(data) > 40 {
		if err := os.WriteFile(eventsPath, data[:len(data)-20], 0o644); err != nil {
			fatal(err)
		}
	}

	fmt.Println("fixtures written below tests/fixtures/")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "generate:", err)
	os.Exit(1)
}
