package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/confusionstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// handleListConfusions returns all confusions for a project.
// Query param: ?state=open|asked|resolved|deleted (optional filter).
func (s *Server) handleListConfusions(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	stateFilter := confusionstore.State(r.URL.Query().Get("state"))
	confusions := store.List(stateFilter)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"confusions": confusions})
}

// handleCreateConfusion adds a new confusion marker.
func (s *Server) handleCreateConfusion(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var c confusionstore.Confusion
	if err := httpx.ReadJSON(r, &c); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if c.QuoteSnapshot == "" {
		httpx.Error(w, http.StatusBadRequest, "quoteSnapshot is required")
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, err := store.Create(c)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if broadcaster != nil {
		broadcaster.Emit("confusion-updated", map[string]any{
			"projectSlug": slug, "action": "create", "id": created.ID,
		})
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"confusion": created})
}

// handleUpdateConfusion patches a confusion (notes, state).
func (s *Server) handleUpdateConfusion(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	confID := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var patch map[string]any
	if err := httpx.ReadJSON(r, &patch); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, err := store.Update(confID, patch)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	if broadcaster != nil {
		broadcaster.Emit("confusion-updated", map[string]any{
			"projectSlug": slug, "action": "update", "id": confID,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"confusion": updated})
}

// handleDeleteConfusion soft-deletes a confusion.
func (s *Server) handleDeleteConfusion(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	confID := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := store.Delete(confID); err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	if broadcaster != nil {
		broadcaster.Emit("confusion-updated", map[string]any{
			"projectSlug": slug, "action": "delete", "id": confID,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
