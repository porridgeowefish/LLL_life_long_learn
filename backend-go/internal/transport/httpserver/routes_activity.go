package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func (s *Server) handleGetActivity(w http.ResponseWriter, r *http.Request) {
	weeks := 26
	if raw := r.URL.Query().Get("weeks"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "weeks must be 26 or 52")
			return
		}
		weeks = parsed
	}
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if project != "" && !workspace.ValidateSlug(project) {
		httpx.Error(w, http.StatusBadRequest, "invalid project")
		return
	}
	summary, err := progressstore.Aggregate(project, weeks, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"summary": summary})
}

func (s *Server) handlePostActivity(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var in struct {
		ID            string `json:"id"`
		SourceID      string `json:"sourceId"`
		Title         string `json:"title"`
		Detail        string `json:"detail"`
		ActivityDelta int    `json:"activityDelta"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	in.ID, in.SourceID, in.Title, in.Detail = strings.TrimSpace(in.ID), strings.TrimSpace(in.SourceID), strings.TrimSpace(in.Title), strings.TrimSpace(in.Detail)
	if in.ID == "" || len(in.ID) > 180 || in.SourceID == "" || len(in.SourceID) > 180 || len(in.Title) > 100 || len(in.Detail) > 240 || in.ActivityDelta < 1 || in.ActivityDelta > 5 {
		httpx.Error(w, http.StatusBadRequest, "invalid activity event")
		return
	}
	event, added, err := awardLearningEvent(slug, progressstore.Event{
		ID: "reading:" + in.ID, SourceType: "reading", SourceID: in.SourceID,
		ActivityDelta: in.ActivityDelta, Title: in.Title, Detail: in.Detail,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"event": event, "added": added})
}

func awardLearningEvent(slug string, event progressstore.Event) (progressstore.Event, bool, error) {
	if event.PolicyVersion == "" {
		event.PolicyVersion = progressstore.LearningPolicyVersion
	}
	store, err := progressstore.New(slug)
	if err != nil {
		return progressstore.Event{}, false, err
	}
	awarded, added, _, err := store.Award(event)
	return awarded, added, err
}
