package promptassembly

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestBuild_HappyPath(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	// Create a project on disk.
	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	// Write a predecessor file so Explain has Intro context.
	if err := workspace.SafeWriteArtifact("test", workspace.ZoneIntro, "output.md", []byte("# Intro\n\nCuriosity hook.")); err != nil {
		t.Fatal(err)
	}

	// Build a fake registry with explain agent.
	reg := agentregistry.New()
	writeTestAgent(t, reg, "explain", []workspace.ZoneName{workspace.ZoneExplain})

	// Build prompt.
	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneExplain,
		AgentID:     "explain",
		Intent:      "Explain Rust ownership in 200 words.",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if pkg.PromptMd == "" {
		t.Fatal("PromptMd empty")
	}
	// Must contain the charter, intent, user story section, and predecessor.
	checks := []string{
		"# Agent Identity",
		"# User Story",
		"# Charter",
		"Explain Rust ownership in 200 words.",
		"intro",
		"output.md",
	}
	for _, c := range checks {
		if !strings.Contains(pkg.PromptMd, c) {
			t.Errorf("prompt missing %q\n--- prompt ---\n%s", c, pkg.PromptMd)
		}
	}
	// Predecessor must be marked as existing (✓).
	if !strings.Contains(pkg.PromptMd, "✓") {
		t.Errorf("expected predecessor to be marked exists (✓) in prompt:\n%s", pkg.PromptMd)
	}
	// Run dir must contain agent name.
	if !strings.Contains(pkg.RunDirName, "-explain") {
		t.Errorf("RunDirName = %q, want suffix -explain", pkg.RunDirName)
	}
	// Package meta JSON serializes.
	js, err := pkg.MarshalPackageMeta()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"agentId": "explain"`) {
		t.Errorf("package meta missing agentId: %s", js)
	}
}

func TestBuild_RejectsUnknownAgent(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	_, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneExplain,
		AgentID:     "nonexistent",
		Intent:      "hi",
	}, reg)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestBuild_RejectsZoneMismatch(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "explain", []workspace.ZoneName{workspace.ZoneExplain})

	// Explain agent is not allowed in Practice zone.
	_, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZonePractice,
		AgentID:     "explain",
		Intent:      "drill",
	}, reg)
	if err == nil {
		t.Fatal("expected zone-mismatch error")
	}
}

func TestBuild_AllowsEmptyIntent(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "intro", []workspace.ZoneName{workspace.ZoneIntro})

	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneIntro,
		AgentID:     "intro",
		Intent:      "",
	}, reg)
	if err != nil {
		t.Fatalf("Build should allow empty intent: %v", err)
	}
	if strings.Contains(pkg.PromptMd, "# Additional Guidance") {
		t.Fatalf("prompt should omit optional guidance section when intent is empty:\n%s", pkg.PromptMd)
	}
}

func TestBuild_IncludesProjectCreationFields(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	input := workspace.ProjectInput{
		Why:      "为了通过编译原理考试",
		Current:  "了解正则表达式",
		Target:   "能独立完成词法分析",
		Standard: "能把正则表达式转换为有限自动机",
	}
	if err := workspace.CreateProjectSkeletonWithInput("lexer", "词法分析", "", input); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "intro", []workspace.ZoneName{workspace.ZoneIntro})

	pkg, err := Build(Request{
		ProjectSlug: "lexer",
		ZoneName:    workspace.ZoneIntro,
		AgentID:     "intro",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	checks := []string{
		"# Project Brief",
		"为了通过编译原理考试",
		"了解正则表达式",
		"能独立完成词法分析",
		"能把正则表达式转换为有限自动机",
		"# Intro Calibration Boundary",
		"do not ask the learner to repeat them",
	}
	for _, check := range checks {
		if !strings.Contains(pkg.PromptMd, check) {
			t.Errorf("prompt missing project context %q\n--- prompt ---\n%s", check, pkg.PromptMd)
		}
	}
	if pkg.PackageMeta.ProjectFile == "" || !strings.HasSuffix(pkg.PackageMeta.ProjectFile, "project.md") {
		t.Errorf("ProjectFile = %q, want project.md path", pkg.PackageMeta.ProjectFile)
	}
}

func TestBuild_AllowsLegacyProjectWithoutProjectBrief(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("legacy", "Legacy", ""); err != nil {
		t.Fatal(err)
	}
	projectRoot, err := workspace.ProjectRootForSlug("legacy")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(projectRoot, "project.md")); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "intro", []workspace.ZoneName{workspace.ZoneIntro})

	pkg, err := Build(Request{
		ProjectSlug: "legacy",
		ZoneName:    workspace.ZoneIntro,
		AgentID:     "intro",
	}, reg)
	if err != nil {
		t.Fatalf("Build should allow a legacy project without project.md: %v", err)
	}
	if !strings.Contains(pkg.PromptMd, "project.md is not present") {
		t.Errorf("missing legacy project fallback:\n%s", pkg.PromptMd)
	}
}

func TestBuild_ProductionIntroPromptDoesNotRepeatProjectCreationInterview(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeletonWithInput("test", "Test", "", workspace.ProjectInput{
		Why:      "工作需要",
		Current:  "了解概念",
		Target:   "能独立应用",
		Standard: "完成一个可运行案例",
	}); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("load production registry: %v", err)
	}

	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneIntro,
		AgentID:     "intro",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	required := []string{
		"创建项目时填写的学习动机、当前水平、目标水平和完成标准均视为已回答，不得重复询问",
		"只覆盖该主题的术语识别、因果理解、前置知识和简单应用",
		"不重复询问 `Project Brief` 中已有的项目创建字段",
	}
	for _, fragment := range required {
		if !strings.Contains(pkg.PromptMd, fragment) {
			t.Errorf("production intro prompt missing contract %q", fragment)
		}
	}
	if strings.Contains(pkg.PromptMd, "覆盖术语识别、因果理解、简单应用和学习目标") {
		t.Errorf("production intro prompt still asks for the already-known learning goal")
	}
}

func TestBuild_AllProductionAgentsForbidTextCharacterDiagrams(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("load production registry: %v", err)
	}

	const prohibition = "Never draw diagrams with ASCII or Unicode text characters"
	const replacement = "Use a fenced Mermaid block for every diagram."
	for _, agent := range reg.List() {
		if len(agent.AllowedZones) == 0 {
			t.Fatalf("agent %s has no allowed zones", agent.ID)
		}
		pkg, err := Build(Request{
			ProjectSlug: "test",
			ZoneName:    agent.AllowedZones[0],
			AgentID:     agent.ID,
		}, reg)
		if err != nil {
			t.Fatalf("Build agent %s: %v", agent.ID, err)
		}
		if !strings.Contains(pkg.PromptMd, prohibition) {
			t.Errorf("agent %s prompt missing text-diagram prohibition", agent.ID)
		}
		if !strings.Contains(pkg.PromptMd, replacement) {
			t.Errorf("agent %s prompt missing Mermaid replacement rule", agent.ID)
		}
	}
}

func TestBuild_PracticeEvaluationUsesAttemptSpecificContract(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "practice", []workspace.ZoneName{workspace.ZonePractice})

	pkg, err := Build(Request{
		ProjectSlug:     "test",
		ZoneName:        workspace.ZonePractice,
		AgentID:         "practice",
		PracticeAttempt: 3,
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	required := []string{
		"# Practice Evaluation Artifact Contract",
		"practice/attempts/3.json",
		"practice/submissions/3.json",
		"practice/evaluations/3.json",
		"`suggestedAnswer`",
		"`summary`",
		"Do not generate or overwrite `practice/tasks.json`",
	}
	for _, fragment := range required {
		if !strings.Contains(pkg.PromptMd, fragment) {
			t.Errorf("evaluation prompt missing %q\n%s", fragment, pkg.PromptMd)
		}
	}
	if pkg.PackageMeta.PracticeAttempt != 3 {
		t.Errorf("PracticeAttempt = %d, want 3", pkg.PackageMeta.PracticeAttempt)
	}
	if len(pkg.PackageMeta.OutputTargets) != 2 ||
		pkg.PackageMeta.OutputTargets[0].Filename != "evaluations/3.json" {
		t.Errorf("unexpected evaluation output targets: %#v", pkg.PackageMeta.OutputTargets)
	}
}

func TestBuild_ProductionSummaryPromptRequiresStructuredFlashcards(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneSummary,
		AgentID:     "summary",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, fragment := range []string{
		"summary/flashcards.json",
		"前端闪卡的唯一事实源",
		"卡片必须贴近概念和核心理解",
		`"category"`,
		`"sourceRefs"`,
	} {
		if !strings.Contains(pkg.PromptMd, fragment) {
			t.Errorf("summary prompt missing %q", fragment)
		}
	}
	if len(pkg.PackageMeta.OutputTargets) != 2 {
		t.Fatalf("summary output targets = %#v", pkg.PackageMeta.OutputTargets)
	}
}

func TestBuild_PracticeGenerationUsesExactQuestionCount(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	pkg, err := Build(Request{
		ProjectSlug:           "test",
		ZoneName:              workspace.ZonePractice,
		AgentID:               "practice",
		PracticeQuestionCount: 7,
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(pkg.PromptMd, "Generate exactly 7 questions") {
		t.Fatalf("prompt missing exact question count:\n%s", pkg.PromptMd)
	}
	if pkg.PackageMeta.PracticeQuestionCount != 7 {
		t.Fatalf("package question count = %d", pkg.PackageMeta.PracticeQuestionCount)
	}
}

// writeTestAgent adds an in-memory agent to the registry by writing to a
// temp dir and reloading.
func writeTestAgent(t *testing.T, reg *agentregistry.Registry, id string, zones []workspace.ZoneName) {
	t.Helper()
	dir := t.TempDir()
	regDir := filepath.Join(dir, "registry")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chPath := filepath.Join(dir, "charters", id+".md")
	if err := os.MkdirAll(filepath.Dir(chPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chPath, []byte("# Test charter"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{
  "id": "` + id + `",
  "name": "` + id + `",
  "userStory": "test user story for ` + id + `",
  "allowedZones": ["` + string(zones[0]) + `"],
  "charterPath": "` + strings.ReplaceAll(chPath, "\\", "\\\\") + `",
  "defaultOutputTargets": [{"zone": "` + string(zones[0]) + `", "filename": "output.md"}]
}`)
	if err := os.WriteFile(filepath.Join(regDir, id+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	defer agentregistry.SetAgentsRootForTest(old)
	if err := reg.Load(); err != nil {
		t.Fatal(err)
	}
}

// writeTestAgentWithPrimitives is like writeTestAgent but also declares
// reasoning primitives and writes the primitive files under the same
// agents root. Returns a cleanup func the caller must defer — agents root
// must stay overridden until AFTER Build has read the primitives from disk.
func writeTestAgentWithPrimitives(
	t *testing.T,
	reg *agentregistry.Registry,
	id string,
	zones []workspace.ZoneName,
	required, optional []string,
) func() {
	t.Helper()
	dir := t.TempDir()
	regDir := filepath.Join(dir, "registry")
	_ = os.MkdirAll(regDir, 0o755)
	primDir := filepath.Join(dir, "primitives")
	_ = os.MkdirAll(primDir, 0o755)
	chPath := filepath.Join(dir, "charters", id+".md")
	_ = os.MkdirAll(filepath.Dir(chPath), 0o755)
	_ = os.WriteFile(chPath, []byte("# Test charter with primitives"), 0o644)
	for _, name := range append(append([]string{}, required...), optional...) {
		_ = os.WriteFile(filepath.Join(primDir, name+".md"), []byte("BODY:"+name), 0o644)
	}
	reqJSON := "["
	for i, n := range required {
		if i > 0 {
			reqJSON += ","
		}
		reqJSON += `"` + n + `"`
	}
	reqJSON += "]"
	optJSON := "["
	for i, n := range optional {
		if i > 0 {
			optJSON += ","
		}
		optJSON += `"` + n + `"`
	}
	optJSON += "]"
	data := []byte(`{
  "id": "` + id + `",
  "name": "` + id + `",
  "userStory": "test user story for ` + id + `",
  "primitives": {"required": ` + reqJSON + `, "optional": ` + optJSON + `},
  "allowedZones": ["` + string(zones[0]) + `"],
  "charterPath": "` + strings.ReplaceAll(chPath, "\\", "\\\\") + `",
  "defaultOutputTargets": [{"zone": "` + string(zones[0]) + `", "filename": "output.md"}]
}`)
	if err := os.WriteFile(filepath.Join(regDir, id+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	old := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	if err := reg.Load(); err != nil {
		t.Fatal(err)
	}
	return func() { agentregistry.SetAgentsRootForTest(old) }
}

func TestBuild_PrimitivesSectionIncluded(t *testing.T) {
	ClearPrimitiveCacheForTest()
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	cleanup := writeTestAgentWithPrimitives(t, reg, "explain",
		[]workspace.ZoneName{workspace.ZoneExplain},
		[]string{"mece_decompose", "first_principles"},
		[]string{"analogy"})
	defer cleanup()
	ClearPrimitiveCacheForTest()

	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneExplain,
		AgentID:     "explain",
		Intent:      "Explain Rust ownership.",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	checks := []string{
		"# User Story",
		"# Reasoning Primitives",
		"### Required primitives",
		"### Optional primitive references",
		"BODY:mece_decompose",
		"BODY:first_principles",
		"BODY:analogy",
	}
	for _, c := range checks {
		if !strings.Contains(pkg.PromptMd, c) {
			t.Errorf("prompt missing %q\n--- prompt tail ---\n%s", c, tail(pkg.PromptMd, 2000))
		}
	}
}

func TestBuild_ExplainIncludesTutorialArtifactContract(t *testing.T) {
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	writeTestAgent(t, reg, "explain", []workspace.ZoneName{workspace.ZoneExplain})

	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneExplain,
		AgentID:     "explain",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	checks := []string{
		"# Explain Tutorial Artifact Contract",
		"可独立阅读的教程",
		"不使用“你”“我们”等对话人称",
		"第一性原理不得成为独立页面、章节、标题或逐步推导",
		"只能融入最后一页的“核心观点”",
		"不得让每一页机械重复同一组栏目",
		"Silently use predecessor files",
	}
	for _, check := range checks {
		if !strings.Contains(pkg.PromptMd, check) {
			t.Errorf("explain prompt missing tutorial contract %q\n--- prompt ---\n%s", check, pkg.PromptMd)
		}
	}
	if strings.Contains(pkg.PromptMd, "Cite predecessor files when building on prior zone output.") {
		t.Errorf("explain prompt must not require predecessor citations:\n%s", pkg.PromptMd)
	}
}

func TestBuild_ProductionExplainPromptHasNoLegacyFirstPrinciplesContract(t *testing.T) {
	ClearPrimitiveCacheForTest()
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(oldWS)

	if err := workspace.CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("load production registry: %v", err)
	}
	pkg, err := Build(Request{
		ProjectSlug: "test",
		ZoneName:    workspace.ZoneExplain,
		AgentID:     "explain",
	}, reg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	forbidden := []string{
		`**必选** 讲解智能体的"第一性原理"章节`,
		"5-8 步推理链",
		"每步必须显式",
	}
	for _, fragment := range forbidden {
		if strings.Contains(pkg.PromptMd, fragment) {
			t.Errorf("production explain prompt still contains legacy first-principles contract %q", fragment)
		}
	}
	required := []string{
		"第一性原理只能影响最终观点的压缩方式",
		"不得出现独立的第一性原理标题、页面或逐步推导",
		"never cite their filenames or narrate learner-profile evidence",
	}
	for _, fragment := range required {
		if !strings.Contains(pkg.PromptMd, fragment) {
			t.Errorf("production explain prompt missing contract %q", fragment)
		}
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
