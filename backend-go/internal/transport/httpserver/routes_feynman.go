package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	assetstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	progressstore "github.com/xmz14/lll/backend-go/internal/modules/learning"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

// ---- practice results: one saved score per practice round ----

func (s *Server) handleListPracticeResults(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	results, err := store.PracticeResults()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handlePutPracticeResult(w http.ResponseWriter, r *http.Request) {
	var in assetstore.PracticeResult
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid practice result")
		return
	}
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	saved, err := store.SavePracticeResult(r.PathValue("versionId"), in)
	switch {
	case errors.Is(err, assetstore.ErrInvalidPracticeResult):
		httpx.Error(w, http.StatusBadRequest, "invalid practice result")
		return
	case errors.Is(err, assetstore.ErrPracticeRoundNotFound):
		httpx.Error(w, http.StatusNotFound, "practice round not found")
		return
	case err != nil:
		learningWorkspaceError(w, err)
		return
	}
	_, _, _ = s.recordLearningActivity(r.PathValue("id"), progressstore.ProgressEvent{
		ID: "practice-result:" + saved.VersionID + ":" + saved.SubmittedAt.Format("20060102T150405.000"), SourceType: "practice-result", SourceID: saved.VersionID,
		ActivityDelta: 1, Title: "完成练习测试", Detail: fmt.Sprintf("得分 %d/%d", saved.Score, saved.Total),
	})
	httpx.WriteJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeletePracticeResult(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if err := store.DeletePracticeResult(r.PathValue("versionId")); err != nil {
		if errors.Is(err, assetstore.ErrPracticeRoundNotFound) {
			httpx.Error(w, http.StatusNotFound, "practice round not found")
			return
		}
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---- 讲给教师听 (Feynman mode) ----

func (s *Server) handleListFeynmanSessions(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	sessions, err := store.FeynmanSessions()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleCreateFeynmanSession(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	review, err := store.Get("review")
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if strings.TrimSpace(review.Content) == "" {
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"code": "review_empty", "message": "复习稿还是空的，先生成或编写复习稿，再来讲给教师听。"}})
		return
	}
	session, err := store.CreateFeynmanSession()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, session)
}

func (s *Server) handleFeynmanStream(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	var in struct {
		ProviderID string `json:"providerId"`
		Content    string `json:"content"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Content = strings.TrimSpace(in.Content)
	store, err := assetstore.New(slug)
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	session, err := store.FeynmanSession(r.PathValue("sessionId"))
	if err != nil {
		if errors.Is(err, assetstore.ErrFeynmanSessionNotFound) {
			httpx.Error(w, http.StatusNotFound, "feynman session not found")
			return
		}
		learningWorkspaceError(w, err)
		return
	}
	// An empty message is only the opening of a brand-new round.
	if in.Content == "" && len(session.Messages) > 0 {
		httpx.Error(w, http.StatusBadRequest, "content required")
		return
	}
	cfg, err := teacher.LoadConfig()
	if err != nil || !cfg.Enabled() {
		httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
		return
	}
	pc := cfg.Find(in.ProviderID)
	if in.ProviderID == "" {
		pc = cfg.Resolve("teacher")
	}
	if pc == nil {
		httpx.Error(w, http.StatusBadRequest, "provider not found")
		return
	}
	review, _, err := store.ReviewContentForFeynman(session)
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}

	msgs := make([]teacher.AIMessage, 0, len(session.Messages)+1)
	for _, m := range session.Messages {
		role := "user"
		if m.Role == "teacher" {
			role = "assistant"
		}
		msgs = append(msgs, teacher.AIMessage{Role: role, Content: m.Content})
	}
	if in.Content == "" {
		msgs = append(msgs, teacher.AIMessage{Role: "user", Content: teacher.FeynmanOpening})
	} else {
		updated, err := store.AppendFeynmanMessage(session.SessionID, assetstore.FeynmanMessage{Role: "learner", Content: in.Content})
		if err != nil {
			learningWorkspaceError(w, err)
			return
		}
		msgs = append(msgs, teacher.AIMessage{Role: "user", Content: in.Content})
		last := updated.Messages[len(updated.Messages)-1]
		_, _, _ = s.recordLearningActivity(slug, progressstore.ProgressEvent{
			ID: "feynman:" + last.ID, SourceType: "feynman", SourceID: session.SessionID,
			ActivityDelta: 1, Title: "讲给教师听",
		})
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	prov := teacher.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model}
	var sb strings.Builder
	writeFrame := func(f teacher.AIFrame) {
		if f.Type == "thinking" {
			return
		}
		b, _ := json.Marshal(f)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		if f.Type == "text" {
			sb.WriteString(f.Content)
		}
	}
	err = teacher.Stream(r.Context(), prov, teacher.FeynmanSystemPrompt(review), msgs, writeFrame)
	status := "completed"
	if err != nil {
		status = "failed"
		if errors.Is(err, context.Canceled) {
			status = "interrupted"
		}
	}
	if sb.Len() > 0 {
		_, _ = store.AppendFeynmanMessage(session.SessionID, assetstore.FeynmanMessage{Role: "teacher", Content: sb.String(), Status: status})
	}
	if err != nil {
		writeFrame(teacher.AIFrame{Type: "error", Content: "回答暂时中断，已保留收到的部分内容。"})
	}
}
