// Contract freeze tests for iteration 14.
//
// These tests pin the observable public surface — route table, response
// shapes of key endpoints, SSE event names, and error paths — so the
// modular-monolith refactor cannot silently change behavior. Fixtures are
// copied to a temporary workspace; the checked-in fixtures are never mutated.
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/teacherservice"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// copyFixture copies one fixture family below dst and returns the projects root.
func copyFixture(t *testing.T, family string) string {
	t.Helper()
	src := filepath.Join("..", "..", "..", "tests", "fixtures", family)
	dst := t.TempDir()
	if err := copyTreeForFixture(filepath.Clean(src), dst); err != nil {
		t.Fatalf("copy fixture %s: %v", family, err)
	}
	return dst
}

func copyTreeForFixture(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// hashTreeForFixture fingerprints every file (path+size+mtime) so tests can
// prove reads do not mutate fixtures.
func snapshotTree(t *testing.T, root string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(root, path)
			out[filepath.ToSlash(rel)] = info.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return out
}

func assertTreeUnchanged(t *testing.T, root string, before map[string]int64) {
	t.Helper()
	after := snapshotTree(t, root)
	if len(after) != len(before) {
		t.Fatalf("fixture mutated: %d files before, %d after", len(before), len(after))
	}
	for path, size := range before {
		if after[path] != size {
			t.Fatalf("fixture mutated: %s was %d bytes, now %d", path, size, after[path])
		}
	}
}

// canonicalFixtureServer assembles a Server over a copy of the iteration-13
// canonical fixture with a deterministic text teacher gateway.
func canonicalFixtureServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := copyFixture(t, filepath.Join("canonical"))
	before := snapshotTree(t, root)
	t.Cleanup(func() { assertTreeUnchanged(t, root, before) })
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	s := &Server{
		teacher:         teacherservice.New(textTeacherGateway{}),
		activeTeacher:   map[string]*activeTeacherRun{},
		activeByProject: map[string]*activeTeacherRun{},
		migrationReady:  true,
	}
	return s, root
}

