// backend-go/internal/server/routes_askai_test.go
package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/confusionstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func setupAskAiTestProject(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
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
	// Fake OpenAI-compatible provider pointed at a stub SSE server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	writeAskAiConfig(t, askaiconfig.Provider{ID: "stub", Kind: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"})

	// Create a confusion to attach the ask exchange to.
	store, err := confusionstore.New("proj")
	if err != nil {
		t.Fatal(err)
	}
	conf, err := store.Create(confusionstore.Confusion{QuoteSnapshot: "sel"})
	if err != nil {
		t.Fatal(err)
	}

	srv2 := newTestServer(t)
	body := `{"content":"why?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj/confusions/"+conf.ID+"/ask-stream", strings.NewReader(body))
	req.SetPathValue("id", "proj")
	req.SetPathValue("confusionId", conf.ID)
	rec := httptest.NewRecorder()
	srv2.handleAskAiStream(rec, req)

	out := rec.Body.String()
	if !strings.Contains(out, `"type":"text"`) || !strings.Contains(out, "Hello") {
		t.Errorf("missing streamed text frame: %s", out)
	}
	if !strings.Contains(out, `"type":"done"`) {
		t.Errorf("missing done frame: %s", out)
	}

	// The assistant reply should be persisted.
	again, _ := confusionstore.New("proj")
	got, _ := again.Get(conf.ID)
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

// newTestServer returns a *Server without probing Claude.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{}
}
