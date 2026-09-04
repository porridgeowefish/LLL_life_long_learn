// Package httpserver wires HTTP routes for the LLL backend. It is a thin
// transport adapter: handlers decode, call one application operation, and
// encode. Composition happens in app/bootstrap, which constructs the
// Server with every dependency injected.
package httpserver

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/compatibility/artifactwatch"
	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/imageconfig"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	runprogress "github.com/xmz14/lll/backend-go/internal/modules/learning"
	projectindex "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

// Server bundles runtime dependencies shared across handlers. Every field
// is injected by the composition root (app/bootstrap); handlers own no
// package-level mutable state.
type Server struct {
	ClaudeBin       string
	ClaudeAvailable bool
	Runtime         assistant.Runtime
	RuntimeOptions  []assistant.Runtime
	ImageConfig     *imageconfig.Config
	ImageAvailable  bool
	runProgress     *runprogress.RunStore
	watcher         *artifactwatch.Watcher
	shutdown        func()
	teacher         *teacher.Service
	agentExecution  AgentExecutionService
	dispatcher      *assistant.Dispatcher
	activeTeacher   *teacher.ActiveResponses
	migrationReady  bool
	migrationFailed []string

	broadcaster      *httpx.Broadcaster
	agents           *assistant.Registry
	sessions         *sessionstore.Store
	cache            *projectindex.Index
	infographicJobs  sync.Map
	practiceEvalJobs sync.Map
	mu               sync.RWMutex
}

type AgentExecutionService interface {
	StartProject(context.Context, assistant.ProjectRequest) (*assistant.RunResult, error)
	StartTask(context.Context, assistant.TaskRequest) error
	Resume(context.Context, assistant.ResumeRequest) (*assistant.RunResult, error)
	StartHeadless(context.Context, assistant.HeadlessRequest) error
}

type Dependencies struct {
	ClaudeBin       string
	ClaudeAvailable bool
	Runtime         assistant.Runtime
	RuntimeOptions  []assistant.Runtime
	ImageConfig     *imageconfig.Config
	ImageAvailable  bool
	RunProgress     *runprogress.RunStore
	Watcher         *artifactwatch.Watcher
	Teacher         *teacher.Service
	Broadcaster     *httpx.Broadcaster
	Agents          *assistant.Registry
	Sessions        *sessionstore.Store
	Cache           *projectindex.Index
	MigrationReady  bool
	MigrationFailed []string
}

// New constructs only the HTTP adapter. Concrete dependency creation and
// background worker startup belong to app/bootstrap.
func New(deps Dependencies) *Server {
	return &Server{
		ClaudeBin:       deps.ClaudeBin,
		ClaudeAvailable: deps.ClaudeAvailable,
		Runtime:         deps.Runtime,
		RuntimeOptions:  deps.RuntimeOptions,
		ImageConfig:     deps.ImageConfig,
		ImageAvailable:  deps.ImageAvailable,
		runProgress:     deps.RunProgress,
		watcher:         deps.Watcher,
		teacher:         deps.Teacher,
		activeTeacher:   teacher.NewActiveResponses(),
		migrationReady:  deps.MigrationReady,
		migrationFailed: deps.MigrationFailed,
		broadcaster:     deps.Broadcaster,
		agents:          deps.Agents,
		sessions:        deps.Sessions,
		cache:           deps.Cache,
	}
}

func (s *Server) AttachExecution(execution AgentExecutionService, dispatcher *assistant.Dispatcher) {
	s.agentExecution = execution
	s.dispatcher = dispatcher
}

// teacherResponses returns the server-wide registry used by every teacher
// request. New normally initializes it, but keeping the fallback behind the
// server mutex makes partial/test composition safe when the first requests
// arrive concurrently.
func (s *Server) teacherResponses() *teacher.ActiveResponses {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTeacher == nil {
		s.activeTeacher = teacher.NewActiveResponses()
	}
	return s.activeTeacher
}

// Broadcaster exposes the SSE event bus for the composition root (tests,
// future integrations). Read-only use only.
func (s *Server) Broadcaster() *httpx.Broadcaster { return s.broadcaster }

