package httpserver

import (
	"errors"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/modules/sources"
	"net/http"
	"os"
)

func structureError(w http.ResponseWriter, e error) {
	status := 500
	code := "structure-failed"
	switch {
	case errors.Is(e, sources.ErrSourceDeleted):
		status = 410
		code = "source-deleted"
	case errors.Is(e, os.ErrNotExist):
		status = 404
		code = "source-not-found"
	case errors.Is(e, sources.ErrDerivedConflict):
		status = 409
		code = "derived-conflict"
	default:
		switch e.Error() {
		case "invalid-arguments", "invalid-pages", "unsupported-structure-format", "no-candidate-pages":
			status = 400
			code = e.Error()
		case "operation-conflict", "structure-active", "structure-unavailable", "provider-unavailable":
			status = 409
			code = e.Error()
		case "structure-queue-full":
			status = 429
			code = e.Error()
		}
	}
	httpx.WriteJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": code}})
}
func (s *Server) handleCreateStructure(w http.ResponseWriter, r *http.Request) {
	var req sources.StructureRequest
	if httpx.ReadJSON(r, &req) != nil {
		httpx.Error(w, 400, "invalid body")
		return
	}
	j, e := sources.CreateStructureJob(r.Context(), r.PathValue("id"), r.PathValue("sourceId"), req)
	if e != nil {
		structureError(w, e)
		return
	}
	httpx.WriteJSON(w, 202, map[string]any{"job": j})
}
func (s *Server) handleGetStructure(w http.ResponseWriter, r *http.Request) {
	j, e := sources.GetStructureJob(r.PathValue("id"), r.PathValue("sourceId"), r.PathValue("jobId"))
	if e != nil {
		structureError(w, e)
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"job": j})
}
func (s *Server) handleCancelStructure(w http.ResponseWriter, r *http.Request) {
	j, e := sources.CancelStructureJob(r.PathValue("id"), r.PathValue("sourceId"), r.PathValue("jobId"))
	if e != nil {
		structureError(w, e)
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"job": j})
}