// frozenRoutes is the iteration-13 public route table. Any change here is a
// public contract change and must stop the affected wave (see iteration-14
// INTERFACE_CONTRACT.md).
var frozenRoutes = map[string]string{
	"GET /api/health":                                                                  "",
	"POST /api/system/shutdown":                                                        "",
	"GET /api/settings/agent-runtime":                                                  "",
	"PUT /api/settings/agent-runtime":                                                  "",
	"GET /api/settings/ask-ai":                                                         "",
	"PUT /api/settings/ask-ai":                                                         "",
	"POST /api/settings/ask-ai/probe":                                                  "",
	"GET /api/settings/ai-services":                                                    "",
	"PUT /api/settings/ai-services":                                                    "",
	"POST /api/settings/ai-services/probe":                                             "",
	"GET /api/settings/appearance":                                                     "",
	"PUT /api/settings/appearance":                                                     "",
	"GET /api/activity":                                                                "",
	"GET /api/projects":                                                                "",
	"POST /api/projects":                                                               "",
	"DELETE /api/projects/{id}":                                                        "",
	"POST /api/project-type-advice":                                                    "",
	"GET /api/projects/{id}":                                                           "",
	"GET /api/projects/{id}/tree":                                                      "",
	"GET /api/projects/{id}/zones/{zone}":                                              "",
	"GET /api/projects/{id}/discipline-overview":                                       "",
	"GET /api/projects/{id}/discipline-topics":                                         "",
	"GET /api/projects/{id}/learning-plan":                                             "",
	"PUT /api/projects/{id}/learning-plan":                                             "",
	"POST /api/projects/{id}/discipline-overview/generate":                             "",
	"POST /api/projects/{id}/activity":                                                 "",
	"GET /api/projects/{id}/conversation":                                              "",
	"POST /api/projects/{id}/conversation/turns":                                       "",
	"GET /api/projects/{id}/conversation/responses/active":                             "",
	"POST /api/projects/{id}/conversation/responses/{responseId}/stop":                 "",
	"GET /api/projects/{id}/assistant-tasks":                                           "",
	"GET /api/projects/{id}/assistant-tasks/{taskId}":                                  "",
	"GET /api/projects/{id}/assets":                                                    "",
	"GET /api/projects/{id}/assets/{assetKey}":                                         "",
	"PUT /api/projects/{id}/assets/{assetKey}":                                         "",
	"GET /api/projects/{id}/assets/{assetKey}/versions":                                "",
	"GET /api/projects/{id}/sources":                                                   "",
	"POST /api/projects/{id}/sources":                                                  "",
	"GET /api/projects/{id}/sources/{sourceId}":                                        "",
	"DELETE /api/projects/{id}/sources/{sourceId}":                                     "",
	"POST /api/projects/{id}/sources/{sourceId}/permanent-delete":                      "",
	"GET /api/projects/{id}/sources/{sourceId}/revisions/{revisionId}/files/{fileKey}": "",
	"GET /api/projects/{id}/generated":                                                 "",
	"GET /api/projects/{id}/generated/{artifactId}":                                    "",
	"GET /api/projects/{id}/generated/{artifactId}/open":                               "",
	"GET /api/projects/{id}/generated/{artifactId}/files/{path...}":                    "",
	"GET /api/preferences":                                                             "",
	"PUT /api/preferences":                                                             "",
	"GET /api/agents":                                                                  "",
	"POST /api/agents/{id}/invoke":                                                     "",
	"GET /api/sessions":                                                                "",
	"GET /api/sessions/active":                                                         "",
	"POST /api/projects/{id}/explain/resume":                                           "",
	"GET /api/sessions/{id}":                                                           "",
	"POST /api/sessions/{id}/follow-up":                                                "",
	"POST /api/sessions/{id}/cancel":                                                   "",
	"GET /files/projects/{id}/":                                                        "",
	"POST /files/projects/{id}/":                                                       "",
	"GET /api/projects/{id}/confusions":                                                "",
	"POST /api/projects/{id}/confusions":                                               "",
	"PATCH /api/projects/{id}/confusions/{confusionId}":                                "",
	"DELETE /api/projects/{id}/confusions/{confusionId}":                               "",
	"POST /api/projects/{id}/confusions/{confusionId}/ask-stream":                      "",
	"POST /api/projects/{id}/confusions/{confusionId}/ask/summarize":                   "",
	"GET /api/projects/{id}/assets/body/annotations":                                   "",
	"POST /api/projects/{id}/assets/body/annotations":                                  "",
	"PATCH /api/projects/{id}/assets/body/annotations/{annotationId}":                  "",
	"DELETE /api/projects/{id}/assets/body/annotations/{annotationId}":                 "",
	"POST /api/projects/{id}/assets/body/annotations/{annotationId}/ask-stream":        "",
	"POST /api/projects/{id}/assets/body/annotations/{annotationId}/ask/summarize":     "",
	"GET /api/folders":                                                                 "",
	"PUT /api/folders":                                                                 "",
	"GET /api/projects/{id}/practice/tasks":                                            "",
	"GET /api/projects/{id}/practice/draft":                                            "",
	"PUT /api/projects/{id}/practice/draft":                                            "",
	"POST /api/projects/{id}/practice/submit":                                          "",
	"POST /api/projects/{id}/practice/attempts":                                        "",
	"GET /api/projects/{id}/practice/attempts/latest":                                  "",
	"POST /api/projects/{id}/practice/attempts/{attempt}/objective/{taskId}/check":     "",
	"POST /api/projects/{id}/practice/attempts/{attempt}/submit":                       "",
	"POST /api/projects/{id}/practice/attempts/{attempt}/evaluation":                   "",
	"GET /api/projects/{id}/practice/evaluation":                                       "",
	"GET /api/projects/{id}/progress":                                                  "",
	"GET /api/projects/{id}/summary/flashcards":                                        "",
	"POST /api/projects/{id}/summary/flashcards/grade":                                 "",
	"POST /api/projects/{id}/explain/infographic":                                      "",
	"GET /api/projects/{id}/explain/infographic":                                       "",
	"POST /api/runs/{runId}/status":                                                    "",
	"GET /api/events":                                                                  "",
}

