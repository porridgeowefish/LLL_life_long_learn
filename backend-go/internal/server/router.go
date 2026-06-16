// Package server wires HTTP routes for the LLL backend.
package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/paths"
)

// Server bundles runtime dependencies shared across handlers.
type Server struct {
	ClaudeBin       string
	ClaudeAvailable bool
}

// broadcaster is the package-global SSE event bus.
var broadcaster = httpx.NewBroadcaster()

// New creates a Server and probes the Claude binary once.
// Loads the agent registry from disk. Errors are logged but non-fatal.
func New() *Server {
	bin := "claude"
	if env := envOr("CLAUDE_BIN", ""); env != "" {
		bin = env
	}
	available := probeClaude(bin, 3*time.Second)
	if err := agents.Load(); err != nil {
		println("agent-registry: load warning:", err.Error())
	}
	return &Server{ClaudeBin: bin, ClaudeAvailable: available}
}

// Handler returns the root HTTP handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Projects
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	mux.HandleFunc("GET /api/projects/{id}", s.handleGetProject)
	mux.HandleFunc("GET /api/projects/{id}/tree", s.handleProjectTree)
	mux.HandleFunc("GET /api/projects/{id}/zones/{zone}", s.handleGetZone)
	mux.HandleFunc("POST /api/projects/{id}/subprojects", s.handleCreateSubproject)

	// Agents
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("POST /api/agents/{id}/invoke", s.handleInvokeAgent)

	// Sessions
	mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	mux.HandleFunc("GET /api/sessions/active", s.handleListActiveSessions)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("POST /api/sessions/{id}/follow-up", s.handleFollowUp)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", s.handleCancelSession)

	// File reads (memory / zone outputs) + memory writes (learner-owned edit)
	mux.HandleFunc("GET /files/projects/{id}/", s.handleReadFile)
	mux.HandleFunc("POST /files/projects/{id}/", s.handleWriteFile)

	// Confusions (Explain zone confusion markers)
	mux.HandleFunc("GET /api/projects/{id}/confusions", s.handleListConfusions)
	mux.HandleFunc("POST /api/projects/{id}/confusions", s.handleCreateConfusion)
	mux.HandleFunc("PATCH /api/projects/{id}/confusions/{confusionId}", s.handleUpdateConfusion)
	mux.HandleFunc("DELETE /api/projects/{id}/confusions/{confusionId}", s.handleDeleteConfusion)

	// Practice
	mux.HandleFunc("GET /api/projects/{id}/practice/tasks", s.handleGetPracticeTasks)
	mux.HandleFunc("GET /api/projects/{id}/practice/draft", s.handleGetPracticeDraft)
	mux.HandleFunc("PUT /api/projects/{id}/practice/draft", s.handlePutPracticeDraft)
	mux.HandleFunc("POST /api/projects/{id}/practice/submit", s.handleSubmitPractice)
	mux.HandleFunc("POST /api/projects/{id}/practice/attempts", s.handleCreatePracticeAttempt)
	mux.HandleFunc("GET /api/projects/{id}/practice/attempts/latest", s.handleGetLatestPracticeAttempt)
	mux.HandleFunc("POST /api/projects/{id}/practice/attempts/{attempt}/objective/{taskId}/check", s.handleCheckObjective)
	mux.HandleFunc("POST /api/projects/{id}/practice/attempts/{attempt}/submit", s.handleSubmitPracticeAttempt)
	mux.HandleFunc("POST /api/projects/{id}/practice/attempts/{attempt}/evaluation", s.handleRequestPracticeEvaluation)
	mux.HandleFunc("GET /api/projects/{id}/practice/evaluation", s.handleGetPracticeEvaluation)
	mux.HandleFunc("GET /api/projects/{id}/progress", s.handleGetProgress)

	// Summary (flashcards)
	mux.HandleFunc("GET /api/projects/{id}/summary/flashcards", s.handleListFlashcards)
	mux.HandleFunc("POST /api/projects/{id}/summary/flashcards/grade", s.handleGradeFlashcard)

	// Events (SSE)
	mux.HandleFunc("GET /api/events", broadcaster.SSEHandler(map[string]any{
		"sessions": []any{},
	}))

	// Static frontend (assets + SPA fallback).
	// In production the Vite build emits frontend/dist/. We serve files
	// from there directly; for any non-asset path that the file server
	// cannot resolve, we return dist/index.html so react-router can take
	// over client-side. Asset paths (containing ".") that miss get a 404.
	mux.Handle("/", spaHandler(paths.FRONTEND_ROOT, paths.FRONTEND_INDEX))

	return logging(mux)
}

// spaHandler serves static files from root, falling back to indexFile for
// any path that does not look like an asset. This lets react-router own
// routes like /project/:id/:zone without confusing the file server.
func spaHandler(root, indexFile string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sanitize: never serve paths that try to escape.
		if strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(root, filepath.FromSlash(r.URL.Path))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			// File exists on disk — let FileServer set the right headers.
			fs.ServeHTTP(w, r)
			return
		}
		// Path looks like an asset (has a file extension in the last
		// segment) but does not exist. Return a real 404 so the browser
		// does not cache index.html under, say, /assets/missing.js.
		if strings.Contains(filepath.Base(r.URL.Path), ".") {
			http.NotFound(w, r)
			return
		}
		// SPA route — serve index.html and let react-router handle it.
		http.ServeFile(w, r, indexFile)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Aggregate dashboard stats from project index + session store.
	if err := cache.Rebuild(); err != nil {
		// Non-fatal: continue with whatever cache has.
		println("health: cache rebuild warning:", err.Error())
	}
	projCount := len(cache.All())
	sessCount, turnCount, activeCount := sessions.Stats()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"workspace": paths.WORKSPACE,
		"claude": map[string]any{
			"bin":       s.ClaudeBin,
			"available": s.ClaudeAvailable,
		},
		"stats": map[string]any{
			"projects":       projCount,
			"sessions":       sessCount,
			"turns":          turnCount,
			"activeSessions": activeCount,
		},
	})
}

// probeClaude runs `<bin> --version` with a timeout; returns true on success.
func probeClaude(bin string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// logging wraps h with simple request logging on stdout.
func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		// Only log API calls to keep noise low.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			println(r.Method, r.URL.Path, time.Since(start).String())
		}
	})
}
