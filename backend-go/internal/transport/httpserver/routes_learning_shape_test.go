package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	folderstore "github.com/xmz14/lll/backend-go/internal/modules/projects"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

func setupLearningShapeTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	oldRoot := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	restoreCfg := teacher.UseConfigPathForTest(filepath.Join(dir, "config.local.json"))
	oldComplete := completeLearningShapeAI
	writeAskAiConfig(t, teacher.ProviderConfig{ID: "stub", Kind: "openai", BaseURL: "http://stub", APIKey: "k", Model: "m"})
	t.Cleanup(func() {
		workspace.SetProjectsRootForTest(oldRoot)
		restoreCfg()
		completeLearningShapeAI = oldComplete
	})
	return dir
}

func TestProjectTypeAdviceHasNoCreationSideEffect(t *testing.T) {
	dir := setupLearningShapeTest(t)
	completeLearningShapeAI = func(context.Context, teacher.Provider, string, []teacher.AIMessage) (string, error) {
		return `{"reply":"建议先建立学科地图。","recommendation":"discipline-map","reason":"先建立领域方向感","tradeoff":"系统学习会更快进入细节","confidence":"high"}`, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/project-type-advice", strings.NewReader(`{"title":"博弈论","current":"未接触","target":"看懂原理","messages":[{"role":"user","content":"我想先知道有哪些研究领域"}]}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleProjectTypeAdvice(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "discipline-map") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 { // config.local.json only
		t.Fatalf("advice created project state: entries=%v err=%v", entries, err)
	}
}

func TestProjectTypeAdviceCanAskClarifyingQuestion(t *testing.T) {
	setupLearningShapeTest(t)
	completeLearningShapeAI = func(context.Context, teacher.Provider, string, []teacher.AIMessage) (string, error) {
		return `{"reply":"你更想先看全貌，还是直接掌握一个具体问题？","recommendation":"undetermined","reason":"","tradeoff":"","confidence":"low"}`, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/project-type-advice", strings.NewReader(`{"current":"未接触","target":"看懂原理","messages":[{"role":"user","content":"我想学数学"}]}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleProjectTypeAdvice(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "更想先看全貌") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "recommendation") {
		t.Fatalf("undetermined response should omit recommendation: %s", rec.Body.String())
	}
}

func TestProjectTypeAdviceTreatsMissingDraftFieldsAsUnknown(t *testing.T) {
	setupLearningShapeTest(t)
	var capturedSystem, capturedInput string
	completeLearningShapeAI = func(_ context.Context, _ teacher.Provider, system string, messages []teacher.AIMessage) (string, error) {
		capturedSystem = system
		capturedInput = messages[0].Content
		return `{"reply":"你更想先看全貌，还是直接掌握一个具体问题？","recommendation":"undetermined","reason":"","tradeoff":"","confidence":"low"}`, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/project-type-advice", strings.NewReader(`{"title":"博弈论","messages":[{"role":"user","content":"帮我选"}]}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleProjectTypeAdvice(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(capturedInput, `"current"`) || strings.Contains(capturedInput, `"standard"`) {
		t.Fatalf("missing learner fields leaked into advisor draft: %s", capturedInput)
	}
	if !strings.Contains(capturedSystem, "缺失字段就是未知") {
		t.Fatalf("advisor system prompt lacks missing-data guard: %s", capturedSystem)
	}
}

