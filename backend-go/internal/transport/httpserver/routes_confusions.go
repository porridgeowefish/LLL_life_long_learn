package server

// The historical /confusions surface is now only a compatibility adapter.
// It reads and writes the same canonical body annotation log as the new UI.

import (
	"net/http"
	"time"

	"github.com/xmz14/lll/backend-go/internal/annotationstore"
	"github.com/xmz14/lll/backend-go/internal/confusionstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func (s *Server) handleListConfusions(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	filter := annotationstore.State(r.URL.Query().Get("state"))
	annotations, err := store.List(filter == annotationstore.StateDeleted)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	confusions := make([]confusionstore.Confusion, 0, len(annotations))
	for _, annotation := range annotations {
		if filter == "" || annotation.Status == filter {
			confusions = append(confusions, legacyConfusion(annotation))
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"confusions": confusions})
}

func (s *Server) handleCreateConfusion(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var legacy confusionstore.Confusion
	if err := httpx.ReadJSON(r, &legacy); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	annotation, err := store.Create(annotationstore.CreateInput{QuoteSnapshot: legacy.QuoteSnapshot, Anchors: annotationstore.Anchors{Start: legacy.CharStart, End: legacy.CharEnd}, Note: legacy.Notes, SourceArtifactID: legacy.SourceArtifactID})
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	emitAnnotationUpdated(slug, "create", annotation.AnnotationID)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"confusion": legacyConfusion(annotation)})
}

func (s *Server) handleUpdateConfusion(w http.ResponseWriter, r *http.Request) {
	slug, id := r.PathValue("id"), r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var patch struct {
		Notes *string                `json:"notes"`
		State *annotationstore.State `json:"state"`
	}
	if err := httpx.ReadJSON(r, &patch); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	annotation, err := store.Update(id, patch.Notes, patch.State)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	emitAnnotationUpdated(slug, "update", id)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"confusion": legacyConfusion(annotation)})
}

func (s *Server) handleDeleteConfusion(w http.ResponseWriter, r *http.Request) {
	slug, id := r.PathValue("id"), r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := store.Delete(id); err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	emitAnnotationUpdated(slug, "delete", id)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func legacyConfusion(annotation annotationstore.Annotation) confusionstore.Confusion {
	legacy := confusionstore.Confusion{ID: annotation.AnnotationID, SourceArtifactID: annotation.SourceArtifactID, CharStart: annotation.Anchors.Start, CharEnd: annotation.Anchors.End, QuoteSnapshot: annotation.QuoteSnapshot, Notes: annotation.Note, State: confusionstore.State(annotation.Status), CreatedAt: annotation.CreatedAt.Format(time.RFC3339)}
	if annotation.Ask != nil {
		legacy.Ask = &confusionstore.Ask{Summary: annotation.Ask.Summary, SummaryState: annotation.Ask.SummaryState, ProviderID: annotation.Ask.ProviderID, UpdatedAt: annotation.Ask.UpdatedAt.Format(time.RFC3339)}
		for _, message := range annotation.Ask.Messages {
			role := message.Role
			if role == "learner" {
				role = "user"
			}
			legacy.Ask.Messages = append(legacy.Ask.Messages, confusionstore.AskMessage{ID: message.ID, Role: role, Content: message.Content, CreatedAt: message.CreatedAt.Format(time.RFC3339)})
		}
	}
	return legacy
}
