package httpserver

import (
	"errors"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"net/http"
)

func (s *Server) handleGetTeacherSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := teacher.LoadTeacherSettings()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"autoNotes": cfg})
}
func (s *Server) handlePutTeacherSettings(w http.ResponseWriter, r *http.Request) {
	var raw struct {
		AutoNotes *struct {
			Enabled    *bool  `json:"enabled"`
			EveryTurns *int   `json:"everyTurns"`
			ProviderID string `json:"providerId"`
			Model      string `json:"model"`
		} `json:"autoNotes"`
	}
	if err := httpx.ReadJSON(r, &raw); err != nil || raw.AutoNotes == nil || raw.AutoNotes.Enabled == nil || raw.AutoNotes.EveryTurns == nil {
		httpx.Error(w, http.StatusBadRequest, "enabled and integer everyTurns are required")
		return
	}
	in := raw.AutoNotes
	cfg := teacher.AutoNotesSettings{Enabled: *in.Enabled, EveryTurns: *in.EveryTurns, ProviderID: in.ProviderID, Model: in.Model}
	if s.notes == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "notes service unavailable")
		return
	}
	if err := s.notes.UpdateSettings(cfg); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"autoNotes": cfg})
}
func (s *Server) handleGetNotes(w http.ResponseWriter, r *http.Request) {
	store, err := teacher.NewNoteStore(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	resource, err := store.List()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resource)
}
func (s *Server) handleRetryNote(w http.ResponseWriter, r *http.Request) {
	store, err := teacher.NewNoteStore(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	resource, err := store.List()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	for _, job := range resource.Jobs {
		if job.ID == r.PathValue("jobId") && job.Status == "succeeded" {
			httpx.WriteJSON(w, http.StatusOK, job)
			return
		}
	}
	job, err := store.Retry(r.PathValue("jobId"))
	if errors.Is(err, teacher.ErrNoteJobNotFound) {
		httpx.Error(w, http.StatusNotFound, "note job not found")
		return
	}
	if errors.Is(err, teacher.ErrNoteJobRunning) {
		httpx.Error(w, http.StatusConflict, "note job already running")
		return
	}
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if s.notes != nil {
		s.notes.Notify(r.PathValue("id"))
	}
	httpx.WriteJSON(w, http.StatusAccepted, job)
}