func TestProjectTypeAdviceFallsBackToNaturalLanguage(t *testing.T) {
	setupLearningShapeTest(t)
	completeLearningShapeAI = func(context.Context, teacher.Provider, string, []teacher.AIMessage) (string, error) {
		return "建议先建立学科地图，因为你目前需要的是研究领域全貌。", nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/project-type-advice", strings.NewReader(`{"current":"未接触","target":"看懂原理","messages":[{"role":"user","content":"我想先看全貌"}]}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleProjectTypeAdvice(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "建议先建立学科地图") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProjectTypeAdviceRecoversMalformedJSONQuotes(t *testing.T) {
	setupLearningShapeTest(t)
	completeLearningShapeAI = func(context.Context, teacher.Provider, string, []teacher.AIMessage) (string, error) {
		return `{"reply":"你需要"先看清全貌"，适合建立地图。","recommendation":"discipline-map","reason":"需要领域关系","tradeoff":"不会立即深入练习","confidence":"high"}`, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/project-type-advice", strings.NewReader(`{"current":"未接触","target":"看懂原理","messages":[{"role":"user","content":"我想先看全貌"}]}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleProjectTypeAdvice(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "先看清全貌") || !strings.Contains(rec.Body.String(), "discipline-map") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGenerateDisciplineOverviewLaunchesExplicitAgentCLI(t *testing.T) {
	setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeletonWithInput("physics", "物理学", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeDisciplineMap}); err != nil {
		t.Fatal(err)
	}
	launched := make(chan assistant.ProjectRequest, 1)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/physics/discipline-overview/generate", strings.NewReader(`{}`))
	req.SetPathValue("id", "physics")
	rec := httptest.NewRecorder()
	srv := newTestServer(t)
	srv.Runtime = assistant.Runtime{
		Definition: assistant.Definition{ID: assistant.RuntimeClaude},
		Bin:        "claude",
		Available:  true,
	}
	srv.agentExecution = recordingExecutionService{projects: launched}
	srv.handleGenerateDisciplineOverview(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var launchReq assistant.ProjectRequest
	select {
	case launchReq = <-launched:
	case <-time.After(time.Second):
		t.Fatal("agent CLI launch was not requested")
	}
	if launchReq.Agent.ID != "encyclopedia" || launchReq.ContextName != "学科总览" {
		t.Fatalf("unexpected launch scope: agent=%s context=%s", launchReq.Agent.ID, launchReq.ContextName)
	}
	for _, want := range []string{"学科百科智能体 Charter", "overview.md", "discipline-topics.json", "Project-Level Invocation Contract"} {
		if !strings.Contains(launchReq.PromptPackage.PromptMd, want) {
			t.Fatalf("prompt missing %q: %s", want, launchReq.PromptPackage.PromptMd)
		}
	}
	root, _ := workspace.ProjectRootForSlug("physics")
	data, _ := os.ReadFile(filepath.Join(root, "overview.md"))
	if !strings.Contains(string(data), "尚未生成") {
		t.Fatalf("HTTP route must not write model output directly: %s", data)
	}
}

type recordingExecutionService struct {
	projects chan assistant.ProjectRequest
}

func (r recordingExecutionService) StartProject(_ context.Context, req assistant.ProjectRequest) (*assistant.RunResult, error) {
	r.projects <- req
	return &assistant.RunResult{RunDirRel: filepath.Join("runs", req.PromptPackage.RunDirName)}, nil
}

func (recordingExecutionService) StartTask(context.Context, assistant.TaskRequest) error {
	return nil
}
func (recordingExecutionService) Resume(context.Context, assistant.ResumeRequest) (*assistant.RunResult, error) {
	return &assistant.RunResult{}, nil
}
func (recordingExecutionService) StartHeadless(context.Context, assistant.HeadlessRequest) error {
	return nil
}

func TestDisciplineTopicsAreReadableAndSnapshottedWhenDeepDiveIsCreated(t *testing.T) {
	dir := setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeletonWithInput("physics", "物理学", "", workspace.ProjectInput{
		ProjectType: workspace.ProjectTypeDisciplineMap,
	}); err != nil {
		t.Fatal(err)
	}
	catalog := `{
  "schemaVersion": 1,
  "topics": [{
    "id": "classical-mechanics",
    "title": "经典力学",
    "chapterTitle": "力与运动",
    "goal": "解释宏观低速物体的运动规律",
    "inScope": ["牛顿运动定律"],
    "outOfScope": ["热现象"],
    "prerequisites": ["向量"],
    "ownedConcepts": ["惯性参考系"],
    "reusedConcepts": ["微积分"]
  }],
  "updatedAt": "2026-07-28T00:00:00Z"
}`
	if err := os.WriteFile(filepath.Join(dir, "physics", "discipline-topics.json"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	overview := "# 物理学：学科总览\n\n## 主要研究领域与知识架构\n\n### 力与运动\n\n#### 经典力学\n"
	if err := os.WriteFile(filepath.Join(dir, "physics", "overview.md"), []byte(overview), 0o644); err != nil {
		t.Fatal(err)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/projects/physics/discipline-topics", nil)
	get.SetPathValue("id", "physics")
	getRec := httptest.NewRecorder()
	server := newTestServer(t)
	server.handleGetDisciplineTopics(getRec, get)
	if getRec.Code != http.StatusOK || !strings.Contains(getRec.Body.String(), `"id":"classical-mechanics"`) {
		t.Fatalf("topics status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	createBody := `{
  "title": "经典力学",
  "slug": "classical-mechanics",
  "projectType": "system-learning",
  "scopeSource": {
    "type": "discipline-map",
    "mapSlug": "physics",
    "topicId": "classical-mechanics"
  }
}`
	create := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(createBody))
	createRec := httptest.NewRecorder()
	server.handleCreateProject(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	scope, err := os.ReadFile(filepath.Join(dir, "classical-mechanics", "learning-scope.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"status": "ready"`,
		`"mapSlug": "physics"`,
		`"topicId": "classical-mechanics"`,
		`"牛顿运动定律"`,
		`"热现象"`,
	} {
		if !strings.Contains(string(scope), want) {
			t.Fatalf("scope missing %q: %s", want, scope)
		}
	}

	updatedCatalog := strings.Replace(catalog, "牛顿运动定律", "地图后来改变", 1)
	if err := os.WriteFile(filepath.Join(dir, "physics", "discipline-topics.json"), []byte(updatedCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	unchanged, err := os.ReadFile(filepath.Join(dir, "classical-mechanics", "learning-scope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unchanged), "牛顿运动定律") || strings.Contains(string(unchanged), "地图后来改变") {
		t.Fatalf("child scope must remain a creation-time snapshot: %s", unchanged)
	}
}

func TestCreateDeepDiveRejectsUnknownMapTopic(t *testing.T) {
	setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeletonWithInput("physics", "物理学", "", workspace.ProjectInput{
		ProjectType: workspace.ProjectTypeDisciplineMap,
	}); err != nil {
		t.Fatal(err)
	}
	body := `{"title":"不存在","projectType":"system-learning","scopeSource":{"type":"discipline-map","mapSlug":"physics","topicId":"missing"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()
	newTestServer(t).handleCreateProject(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "discipline_topic_not_found") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDisciplineLearningPlanIsLearnerOwnedOrderedTaskList(t *testing.T) {
	setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeletonWithInput("physics-plan", "物理学", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeDisciplineMap}); err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t)
	body := `{"schemaVersion":1,"items":[{"id":"task-1","topicTitle":"经典力学","status":"in-progress","addedAt":"2026-07-18T00:00:00Z","startedAt":"2026-07-18T01:00:00Z"},{"id":"task-2","topicTitle":"热力学","status":"planned","addedAt":"2026-07-18T00:01:00Z"},{"id":"task-3","topicTitle":"电磁学","status":"completed","addedAt":"2026-07-18T00:02:00Z","completedAt":"2026-07-18T02:00:00Z"}]}`
	put := httptest.NewRequest(http.MethodPut, "/api/projects/physics-plan/learning-plan", strings.NewReader(body))
	put.SetPathValue("id", "physics-plan")
	putRec := httptest.NewRecorder()
	server.handlePutDisciplineLearningPlan(putRec, put)
	if putRec.Code != http.StatusOK || !strings.Contains(putRec.Body.String(), `"topicTitle":"经典力学"`) {
		t.Fatalf("put status=%d body=%s", putRec.Code, putRec.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/projects/physics-plan/learning-plan", nil)
	get.SetPathValue("id", "physics-plan")
	getRec := httptest.NewRecorder()
	server.handleGetDisciplineLearningPlan(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	first := strings.Index(getRec.Body.String(), "经典力学")
	second := strings.Index(getRec.Body.String(), "热力学")
	if first < 0 || second < 0 || first >= second {
		t.Fatalf("task order was not preserved: %s", getRec.Body.String())
	}
	root, _ := workspace.ProjectRootForSlug("physics-plan")
	events, err := os.ReadFile(filepath.Join(root, "progress", "events.jsonl"))
	if err != nil || !strings.Contains(string(events), `"sourceType":"learning-task-complete"`) {
		t.Fatalf("completed task did not create an activity check-in: %v %s", err, events)
	}
}

func TestCreateProjectUsesFlatStorage(t *testing.T) {
	dir := setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeletonWithInput("physics", "Physics", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeDisciplineMap}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{"title":"Mechanics","slug":"mechanics","projectType":"system-learning"}`))
	rec := httptest.NewRecorder()
	newTestServer(t).handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "mechanics", "state.json")); err != nil {
		t.Fatalf("flat project missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "physics", "subprojects")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested directory created: %v", err)
	}
}

func TestDeleteProjectRemovesArtifactsReferencesAndSessions(t *testing.T) {
	dir := setupLearningShapeTest(t)
	oldWorkspace := paths.WORKSPACE
	paths.WORKSPACE = dir
	t.Cleanup(func() { paths.WORKSPACE = oldWorkspace })
	if err := workspace.CreateProjectSkeleton("delete-me", "Delete Me", ""); err != nil {
		t.Fatal(err)
	}
	root, _ := workspace.ProjectRootForSlug("delete-me")
	if err := os.WriteFile(filepath.Join(root, "assets", "note.md"), []byte("artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := folderstore.NewFolderStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Replace(folderstore.Layout{Folders: []folderstore.Folder{{ID: "f1", Name: "Folder", SlugOrder: []string{"delete-me"}}}}); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t)
	srv.sessions.Create(&sessionstore.Session{ID: "done", ProjectSlug: "delete-me", State: sessionstore.StateCompleted})

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/delete-me", nil)
	req.SetPathValue("id", "delete-me")
	rec := httptest.NewRecorder()
	srv.handleDeleteProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project root still exists: %v", err)
	}
	reloaded, _ := folderstore.NewFolderStore()
	if got := reloaded.Layout().Folders[0].SlugOrder; len(got) != 0 {
		t.Fatalf("folder reference remains: %v", got)
	}
	if len(srv.sessions.List("delete-me")) != 0 {
		t.Fatal("in-memory sessions remain")
	}
}

func TestDeleteProjectRejectsActiveSession(t *testing.T) {
	setupLearningShapeTest(t)
	if err := workspace.CreateProjectSkeleton("busy", "Busy", ""); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t)
	srv.sessions.Create(&sessionstore.Session{ID: "running", ProjectSlug: "busy", State: sessionstore.StateRunning})
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/busy", nil)
	req.SetPathValue("id", "busy")
	rec := httptest.NewRecorder()
	srv.handleDeleteProject(rec, req)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "project_has_active_session") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if exists, _ := workspace.ProjectExists("busy"); !exists {
		t.Fatal("active project was deleted")
	}
}