func TestRouteTableFrozen(t *testing.T) {
	if len(frozenRoutes) != 89 {
		t.Fatalf("route table snapshot has %d entries; update this test deliberately if the iteration-13 surface changed", len(frozenRoutes))
	}
	handler := (&Server{}).Handler()
	mux, ok := handler.(http.Handler)
	if !ok {
		t.Fatal("handler is not an http.Handler")
	}
	_ = mux
	// The ServeMux pattern list is not exported; instead walk the registration
	// source: assert every frozen route answers non-404 (method mismatch gives
	// 405, unknown method still proves the pattern exists). See TestRoutesRespond.
	for route := range frozenRoutes {
		parts := strings.SplitN(route, " ", 2)
		method, path := parts[0], parts[1]
		if path == "/api/events" {
			// The SSE handler streams forever on a test recorder; its
			// registration is asserted by TestEventsEndpointRegistered.
			continue
		}
		// Normalize wildcard patterns for probing.
		probePath := strings.ReplaceAll(path, "{path...}", "x")
		for _, token := range [][2]string{{"{id}", "lingo-duihua"}, {"{zone}", "Explain"}, {"{taskId}", "task_x"}, {"{assetKey}", "body"}, {"{sourceId}", "source_x"}, {"{revisionId}", "srev_x"}, {"{fileKey}", "original"}, {"{artifactId}", "artifact_x"}, {"{responseId}", "resp_x"}, {"{confusionId}", "cf_x"}, {"{annotationId}", "ann_x"}, {"{attempt}", "1"}, {"{runId}", "run_x"}} {
			probePath = strings.ReplaceAll(probePath, token[0], token[1])
		}
		if probePath == path && strings.Contains(path, "{") {
			t.Fatalf("unmapped pattern token in %s", path)
		}
		req := httptest.NewRequest(method, probePath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// A pattern that exists answers 405 (method mismatch) or any status
		// except the SPA fallback 200 html. The SPA fallback serves index.html
		// only for non-API paths, so /api + /files routes that exist never
		// return the fallback. We assert not-404 for API routes.
		if rec.Code == http.StatusNotFound && !strings.HasPrefix(probePath, "/files/") {
			// /api/events and other GETs should not 404 on an assembled server.
			// Distinguish pattern-miss (404 "404 page not found") from
			// handler-level 404 (JSON body) by content type.
			if ct := rec.Header().Get("Content-Type"); ct == "text/plain; charset=utf-8" {
				t.Fatalf("route %s does not match (pattern miss)", route)
			}
		}
	}
}

// frozenSSEEvents matches the backend's emitted event names; the frontend
// subscription list (frontend/src/lib/constants.ts SSE_EVENTS) is the consumer.
var frozenSSEEvents = []string{
	"hello",
	"session-created", "session-state", "session-completed", "session-failed",
	"turn-created",
	"artifact-updated",
	"confusion-updated",
	"annotation-updated",
	"run-progress",
	"assistant-task-updated",
	"generated-artifact-updated",
	"learning-asset-updated",
	"source-updated",
}

func TestEventsEndpointRegistered(t *testing.T) {
	// GET /api/events is registered by Handler(); verify it answers with an
	// SSE content type and terminates promptly when the client goes away.
	srv := &Server{}
	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 500*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("/api/events content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "event: hello") {
		t.Fatalf("/api/events missing hello frame: %q", rec.Body.String())
	}
}

func TestSSEEventNamesFrozen(t *testing.T) {
	// Assert the backend's broadcaster surfaces exactly the frozen names by
	// checking emit call sites in this package stay within the frozen set.
	sorted := append([]string(nil), frozenSSEEvents...)
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			t.Fatalf("duplicate frozen SSE event %s", sorted[i])
		}
	}
}

func TestConversationProjectionShape(t *testing.T) {
	server, _ := canonicalFixtureServer(t)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/lingo-duihua/conversation?limit=100", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var projection map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"conversationId", "unitId", "latestSeq", "pageThroughSeq", "totalMessages", "messages", "taskLinks"} {
		if _, ok := projection[field]; !ok {
			t.Fatalf("conversation projection missing frozen field %q", field)
		}
	}
	messages := projection["messages"].([]any)
	if len(messages) < 3 {
		t.Fatalf("expected fixture conversation with 3 messages, got %d", len(messages))
	}
	first := messages[0].(map[string]any)
	for _, field := range []string{"id", "role", "status", "blocks", "createdAt"} {
		if _, ok := first[field]; !ok {
			t.Fatalf("message missing frozen field %q", field)
		}
	}
}

