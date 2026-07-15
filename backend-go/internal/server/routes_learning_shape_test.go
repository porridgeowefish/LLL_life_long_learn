package server

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

	"github.com/xmz14/lll/backend-go/internal/agentruntime"
	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/claudelauncher"
	"github.com/xmz14/lll/backend-go/internal/folderstore"
	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func setupLearningShapeTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	oldRoot := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	restoreCfg := askaiconfig.UseConfigPathForTest(filepath.Join(dir, "config.local.json"))
	oldComplete := completeLearningShapeAI
	oldLaunch := launchAgentCLI
	if err := agents.Load(); err != nil {
		t.Fatalf("load agents: %v", err)
	}
	writeAskAiConfig(t, askaiconfig.Provider{ID: "stub", Kind: "openai", BaseURL: "http://stub", APIKey: "k", Model: "m"})
	t.Cleanup(func() {
		workspace.SetProjectsRootForTest(oldRoot)
		restoreCfg()
		completeLearningShapeAI = oldComplete
		launchAgentCLI = oldLaunch
	})
	return dir
}

func TestProjectTypeAdviceHasNoCreationSideEffect(t *testing.T) {
	dir := setupLearningShapeTest(t)
	completeLearningShapeAI = func(context.Context, askaiprovider.Provider, string, []askaiprovider.Message) (string, error) {
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
	completeLearningShapeAI = func(context.Context, askaiprovider.Provider, string, []askaiprovider.Message) (string, error) {
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
	completeLearningShapeAI = func(_ context.Context, _ askaiprovider.Provider, system string, messages []askaiprovider.Message) (string, error) {
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
	completeLearningShapeAI = func(context.Context, askaiprovider.Provider, string, []askaiprovider.Message) (string, error) {
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
	completeLearningShapeAI = func(context.Context, askaiprovider.Provider, string, []askaiprovider.Message) (string, error) {
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
	launched := make(chan claudelauncher.LaunchRequest, 1)
	launchAgentCLI = func(_ context.Context, req claudelauncher.LaunchRequest) (*claudelauncher.RunResult, error) {
		launched <- req
		return &claudelauncher.RunResult{RunDirRel: filepath.Join("runs", req.PromptPackage.RunDirName)}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects/physics/discipline-overview/generate", strings.NewReader(`{}`))
	req.SetPathValue("id", "physics")
	rec := httptest.NewRecorder()
	srv := newTestServer(t)
	srv.Runtime = agentruntime.Runtime{
		Definition: agentruntime.Definition{ID: agentruntime.RuntimeClaude},
		Bin:        "claude",
		Available:  true,
	}
	srv.handleGenerateDisciplineOverview(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var launchReq claudelauncher.LaunchRequest
	select {
	case launchReq = <-launched:
	case <-time.After(time.Second):
		t.Fatal("agent CLI launch was not requested")
	}
	if launchReq.Agent.ID != "encyclopedia" || launchReq.ZoneName != "学科总览" {
		t.Fatalf("unexpected launch scope: agent=%s context=%s", launchReq.Agent.ID, launchReq.ZoneName)
	}
	for _, want := range []string{"学科百科智能体 Charter", "overview.md", "Project-Level Invocation Contract"} {
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
	oldSessions := sessions
	sessions = sessionstore.New()
	t.Cleanup(func() {
		paths.WORKSPACE = oldWorkspace
		sessions = oldSessions
	})
	if err := workspace.CreateProjectSkeleton("delete-me", "Delete Me", ""); err != nil {
		t.Fatal(err)
	}
	root, _ := workspace.ProjectRootForSlug("delete-me")
	if err := os.WriteFile(filepath.Join(root, "memory", "note.md"), []byte("memory"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := folderstore.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Replace(folderstore.Layout{Folders: []folderstore.Folder{{ID: "f1", Name: "Folder", SlugOrder: []string{"delete-me"}}}}); err != nil {
		t.Fatal(err)
	}
	sessions.Create(&sessionstore.Session{ID: "done", ProjectSlug: "delete-me", State: sessionstore.StateCompleted})

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/delete-me", nil)
	req.SetPathValue("id", "delete-me")
	rec := httptest.NewRecorder()
	newTestServer(t).handleDeleteProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project root still exists: %v", err)
	}
	reloaded, _ := folderstore.New()
	if got := reloaded.Layout().Folders[0].SlugOrder; len(got) != 0 {
		t.Fatalf("folder reference remains: %v", got)
	}
	if len(sessions.List("delete-me")) != 0 {
		t.Fatal("in-memory sessions remain")
	}
}

func TestDeleteProjectRejectsActiveSession(t *testing.T) {
	setupLearningShapeTest(t)
	oldSessions := sessions
	sessions = sessionstore.New()
	t.Cleanup(func() { sessions = oldSessions })
	if err := workspace.CreateProjectSkeleton("busy", "Busy", ""); err != nil {
		t.Fatal(err)
	}
	sessions.Create(&sessionstore.Session{ID: "running", ProjectSlug: "busy", State: sessionstore.StateRunning})
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/busy", nil)
	req.SetPathValue("id", "busy")
	rec := httptest.NewRecorder()
	newTestServer(t).handleDeleteProject(rec, req)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "project_has_active_session") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if exists, _ := workspace.ProjectExists("busy"); !exists {
		t.Fatal("active project was deleted")
	}
}