// Handler returns the root HTTP handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/system/shutdown", s.handleShutdown)
	mux.HandleFunc("GET /api/settings/agent-runtime", s.handleGetAgentRuntimeSettings)
	mux.HandleFunc("PUT /api/settings/agent-runtime", s.handlePutAgentRuntimeSettings)
	mux.HandleFunc("GET /api/settings/ask-ai", s.handleGetAskAiSettings)
	mux.HandleFunc("PUT /api/settings/ask-ai", s.handlePutAskAiSettings)
	mux.HandleFunc("POST /api/settings/ask-ai/probe", s.handleProbeAskAi)
	mux.HandleFunc("GET /api/settings/ai-services", s.handleGetAIServices)
	mux.HandleFunc("PUT /api/settings/ai-services", s.handlePutAIServices)
	mux.HandleFunc("POST /api/settings/ai-services/probe", s.handleProbeAIServices)
	mux.HandleFunc("GET /api/settings/appearance", s.handleGetAppearance)
	mux.HandleFunc("PUT /api/settings/appearance", s.handlePutAppearance)
	mux.HandleFunc("GET /api/activity", s.handleGetActivity)
	mux.HandleFunc("GET /api/usage/teacher", s.handleListTeacherUsage)

	// Projects
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	mux.HandleFunc("DELETE /api/projects/{id}", s.handleDeleteProject)
	mux.HandleFunc("POST /api/project-type-advice", s.handleProjectTypeAdvice)
	mux.HandleFunc("GET /api/projects/{id}", s.handleGetProject)
	mux.HandleFunc("GET /api/projects/{id}/tree", s.handleProjectTree)
	mux.HandleFunc("GET /api/projects/{id}/zones/{zone}", s.handleGetZone)
	mux.HandleFunc("GET /api/projects/{id}/discipline-overview", s.handleGetDisciplineOverview)
	mux.HandleFunc("GET /api/projects/{id}/discipline-topics", s.handleGetDisciplineTopics)
	mux.HandleFunc("GET /api/projects/{id}/learning-plan", s.handleGetDisciplineLearningPlan)
	mux.HandleFunc("PUT /api/projects/{id}/learning-plan", s.handlePutDisciplineLearningPlan)
	mux.HandleFunc("POST /api/projects/{id}/discipline-overview/generate", s.handleGenerateDisciplineOverview)
	mux.HandleFunc("POST /api/projects/{id}/activity", s.handlePostActivity)
	mux.HandleFunc("GET /api/projects/{id}/conversation", s.handleGetConversation)
	mux.HandleFunc("POST /api/projects/{id}/conversation/turns", s.handleTeacherTurn)
	mux.HandleFunc("GET /api/projects/{id}/conversation/responses/active", s.handleActiveTeacherResponse)
	mux.HandleFunc("POST /api/projects/{id}/conversation/responses/{responseId}/stop", s.handleStopTeacherResponse)
	mux.HandleFunc("GET /api/projects/{id}/assistant-tasks", s.handleListAssistantTasks)
	mux.HandleFunc("GET /api/projects/{id}/assistant-tasks/{taskId}", s.handleGetAssistantTask)
	mux.HandleFunc("GET /api/projects/{id}/assets", s.handleListLearningAssets)
	mux.HandleFunc("GET /api/projects/{id}/assets/{assetKey}", s.handleGetLearningAsset)
	mux.HandleFunc("PUT /api/projects/{id}/assets/{assetKey}", s.handlePutLearningAsset)
	mux.HandleFunc("GET /api/projects/{id}/assets/{assetKey}/versions", s.handleListLearningAssetVersions)
	mux.HandleFunc("GET /api/projects/{id}/sources", s.handleListSources)
	mux.HandleFunc("POST /api/projects/{id}/sources", s.handleUploadSource)
	mux.HandleFunc("GET /api/projects/{id}/sources/{sourceId}", s.handleGetSource)
	mux.HandleFunc("DELETE /api/projects/{id}/sources/{sourceId}", s.handleDeleteSource)
	mux.HandleFunc("POST /api/projects/{id}/sources/{sourceId}/permanent-delete", s.handlePermanentDeleteSource)
	mux.HandleFunc("GET /api/projects/{id}/sources/{sourceId}/revisions/{revisionId}/content", s.handleReadSourceContent)
	mux.HandleFunc("GET /api/projects/{id}/sources/{sourceId}/revisions/{revisionId}/files/{fileKey}", s.handleReadSourceFile)
	mux.HandleFunc("GET /api/projects/{id}/generated", s.handleListGeneratedArtifacts)
	mux.HandleFunc("GET /api/projects/{id}/generated/{artifactId}", s.handleGetGeneratedArtifact)
	mux.HandleFunc("GET /api/projects/{id}/generated/{artifactId}/open", s.handleOpenGeneratedArtifact)
	mux.HandleFunc("GET /api/projects/{id}/generated/{artifactId}/files/{path...}", s.handleReadGeneratedArtifactFile)
	mux.HandleFunc("GET /api/preferences", s.handleGetPreferences)
	mux.HandleFunc("PUT /api/preferences", s.handlePutPreferences)

	// Agents
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("POST /api/agents/{id}/invoke", s.handleInvokeAgent)

	// Sessions
	mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	mux.HandleFunc("GET /api/sessions/active", s.handleListActiveSessions)
	mux.HandleFunc("POST /api/projects/{id}/explain/resume", s.handleResumeExplainSession)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("POST /api/sessions/{id}/follow-up", s.handleFollowUp)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", s.handleCancelSession)

	// Legacy project file reads and learner-owned zone edits. Project memory is
	// deliberately not part of this surface; global preferences use /api/preferences.
	mux.HandleFunc("GET /files/projects/{id}/", s.handleReadFile)
	mux.HandleFunc("POST /files/projects/{id}/", s.handleWriteFile)

	// Confusions (Explain zone confusion markers)
	mux.HandleFunc("GET /api/projects/{id}/confusions", s.handleListConfusions)
	mux.HandleFunc("POST /api/projects/{id}/confusions", s.handleCreateConfusion)
	mux.HandleFunc("PATCH /api/projects/{id}/confusions/{confusionId}", s.handleUpdateConfusion)
	mux.HandleFunc("DELETE /api/projects/{id}/confusions/{confusionId}", s.handleDeleteConfusion)
	mux.HandleFunc("POST /api/projects/{id}/confusions/{confusionId}/ask-stream", s.handleAskAiStream)
	mux.HandleFunc("POST /api/projects/{id}/confusions/{confusionId}/ask/summarize", s.handleAskAiSummarize)
	mux.HandleFunc("GET /api/projects/{id}/assets/body/annotations", s.handleListAnnotations)
	mux.HandleFunc("POST /api/projects/{id}/assets/body/annotations", s.handleCreateAnnotation)
	mux.HandleFunc("PATCH /api/projects/{id}/assets/body/annotations/{annotationId}", s.handleUpdateAnnotation)
	mux.HandleFunc("DELETE /api/projects/{id}/assets/body/annotations/{annotationId}", s.handleDeleteAnnotation)
	mux.HandleFunc("POST /api/projects/{id}/assets/body/annotations/{annotationId}/ask-stream", s.handleAnnotationAskAiStream)
	mux.HandleFunc("POST /api/projects/{id}/assets/body/annotations/{annotationId}/ask/summarize", s.handleAnnotationAskAiSummarize)

	// Project folders (workspace-global virtual grouping of projects)
	mux.HandleFunc("GET /api/folders", s.handleGetFolders)
	mux.HandleFunc("PUT /api/folders", s.handlePutFolders)

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

	// Explain infographic
	mux.HandleFunc("POST /api/projects/{id}/explain/infographic", s.handleRequestExplainInfographic)
	mux.HandleFunc("GET /api/projects/{id}/explain/infographic", s.handleGetExplainInfographic)

	// Run progress reports from injected Claude Code hooks (Phase C).
	mux.HandleFunc("POST /api/runs/{runId}/status", s.handleRunStatus)

	// Events (SSE)
	mux.HandleFunc("GET /api/events", s.broadcaster.SSEHandler(map[string]any{
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

// SetShutdownFunc installs the process-level shutdown callback used by the
// local-only frontend exit button. Tests can leave this unset.
func (s *Server) SetShutdownFunc(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = fn
}

// Close releases background resources (the artifact file watcher).
func (s *Server) Close() {
	if s.dispatcher != nil {
		s.dispatcher.Close()
	}
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRemoteAddr(r.RemoteAddr) {
		httpx.Error(w, http.StatusForbidden, "shutdown is only allowed from localhost")
		return
	}

	s.mu.RLock()
	shutdown := s.shutdown
	s.mu.RUnlock()
	if shutdown == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "shutdown handler is not configured")
		return
	}

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"shuttingDown": true})
	go shutdown()
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
	if err := s.cache.Rebuild(); err != nil {
		// Non-fatal: continue with whatever cache has.
		println("health: cache rebuild warning:", err.Error())
	}
	projCount := len(s.cache.All())
	sessCount, turnCount, activeCount := s.sessions.Stats()
	rt, options := s.runtimeSnapshot()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"workspace": paths.WORKSPACE,
		"claude": map[string]any{
			"bin":       s.ClaudeBin,
			"available": s.ClaudeAvailable,
		},
		"agentRuntime": map[string]any{
			"selected":  rt.ID,
			"runtime":   rt,
			"providers": options,
		},
		"learningWorkspace": map[string]any{
			"mode":           map[bool]string{true: "teacher", false: "legacy"}[s.migrationReady],
			"failedProjects": append([]string{}, s.migrationFailed...),
		},
		"stats": map[string]any{
			"projects":       projCount,
			"sessions":       sessCount,
			"turns":          turnCount,
			"activeSessions": activeCount,
		},
	})
}

func (s *Server) runtimeSnapshot() (assistant.Runtime, []assistant.Runtime) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]assistant.Runtime, len(s.RuntimeOptions))
	copy(out, s.RuntimeOptions)
	return s.Runtime, out
}

func (s *Server) RuntimeSnapshot() assistant.Runtime {
	runtime, _ := s.runtimeSnapshot()
	return runtime
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
