package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/claudelauncher"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// sessions is the package-global session store.
var sessions = sessionstore.New()

// invokeRequest is the body of POST /api/agents/{id}/invoke.
type invokeRequest struct {
	ProjectID             string   `json:"projectId"`
	Zone                  string   `json:"zone"`
	Intent                string   `json:"intent"`
	PermissionMode        string   `json:"permissionMode,omitempty"`
	SourceRefs            []string `json:"sourceRefs,omitempty"` // confusion IDs to inject
	ParentPageID          string   `json:"parentPageId,omitempty"`
	PracticeAttempt       int      `json:"practiceAttempt,omitempty"`
	PracticeQuestionCount int      `json:"practiceQuestionCount,omitempty"`
}

// handleInvokeAgent orchestrates an agent invocation: validates, resolves
// predecessors, builds the prompt, creates a session, launches Claude.
// The Claude process runs in a background goroutine; this handler returns
// 201 with the session as soon as the process is spawned.
func (s *Server) handleInvokeAgentImpl(w http.ResponseWriter, r *http.Request) {
	runtime, _ := s.runtimeSnapshot()
	if !runtime.Available {
		httpx.Error(w, http.StatusServiceUnavailable, string(runtime.ID)+" binary not available")
		return
	}
	agentID := r.PathValue("id")
	agent, ok := agents.Get(agentID)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "agent not found: "+agentID)
		return
	}
	var req invokeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if !workspace.ValidateSlug(req.ProjectID) {
		httpx.Error(w, http.StatusBadRequest, "invalid projectId")
		return
	}
	if !workspace.ValidateZoneName(req.Zone) {
		httpx.Error(w, http.StatusBadRequest, "invalid zone: "+req.Zone)
		return
	}
	intent := strings.TrimSpace(req.Intent)
	permissionMode := claudelauncher.NormalizePermissionMode(req.PermissionMode)
	exists, err := workspace.ProjectExists(req.ProjectID)
	if err != nil || !exists {
		httpx.Error(w, http.StatusNotFound, "project not found")
		return
	}
	zoneName := workspace.ZoneName(req.Zone)

	// Build the prompt package.
	pkg, err := promptassembly.Build(promptassembly.Request{
		ProjectSlug:           req.ProjectID,
		ZoneName:              zoneName,
		AgentID:               agent.ID,
		Intent:                intent,
		SourceRefs:            req.SourceRefs,
		ParentPageID:          strings.TrimSpace(req.ParentPageID),
		PracticeAttempt:       req.PracticeAttempt,
		PracticeQuestionCount: req.PracticeQuestionCount,
	}, agents)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "prompt assembly: "+err.Error())
		return
	}

	// Create session.
	sessID := newSessionID()
	sess := &sessionstore.Session{
		ID:          sessID,
		ProjectSlug: req.ProjectID,
		ZoneName:    string(zoneName),
		AgentID:     agent.ID,
		State:       sessionstore.StatePreparing,
		CreatedAt:   time.Now().UTC(),
	}
	sessions.Create(sess)
	// Initial system turn.
	sessions.AppendTurn(sessID, "system", "session created", "")
	_, _, _ = awardLearningEvent(req.ProjectID, progressstore.Event{
		ID: "agent-invoke:" + sessID, SourceType: "agent-invoke", SourceID: agent.ID,
		ActivityDelta: 1, Title: "开始学习", Detail: string(zoneName) + " · " + agent.Name,
	})
	if broadcaster != nil {
		broadcaster.Emit("session-created", map[string]any{"sessionId": sessID, "session": sess})
	}

	// Fire the launcher in a background goroutine.
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sessions.SetCancel(sessID, make(chan struct{}))
		// Wire cancel from the session's channel.
		go func() {
			if ch := getSessionCancelChannel(sessID); ch != nil {
				select {
				case <-ch:
					cancel()
				case <-ctx.Done():
				}
			}
		}()
		_, launchErr := claudelauncher.Launch(ctx, claudelauncher.LaunchRequest{
			ProjectSlug:    req.ProjectID,
			ZoneName:       zoneName,
			Agent:          agent,
			PromptPackage:  pkg,
			PermissionMode: permissionMode,
			Session:        sess,
			Store:          sessions,
			Events:         broadcaster,
			ClaudeBin:      s.ClaudeBin,
			Runtime:        &runtime,
			RunProgress:    s.runProgress,
		})
		if launchErr != nil {
			os.WriteFile(filepath.Join(
				mustProjectRoot(req.ProjectID), pkg.RunDirName, "stderr.log"),
				[]byte("launch error: "+launchErr.Error()), 0o644)
		}
	}()

	// Return the session to the caller immediately.
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"session": sess,
		"runDir":  pkg.RunDirName,
	})
}

