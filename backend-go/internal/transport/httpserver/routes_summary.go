package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xmz14/lll/backend-go/internal/flashcardstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// GET /api/projects/{id}/summary/flashcards
func (s *Server) handleListFlashcards(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := flashcardstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	cards, err := store.ReadFlashcards()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	prog, _ := store.ReadProgress()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"flashcards": cards,
		"progress":   prog,
	})
}

// POST /api/projects/{id}/summary/flashcards/grade
// Body: { "cardId": "...", "grade": "forgot"|"fuzzy"|"got-it"|"easy" }
func (s *Server) handleGradeFlashcard(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var body struct {
		CardID  string `json:"cardId"`
		Grade   string `json:"grade"`
		EventID string `json:"eventId"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	validGrades := map[string]bool{"forgot": true, "fuzzy": true, "got-it": true, "easy": true}
	if !validGrades[body.Grade] {
		httpx.Error(w, http.StatusBadRequest, "invalid grade: "+body.Grade)
		return
	}
	if body.CardID == "" {
		httpx.Error(w, http.StatusBadRequest, "cardId is required")
		return
	}
	store, err := flashcardstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := store.GradeCard(body.CardID, body.Grade); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	eventID := body.EventID
	if eventID == "" {
		eventID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	growth := 0
	if body.Grade == "got-it" || body.Grade == "easy" {
		growth = 1
	}
	_, _, _ = awardLearningEvent(slug, progressstore.Event{
		ID: "flashcard:" + eventID, SourceType: "flashcard-review", SourceID: body.CardID,
		ActivityDelta: 1, Delta: growth, Outcome: body.Grade,
		Title: "复习闪卡", Detail: body.CardID,
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
