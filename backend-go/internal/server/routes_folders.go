package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/folderstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
)

// handleGetFolders returns the workspace-global folder layout.
func (s *Server) handleGetFolders(w http.ResponseWriter, r *http.Request) {
	store, err := folderstore.New()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, store.Layout())
}

// handlePutFolders replaces the whole folder layout. The client computes the
// new state after each create/rename/delete/assign and sends the full layout;
// the store sanitizes and persists atomically. Tree-style membership (a project
// belongs to at most one folder) is enforced server-side.
func (s *Server) handlePutFolders(w http.ResponseWriter, r *http.Request) {
	var in folderstore.Layout
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := folderstore.New()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := store.Replace(in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
