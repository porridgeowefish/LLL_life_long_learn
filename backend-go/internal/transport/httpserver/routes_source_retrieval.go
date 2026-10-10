package httpserver

import (
	"errors"
	"net/http"
	"os"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	sources "github.com/xmz14/lll/backend-go/internal/modules/sources"
)

func (s *Server) handleSourceEvidence(w http.ResponseWriter, r *http.Request) {
	store, err := sources.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	reader := sources.NewReader(store)
	var preview sources.SourceEvidencePreview
	if direction := r.URL.Query().Get("direction"); direction != "" {
		preview, err = reader.PreviewAdjacent(r.Context(), r.PathValue("sourceId"), r.PathValue("evidenceId"), direction)
	} else {
		preview, err = reader.Preview(r.Context(), r.PathValue("sourceId"), r.PathValue("evidenceId"))
	}
	if err != nil {
		status, code := http.StatusInternalServerError, "internal_error"
		var toolError *sources.SourceToolError
		switch {
		case errors.Is(err, sources.ErrSourceDeleted):
			status, code = http.StatusGone, "source_deleted"
		case errors.Is(err, sources.ErrRevisionChanged):
			status, code = http.StatusConflict, "revision_changed"
		case errors.Is(err, os.ErrNotExist):
			status, code = http.StatusNotFound, "invalid_locator"
		case errors.Is(err, sources.ErrSourceNotReadable):
			status, code = http.StatusConflict, "source_not_readable"
		case errors.As(err, &toolError):
			status, code = http.StatusBadRequest, toolError.Code
		}
		httpx.WriteJSON(w, status, map[string]any{"error": "资料引用不可读取。", "code": code})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, preview)
}

func (s *Server) handleSourceRetrievalStatus(w http.ResponseWriter, r *http.Request) {
	store, err := sources.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, revision, err := store.Get(r.PathValue("sourceId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if source.Status == "deleted" || source.Status == "tombstoned" {
		httpx.WriteJSON(w, http.StatusGone, map[string]any{"error": "资料已删除。", "code": "source_deleted"})
		return
	}
	status := source.Retrieval
	if status == nil {
		status = &sources.RetrievalStatus{State: "absent"}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sourceId": source.SourceID, "revisionId": revision.RevisionID, "retrieval": status})
}
