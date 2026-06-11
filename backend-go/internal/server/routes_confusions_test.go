package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	oldWS := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)

	// Create a project
	if err := workspace.CreateProjectSkeleton("testproj", "Test Project", ""); err != nil {
		t.Fatal(err)
	}

	srv := &Server{ClaudeBin: "echo", ClaudeAvailable: false}

	// Load agents so registry is populated (needed for router).
	regDir := dir + "/registry"
	_ = os.MkdirAll(regDir, 0o755)
	// Minimal agent to avoid load errors
	agentJSON := []byte(`{
		"id": "explain",
		"name": "explain",
		"userStory": "test",
		"allowedZones": ["Explain"],
		"charterPath": "",
		"defaultOutputTargets": [{"zone": "Explain", "filename": "output.md"}]
	}`)
	_ = os.WriteFile(regDir+"/explain.json", agentJSON, 0o644)
	oldAgents := agentregistry.AgentsRootForTest()
	agentregistry.SetAgentsRootForTest(dir)
	_ = agents.Load()
	agentregistry.SetAgentsRootForTest(oldAgents)

	return srv, func() {
		workspace.SetProjectsRootForTest(oldWS)
	}
}

func TestConfusionCRUD(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. List (empty)
	req := httptest.NewRequest("GET", "/api/projects/testproj/confusions", nil)
	req.SetPathValue("id", "testproj")
	w := httptest.NewRecorder()
	srv.handleListConfusions(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: status %d, body %s", w.Code, w.Body.String())
	}
	var listRes struct {
		Confusions []map[string]any `json:"confusions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listRes); err != nil {
		t.Fatal(err)
	}
	if len(listRes.Confusions) != 0 {
		t.Fatalf("expected empty, got %d", len(listRes.Confusions))
	}

	// 2. Create
	body := `{"quoteSnapshot":"什么是所有权？","charStart":0,"charEnd":6}`
	req = httptest.NewRequest("POST", "/api/projects/testproj/confusions", strings.NewReader(body))
	req.SetPathValue("id", "testproj")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.handleCreateConfusion(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", w.Code, w.Body.String())
	}
	var createRes struct {
		Confusion map[string]any `json:"confusion"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createRes); err != nil {
		t.Fatal(err)
	}
	confID, ok := createRes.Confusion["id"].(string)
	if !ok || confID == "" {
		t.Fatal("expected non-empty id")
	}
	if state, _ := createRes.Confusion["state"].(string); state != "open" {
		t.Fatalf("expected state=open, got %s", state)
	}

	// 3. List (1 item)
	req = httptest.NewRequest("GET", "/api/projects/testproj/confusions", nil)
	req.SetPathValue("id", "testproj")
	w = httptest.NewRecorder()
	srv.handleListConfusions(w, req)
	json.Unmarshal(w.Body.Bytes(), &listRes)
	if len(listRes.Confusions) != 1 {
		t.Fatalf("expected 1, got %d", len(listRes.Confusions))
	}

	// 4. Update (mark as asked)
	patchBody := `{"state":"asked","notes":"batch ask"}`
	req = httptest.NewRequest("PATCH", "/api/projects/testproj/confusions/"+confID, strings.NewReader(patchBody))
	req.SetPathValue("id", "testproj")
	req.SetPathValue("confusionId", confID)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.handleUpdateConfusion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: status %d, body %s", w.Code, w.Body.String())
	}

	// 5. Delete (soft)
	req = httptest.NewRequest("DELETE", "/api/projects/testproj/confusions/"+confID, nil)
	req.SetPathValue("id", "testproj")
	req.SetPathValue("confusionId", confID)
	w = httptest.NewRecorder()
	srv.handleDeleteConfusion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: status %d, body %s", w.Code, w.Body.String())
	}
}

func TestPracticeSubmitAndGetTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Get tasks (none yet)
	req := httptest.NewRequest("GET", "/api/projects/testproj/practice/tasks", nil)
	req.SetPathValue("id", "testproj")
	w := httptest.NewRecorder()
	srv.handleGetPracticeTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get tasks: status %d, body %s", w.Code, w.Body.String())
	}
	var tasksRes struct {
		Tasks     any   `json:"tasks"`
		Generated bool  `json:"generated"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tasksRes); err != nil {
		t.Fatal(err)
	}
	if tasksRes.Generated {
		t.Fatal("expected generated=false for empty project")
	}

	// 2. Submit answers
	body := `{"submissions":[{"taskId":"q1","answer":"ownership means memory is managed","selfAssess":3}]}`
	req = httptest.NewRequest("POST", "/api/projects/testproj/practice/submit", strings.NewReader(body))
	req.SetPathValue("id", "testproj")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.handleSubmitPractice(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("submit: status %d, body %s", w.Code, w.Body.String())
	}
	var submitRes struct {
		Attempt int `json:"attempt"`
		Saved   int `json:"saved"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &submitRes); err != nil {
		t.Fatal(err)
	}
	if submitRes.Attempt != 1 {
		t.Fatalf("expected attempt=1, got %d", submitRes.Attempt)
	}
	if submitRes.Saved != 1 {
		t.Fatalf("expected saved=1, got %d", submitRes.Saved)
	}
}

func TestFlashcardListAndGrade(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. List (empty)
	req := httptest.NewRequest("GET", "/api/projects/testproj/summary/flashcards", nil)
	req.SetPathValue("id", "testproj")
	w := httptest.NewRecorder()
	srv.handleListFlashcards(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list flashcards: status %d, body %s", w.Code, w.Body.String())
	}

	// 2. Grade a card
	body := `{"cardId":"fc1","grade":"got-it"}`
	req = httptest.NewRequest("POST", "/api/projects/testproj/summary/flashcards/grade", strings.NewReader(body))
	req.SetPathValue("id", "testproj")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.handleGradeFlashcard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("grade: status %d, body %s", w.Code, w.Body.String())
	}

	// 3. Verify progress persisted
	req = httptest.NewRequest("GET", "/api/projects/testproj/summary/flashcards", nil)
	req.SetPathValue("id", "testproj")
	w = httptest.NewRecorder()
	srv.handleListFlashcards(w, req)
	var res struct {
		Progress []map[string]any `json:"progress"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Progress) != 1 {
		t.Fatalf("expected 1 progress entry, got %d", len(res.Progress))
	}
}

func TestConfusionCreate_MissingQuote(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"charStart":0,"charEnd":5}`
	req := httptest.NewRequest("POST", "/api/projects/testproj/confusions", strings.NewReader(body))
	req.SetPathValue("id", "testproj")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleCreateConfusion(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWriteFile_ExpandedWhitelist(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		path string
		ok   bool
	}{
		{"memory/notes.md", true},
		{"summary/report.md", true},
		{"explain/notes.md", true},
		{"intro/output.md", false},
		{"practice/tasks.md", false},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/files/projects/testproj/"+tt.path, strings.NewReader("test content"))
		req.SetPathValue("id", "testproj")
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		srv.handleWriteFile(w, req)
		if tt.ok && w.Code != http.StatusOK {
			t.Errorf("write %s: expected 200, got %d — %s", tt.path, w.Code, w.Body.String())
		}
		if !tt.ok && w.Code != http.StatusForbidden {
			t.Errorf("write %s: expected 403, got %d — %s", tt.path, w.Code, w.Body.String())
		}
	}
}
