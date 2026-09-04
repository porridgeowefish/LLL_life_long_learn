// backend-go/internal/server/routes_askai_test.go
package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	annotationstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	"github.com/xmz14/lll/backend-go/internal/projectindex"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/teacherservice"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func setupAskAiTestProject(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	if err := workspace.CreateProjectSkeletonWithInput("proj", "测试", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	restoreCfg := askaiconfig.UseConfigPathForTest(filepath.Join(dir, "config.local.json"))
	t.Cleanup(func() {
		workspace.SetProjectsRootForTest(old)
		restoreCfg()
	})
}

func writeAskAiConfig(t *testing.T, p askaiconfig.Provider) {
	t.Helper()
	cfg := askaiconfig.Config{Default: p.ID, Providers: []askaiconfig.Provider{p}}
	if err := askaiconfig.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestAskAiStreamPersistsAndStreams(t *testing.T) {
	setupAskAiTestProject(t)
	var receivedSystem string
	// Fake OpenAI-compatible provider pointed at a stub SSE server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []askaiprovider.Message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if len(request.Messages) > 0 {
			receivedSystem = request.Messages[0].Content
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"secret-chain\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	writeAskAiConfig(t, askaiconfig.Provider{ID: "stub", Kind: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m", Reasoning: true})

	// Create a confusion to attach the ask exchange to.
	store, err := annotationstore.NewAnnotations("proj")
	if err != nil {
		t.Fatal(err)
	}
	conf, err := store.Create(annotationstore.CreateInput{QuoteSnapshot: "sel"})
	if err != nil {
		t.Fatal(err)
	}

	srv2 := newTestServer(t)
	body := `{"content":"why?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj/assets/body/annotations/"+conf.AnnotationID+"/ask-stream", strings.NewReader(body))
	req.SetPathValue("id", "proj")
	req.SetPathValue("confusionId", conf.AnnotationID)
	rec := httptest.NewRecorder()
	srv2.handleAskAiStream(rec, req)

	out := rec.Body.String()
	if !strings.Contains(out, `"type":"text"`) || !strings.Contains(out, "Hello") {
		t.Errorf("missing streamed text frame: %s", out)
	}
	if !strings.Contains(out, `"type":"done"`) {
		t.Errorf("missing done frame: %s", out)
	}
	if strings.Contains(out, "secret-chain") || strings.Contains(out, `"type":"thinking"`) {
		t.Errorf("raw thinking leaked to annotation stream: %s", out)
	}
	if !strings.Contains(receivedSystem, "sel") || !strings.Contains(receivedSystem, conf.AssetVersionID) {
		t.Errorf("annotation quote/version missing from system context: %s", receivedSystem)
	}

	// The assistant reply should be persisted.
	again, _ := annotationstore.NewAnnotations("proj")
	got, _ := again.Get(conf.AnnotationID)
	var assistant string
	for _, m := range got.Ask.Messages {
		if m.Role == "assistant" {
			assistant = m.Content
		}
	}
	if assistant != "Hello" {
		t.Errorf("assistant not persisted; got %q", assistant)
	}
}

type askRoundTripFunc func(*http.Request) (*http.Response, error)

func (f askRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type failingAskBody struct {
	data []byte
	done bool
}

func (b *failingAskBody) Read(target []byte) (int, error) {
	if !b.done {
		b.done = true
		return copy(target, b.data), nil
	}
	return 0, errors.New("stream disconnected")
}

func (b *failingAskBody) Close() error { return nil }

func TestAskAiStreamPersistsPartialFailure(t *testing.T) {
	setupAskAiTestProject(t)
	writeAskAiConfig(t, askaiconfig.Provider{ID: "stub", Kind: "openai", BaseURL: "http://provider.invalid", APIKey: "k", Model: "m"})
	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: askRoundTripFunc(func(*http.Request) (*http.Response, error) {
		body := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"Partial\"}}]}\n\n")
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failingAskBody{data: body}}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = oldClient })

	store, _ := annotationstore.NewAnnotations("proj")
	annotation, _ := store.Create(annotationstore.CreateInput{QuoteSnapshot: "selected"})
	request := httptest.NewRequest(http.MethodPost, "/api/projects/proj/assets/body/annotations/"+annotation.AnnotationID+"/ask-stream", bytes.NewBufferString(`{"content":"why?"}`))
	request.SetPathValue("id", "proj")
	request.SetPathValue("confusionId", annotation.AnnotationID)
	recorder := httptest.NewRecorder()
	newTestServer(t).handleAskAiStream(recorder, request)
	updated, err := store.Get(annotation.AnnotationID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, message := range updated.Ask.Messages {
		if message.Role == "assistant" && message.Content == "Partial" && message.Status == "failed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("partial failed answer was not persisted: %#v", updated.Ask)
	}
	if !strings.Contains(recorder.Body.String(), "已保留收到的部分内容") {
		t.Fatalf("safe partial failure frame missing: %s", recorder.Body.String())
	}
}

var _ io.ReadCloser = (*failingAskBody)(nil)

// newTestServer returns a *Server without probing Claude. Every handler-
// reachable field is wired because the hidden globals became struct fields.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		broadcaster:     httpx.NewBroadcaster(),
		teacher:         teacherservice.New(nil),
		activeTeacher:   map[string]*activeTeacherRun{},
		activeByProject: map[string]*activeTeacherRun{},
		runProgress:     runprogress.New(),
		sessions:        sessionstore.New(),
		cache:           projectindex.New(),
		migrationReady:  true,
	}
	s.agents = agentregistry.New()
	if err := s.agents.Load(); err != nil {
		t.Logf("agent registry load warning: %v", err)
	}
	return s
}
