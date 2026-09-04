package httpserver

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/httpx"
)

// handleListAgents returns every loaded agent.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"agents": s.agents.List()})
}

// handleInvokeAgent delegates to the Phase E implementation in routes_sessions.go.
func (s *Server) handleInvokeAgent(w http.ResponseWriter, r *http.Request) {
	s.handleInvokeAgentImpl(w, r)
}