// getSessionCancelChannel returns the cancel channel for a session (or nil).
func getSessionCancelChannel(id string) chan struct{} {
	// We don't expose the channel directly; let Cancel() do the work.
	return nil
}

func mustProjectRoot(slug string) string {
	r, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return "."
	}
	return r
}

// handleListSessions supports query parameters:
//   - ?active=true             → only in-flight sessions
//   - ?recent=true&limit=N     → most recent N sessions (default 10)
//   - (no params)              → all sessions
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 10
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = n
	}
	var list []*sessionstore.Session
	switch {
	case q.Has("active"):
		list = sessions.ListActive()
	case q.Has("recent"):
		list = sessions.ListRecent(limit)
	default:
		list = sessions.List("")
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

// handleListActiveSessions returns sessions currently in flight (preparing / launching / running / awaiting-follow-up).
// Convenience endpoint equivalent to GET /api/sessions?active=true.
func (s *Server) handleListActiveSessions(w http.ResponseWriter, r *http.Request) {
	list := sessions.ListActive()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

// handleResumeExplainSession opens Claude Code's most recent conversation via
// `claude -c` from the project root. It is intentionally scoped to Explain:
// the UI uses this when the learner is reading generated Explain material and
// wants to reopen the closed native TUI to keep asking questions.
func (s *Server) handleResumeExplainSession(w http.ResponseWriter, r *http.Request) {
	if !s.ClaudeAvailable {
		httpx.Error(w, http.StatusServiceUnavailable, "claude binary not available")
		return
	}
	projectID := r.PathValue("id")
	if !workspace.ValidateSlug(projectID) {
		httpx.Error(w, http.StatusBadRequest, "invalid projectId")
		return
	}
	exists, err := workspace.ProjectExists(projectID)
	if err != nil || !exists {
		httpx.Error(w, http.StatusNotFound, "project not found")
		return
	}
	sess := latestProjectZoneSession(projectID, string(workspace.ZoneExplain))
	runDirName := promptassembly.MakeRunDirName("explain-resume", time.Now().UTC())
	result, err := claudelauncher.LaunchResume(context.Background(), claudelauncher.ResumeRequest{
		ProjectSlug: projectID,
		ZoneName:    workspace.ZoneExplain,
		Session:     sess,
		Store:       sessions,
		Events:      broadcaster,
		ClaudeBin:   s.ClaudeBin,
		RunDirName:  runDirName,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "resume explain session: "+err.Error())
		return
	}
	_, _, _ = awardLearningEvent(projectID, progressstore.Event{
		ID: "agent-resume:" + runDirName, SourceType: "agent-resume", SourceID: "explain",
		ActivityDelta: 1, Title: "继续学习", Detail: "Explain · 恢复会话",
	})
	payload := map[string]any{
		"resumed": true,
		"runDir":  filepath.Base(result.RunDirRel),
	}
	if sess != nil {
		payload["session"] = sess
	}
	httpx.WriteJSON(w, http.StatusAccepted, payload)
}

// handleGetSession returns one session's state including the full turn timeline.
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := sessions.Get(id)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "session not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"session": sess})
}

// followupRequest is the body of POST /api/sessions/{id}/follow-up.
type followupRequest struct {
	Text           string `json:"text"`
	PermissionMode string `json:"permissionMode,omitempty"`
}

// handleFollowUp appends a follow-up turn to an existing session by re-launching
// Claude with prior result.md paths as context.
func (s *Server) handleFollowUp(w http.ResponseWriter, r *http.Request) {
	runtime, _ := s.runtimeSnapshot()
	if !runtime.Available {
		httpx.Error(w, http.StatusServiceUnavailable, string(runtime.ID)+" binary not available")
		return
	}
	id := r.PathValue("id")
	sess, ok := sessions.Get(id)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "session not found")
		return
	}
	var req followupRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		httpx.Error(w, http.StatusBadRequest, "text is required")
		return
	}
	permissionMode := claudelauncher.NormalizePermissionMode(req.PermissionMode)
	agent, ok := agents.Get(sess.AgentID)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "agent missing: "+sess.AgentID)
		return
	}
	// Build the list of prior assistant result.md paths to include as context.
	priorPaths := collectPriorResultPaths(sess)

	pkg, err := promptassembly.Build(promptassembly.Request{
		ProjectSlug:              sess.ProjectSlug,
		ZoneName:                 workspace.ZoneName(sess.ZoneName),
		AgentID:                  sess.AgentID,
		Intent:                   text,
		FollowupPriorResultPaths: priorPaths,
	}, agents)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "prompt assembly: "+err.Error())
		return
	}
	// Reset session state to running, then launch again.
	sessions.SetState(sess.ID, sessionstore.StateRunning)
	_, _, _ = awardLearningEvent(sess.ProjectSlug, progressstore.Event{
		ID: "agent-followup:" + sess.ID + ":" + pkg.RunDirName, SourceType: "agent-followup", SourceID: sess.AgentID,
		ActivityDelta: 1, Title: "追问学习 Agent", Detail: sess.ZoneName,
	})
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, _ = claudelauncher.Launch(ctx, claudelauncher.LaunchRequest{
			ProjectSlug:    sess.ProjectSlug,
			ZoneName:       workspace.ZoneName(sess.ZoneName),
			Agent:          agent,
			PromptPackage:  pkg,
			PermissionMode: permissionMode,
			Session:        sess,
			Store:          sessions,
			Events:         broadcaster,
			ClaudeBin:      s.ClaudeBin,
			Runtime:        &runtime,
			RunProgress:    s.runProgress,
		})
	}()

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{
		"sessionId": sess.ID,
		"runDir":    pkg.RunDirName,
	})
}

// handleCancelSession cancels a running session.
func (s *Server) handleCancelSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ok := sessions.Cancel(id)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "session not running or not found")
		return
	}
	sessions.SetState(id, sessionstore.StateCancelled)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessionId": id, "cancelled": true})
}

// collectPriorResultPaths returns absolute paths to result.md for every
// assistant turn in the session (so the follow-up can read them).
func collectPriorResultPaths(sess *sessionstore.Session) []string {
	root, err := workspace.ProjectRootForSlug(sess.ProjectSlug)
	if err != nil {
		return nil
	}
	var out []string
	for _, t := range sess.Turns {
		if t.Type == "assistant" && t.RunDirRel != "" {
			out = append(out, filepath.Join(root, t.RunDirRel, "result.md"))
		}
	}
	return out
}

func latestProjectZoneSession(projectSlug, zoneName string) *sessionstore.Session {
	var latest *sessionstore.Session
	for _, sess := range sessions.List(projectSlug) {
		if sess.ZoneName != zoneName {
			continue
		}
		if latest == nil || sess.CreatedAt.After(latest.CreatedAt) {
			latest = sess
		}
	}
	return latest
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString(b[:]) // fallback to zeros
	}
	return hex.EncodeToString(b[:])
}

var _ = errors.New
