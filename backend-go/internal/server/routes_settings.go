package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/agentruntime"
	"github.com/xmz14/lll/backend-go/internal/httpx"
)

type updateAgentRuntimeRequest struct {
	Selected agentruntime.ID `json:"selected"`
}

func (s *Server) handleGetAgentRuntimeSettings(w http.ResponseWriter, r *http.Request) {
	rt, options := s.runtimeSnapshot()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"selected":  rt.ID,
		"runtime":   rt,
		"providers": options,
	})
}

func (s *Server) handlePutAgentRuntimeSettings(w http.ResponseWriter, r *http.Request) {
	var req updateAgentRuntimeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if _, ok := agentruntime.DefinitionByID(req.Selected); !ok {
		httpx.Error(w, http.StatusBadRequest, "unknown agent runtime")
		return
	}
	if err := agentruntime.SaveSelected(req.Selected); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "save agent runtime: "+err.Error())
		return
	}
	cfg, err := agentruntime.Load()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "reload agent runtime: "+err.Error())
		return
	}
	selected := agentruntime.Resolve(cfg)
	options := agentruntime.List(cfg)

	s.mu.Lock()
	s.Runtime = selected
	s.RuntimeOptions = options
	s.mu.Unlock()

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"selected":  selected.ID,
		"runtime":   selected,
		"providers": options,
	})
}
