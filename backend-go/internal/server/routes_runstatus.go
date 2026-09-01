package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
)

// handleRunStatus receives per-run progress reports from injected Claude Code
// hooks (Phase C). It authenticates via a per-run X-Run-Token issued by the
// launcher, merges the update into the run store, fans out a run-progress SSE
// event, and — on done:true — marks the session completed and emits
// session-completed (the reliable completion signal Phase A lacked).
func (s *Server) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	runId := r.PathValue("runId")
	token := r.Header.Get("X-Run-Token")
	var in runprogress.Status
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	updated, ok := s.runProgress.Set(runId, token, in)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "bad run token")
		return
	}
	broadcaster.Emit("run-progress", map[string]any{
		"runId":        runId,
		"phase":        updated.Phase,
		"activity":     updated.Activity,
		"pagesDone":    updated.PagesDone,
		"pagesPlanned": updated.PagesPlanned,
		"done":         updated.Done,
	})
	if in.Done {
		// Reliable completion for Claude runs (fixes "sessions stay running").
		// SetFinished is a method on the package-global *sessionstore.Store
		// (see routes_sessions.go). For an unknown runId it is a no-op, so a
		// hook firing after a server restart does not fail the request.
		sessions.SetFinished(runId, sessionstore.StateCompleted, 0)
		broadcaster.Emit("session-completed", map[string]any{"runId": runId})
	} else if in.Failed {
		sessions.SetFinished(runId, sessionstore.StateFailed, 1)
		broadcaster.Emit("session-failed", map[string]any{"runId": runId, "error": "runtime exited with a non-zero status"})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": updated})
}