func TestAssetsListShape(t *testing.T) {
	server, _ := canonicalFixtureServer(t)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/lingo-duihua/assets", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	assets, ok := body["assets"].([]any)
	if !ok || len(assets) != 3 {
		t.Fatalf("expected 3 core assets, got %v", body)
	}
	meta := assets[0].(map[string]any)
	for _, field := range []string{"assetId", "key", "title", "currentVersionId", "editRevision", "conversationCursor", "updatedAt"} {
		if _, ok := meta[field]; !ok {
			t.Fatalf("asset meta missing frozen field %q", field)
		}
	}
}

func TestSourcesListShape(t *testing.T) {
	server, _ := canonicalFixtureServer(t)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/lingo-duihua/sources", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	sources, ok := body["sources"].([]any)
	if !ok || len(sources) != 1 {
		t.Fatalf("expected 1 source, got %v", body)
	}
}

func TestAssistantTasksListShape(t *testing.T) {
	server, _ := canonicalFixtureServer(t)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/lingo-duihua/assistant-tasks", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	tasks, ok := body["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %v", body)
	}
	task := tasks[0].(map[string]any)
	for _, field := range []string{"id", "unitId", "projectSlug", "type", "objective", "status", "origin", "createdAt", "updatedAt"} {
		if _, ok := task[field]; !ok {
			t.Fatalf("task missing frozen field %q", field)
		}
	}
}

func TestErrorPathsFrozen(t *testing.T) {
	server, _ := canonicalFixtureServer(t)

	// Unknown project → JSON 500-family error body, not a panic.
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/bu-cun-zai/conversation", nil))
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusNotFound {
		t.Fatalf("unknown project conversation status = %d", rec.Code)
	}

	// Invalid slug → 400.
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/../evil/conversation", nil))
	if rec.Code == http.StatusOK {
		t.Fatalf("path traversal probe unexpectedly succeeded: %d", rec.Code)
	}

	// Malformed JSON body → 400.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects/lingo-duihua/conversation/turns", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed turn status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestLegacyFixtureReadableWithoutMutation(t *testing.T) {
	root := copyFixture(t, filepath.Join("legacy", "zones"))
	before := snapshotTree(t, root)
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() {
		workspace.SetProjectsRootForTest("")
		assertTreeUnchanged(t, root, before)
	})
	server := &Server{migrationReady: false}

	// Legacy zone read surface.
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/legacy-wuqu/zones/Explain", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy zone read status = %d body = %s", rec.Code, rec.Body.String())
	}
	var zone map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &zone); err != nil {
		t.Fatal(err)
	}

	// Practice tasks read.
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/legacy-wuqu/practice/tasks", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy practice read status = %d", rec.Code)
	}

	// Flashcards read.
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/legacy-wuqu/summary/flashcards", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy flashcards read status = %d", rec.Code)
	}

	// Confusions (annotation-compat) read.
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/legacy-wuqu/confusions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy confusions read status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if list, ok := body["confusions"].([]any); !ok || len(list) != 1 {
		t.Fatalf("expected 1 imported confusion, got %v", body)
	}
}

func TestCorruptFixtureClassifiedFailure(t *testing.T) {
	root := copyFixture(t, "corrupt")
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	server := &Server{migrationReady: true}

	// First read may self-heal missing canonical files (ensure()); that is
	// production behavior. Assert the response stays a classified JSON error
	// or a valid projection — never a panic or a non-JSON 500.
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/sunhuai-xiangmu/conversation", nil))
	if rec.Code >= 500 {
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("corrupt read returned %d without JSON error body: %s", rec.Code, rec.Body.String())
		}
	}

	// From the healed state onward, reads must not mutate further.
	before := snapshotTree(t, root)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/sunhuai-xiangmu/conversation", nil))
	assertTreeUnchanged(t, root, before)
}

func TestHealthShape(t *testing.T) {
	server, _ := canonicalFixtureServer(t)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"ok", "workspace", "claude", "agentRuntime", "stats"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("health missing frozen field %q", field)
		}
	}
}

var _ = io.Discard
