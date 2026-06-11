package server

import (
	"net/http"
	"strconv"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/practicestore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// GET /api/projects/{id}/practice/tasks
func (s *Server) handleGetPracticeTasks(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := practicestore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	tasks, err := store.ReadTasks()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"tasks": nil, "generated": false})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tasks": tasks.Tasks, "generated": true, "generatedAt": tasks.GeneratedAt})
}

// POST /api/projects/{id}/practice/submit — batch submit answers.
// Body: { "submissions": [{ taskId, answer, selfAssess }] }
func (s *Server) handleSubmitPractice(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var body struct {
		Submissions []practicestore.Submission `json:"submissions"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if len(body.Submissions) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no submissions")
		return
	}
	store, err := practicestore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	attempt := store.NextAttempt()
	if err := store.WriteSubmission(attempt, body.Submissions); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"attempt": attempt,
		"saved":   len(body.Submissions),
	})
}

// GET /api/projects/{id}/practice/evaluation?attempt=N
func (s *Server) handleGetPracticeEvaluation(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	attemptStr := r.URL.Query().Get("attempt")
	if attemptStr == "" {
		attemptStr = "1"
	}
	attempt, err := strconv.Atoi(attemptStr)
	if err != nil || attempt < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid attempt")
		return
	}
	store, err := practicestore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	ev, err := store.ReadEvaluation(attempt)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "evaluation not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"evaluation": ev})
}
