# Iteration 06 — Phase B (Backend): Ask-AI API — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the backend half of Ask-AI: multi-vendor provider config in `config.local.json`, OpenAI-compatible + Anthropic streaming/summary clients, settings endpoints, and confusion-tied streaming + summarize endpoints that persist the exchange and emit `confusion-updated`.

**Architecture:** New packages `askaiconfig` (load/save `askAiProviders`, raw-JSON merge to preserve other keys) and `askaiprovider` (HTTP clients that normalize OpenAI/Anthropic SSE into `{type,content}` frames, plus a non-streaming `Complete` for summaries). The confusion store gains an optional `Ask` field (messages + summary + summaryState). Three new route handlers wire it up; the stream handler is request-scoped (`text/event-stream` on the response, not on the app-wide bus), persists the assistant reply on `done`, and the summarize handler runs in a goroutine emitting `confusion-updated` on completion. Independently testable with `httptest` + tempdirs — no frontend needed.

**Tech Stack:** Go 1.25.0 (`github.com/xmz14/lll`), net/http, encoding/json, `net/http/httptest` for tests. Reuses `internal/httpx` (`ReadJSON`/`WriteJSON`/`Error`), `internal/workspace` (`AtomicWriteFile`, `ProjectRootForSlug`, `ValidateSlug`, `SetProjectsRootForTest`), `internal/confusionstore`, `internal/paths`.

## Global Constraints

- File-first, no DB. Ask-AI exchanges persist as an optional `ask` field on a confusion in `<projectRoot>/explain/confusions.json` (atomic write).
- API keys live in `config.local.json` (gitignored, already holds `imageApiKey`). `GET /api/settings/ask-ai` MUST NOT return plaintext keys — mask as `"••••"`.
- The stream endpoint is request-scoped (`text/event-stream` on the HTTP response). Chat tokens are NOT broadcast on the app-wide SSE bus.
- Normalized frame shape consumed by the frontend: `{ "type": "text"|"thinking"|"done"|"error", "content": "..." }`.
- OpenAI-compatible endpoints: POST `<baseURL>/chat/completions`, `Authorization: Bearer <key>`, deltas `choices[].delta.content` (+ `reasoning_content` when `reasoning=true`), `[DONE]` terminator.
- Anthropic endpoints: POST `<baseURL>/v1/messages`, `x-api-key` + `anthropic-version: 2023-06-01`, events `content_block_delta` (`text_delta`/`thinking_delta`) + `message_stop`; `thinking` enabled via `{type:"enabled",budget_tokens}`.
- Backend tests: `go test ./backend-go/...`. Run from repo root.
- Module root is repo root (`D:/2_Study/lll/go.mod`).
- The package-global `broadcaster` (`backend-go/internal/server/router.go:34`) is reused for `confusion-updated`.

## File Structure

- Create `backend-go/internal/askaiconfig/askaiconfig.go` (+ `_test.go`) — config load/save.
- Create `backend-go/internal/askaiprovider/askaiprovider.go` — types + dispatch.
- Create `backend-go/internal/askaiprovider/openai.go` (+ `_test.go`) — OpenAI-compatible stream + complete.
- Create `backend-go/internal/askaiprovider/anthropic.go` (+ `_test.go`) — Anthropic stream + complete.
- Modify `backend-go/internal/confusionstore/store.go` (+ extend `_test.go`) — `Ask` field + `AppendAskMessage`/`SetAskSummary`.
- Create `backend-go/internal/server/routes_askai.go` — settings + stream + summarize handlers.
- Modify `backend-go/internal/server/router.go` — register the 5 new routes.

---

### Task 1: `askaiconfig` package — load + save with raw-JSON merge (TDD)

**Files:**
- Create: `backend-go/internal/askaiconfig/askaiconfig.go`
- Test: `backend-go/internal/askaiconfig/askaiconfig_test.go`

**Interfaces:**
- Produces: `type Provider`, `type Config`, `func Load() (*Config, error)` (nil,nil when absent), `func Save(Config) error`, `(c *Config) Enabled() bool`, `(c *Config) Find(id string) *Provider`. A package var `pathFn` lets tests redirect the config file.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/askaiconfig/askaiconfig_test.go
package askaiconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func useTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.local.json")
	old := pathFn
	pathFn = func() string { return path }
	t.Cleanup(func() { pathFn = old })
	return path
}

func TestLoadAbsentReturnsNil(t *testing.T) {
	useTempConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when file absent, got %+v", cfg)
	}
}

func TestLoadParsesProviders(t *testing.T) {
	path := useTempConfig(t)
	os.WriteFile(path, []byte(`{"imageApiKey":"keep-me","askAiProviders":{
		"default":"deepseek","searchEngine":"google",
		"providers":[{"id":"deepseek","kind":"openai","baseURL":"https://x/v1","apiKey":"sk-1","model":"m","reasoning":true}]}}`), 0o644)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.Default != "deepseek" || len(cfg.Providers) != 1 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.Providers[0].APIKey != "sk-1" || !cfg.Providers[0].Reasoning {
		t.Errorf("provider not parsed: %+v", cfg.Providers[0])
	}
	if !cfg.Enabled() {
		t.Errorf("expected Enabled=true")
	}
}

func TestFindByIDThenDefaultThenFirst(t *testing.T) {
	cfg := &Config{Default: "b", Providers: []Provider{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if cfg.Find("c").ID != "c" {
		t.Errorf("Find by id failed")
	}
	if cfg.Find("").ID != "b" {
		t.Errorf("Find default failed")
	}
	cfg.Default = "zzz"
	if cfg.Find("").ID != "a" {
		t.Errorf("Find fallback-to-first failed")
	}
}

func TestSavePreservesOtherKeys(t *testing.T) {
	path := useTempConfig(t)
	os.WriteFile(path, []byte(`{"imageApiKey":"keep-me","agentRuntime":"claude"}`), 0o644)
	if err := Save(Config{Default: "p1", SearchEngine: "bing", Providers: []Provider{{ID: "p1", Kind: "openai", APIKey: "sk"}}}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg == nil || cfg.Default != "p1" || cfg.SearchEngine != "bing" {
		t.Fatalf("round-trip failed: %+v %v", cfg, err)
	}
	// Other keys preserved?
	raw, _ := os.ReadFile(path)
	s := string(raw)
	if !contains(s, "\"imageApiKey\":") || !contains(s, "\"keep-me\"") || !contains(s, "\"agentRuntime\":") {
		t.Errorf("Save clobbered other keys: %s", s)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/askaiconfig/`
Expected: build failure — package/types undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// backend-go/internal/askaiconfig/askaiconfig.go
// Package askaiconfig loads/saves the Ask-AI provider config in config.local.json.
package askaiconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Provider is one configured Ask-AI model source.
type Provider struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`      // "openai" | "anthropic"
	Name      string `json:"name"`
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	Reasoning bool   `json:"reasoning,omitempty"` // openai-compatible: parse reasoning_content
	Thinking  bool   `json:"thinking,omitempty"`  // anthropic: extended thinking
}

// Config is the askAiProviders section.
type Config struct {
	Default      string     `json:"default"`
	SearchEngine string     `json:"searchEngine"` // "google" | "bing"
	Providers    []Provider `json:"providers"`
}

// Enabled reports whether at least one provider with key+baseURL+model exists.
func (c *Config) Enabled() bool {
	if c == nil {
		return false
	}
	for _, p := range c.Providers {
		if p.APIKey != "" && p.BaseURL != "" && p.Model != "" {
			return true
		}
	}
	return false
}

// Find returns the provider with id, else the default, else the first, else nil.
func (c *Config) Find(id string) *Provider {
	if c == nil {
		return nil
	}
	if id != "" {
		for i := range c.Providers {
			if c.Providers[i].ID == id {
				return &c.Providers[i]
			}
		}
	}
	for i := range c.Providers {
		if c.Providers[i].ID == c.Default {
			return &c.Providers[i]
		}
	}
	if len(c.Providers) > 0 {
		return &c.Providers[0]
	}
	return nil
}

// pathFn resolves the config file path. It is a var so tests can redirect it.
var pathFn = func() string {
	base := paths.WORKSPACE
	if base == "" {
		base = paths.PROJECT_ROOT
	}
	return filepath.Join(base, "config.local.json")
}

// UseConfigPathForTest redirects the config file path and returns a restore
// function. For tests in OTHER packages (e.g. server handler tests) that
// cannot touch the unexported pathFn directly.
func UseConfigPathForTest(path string) (restore func()) {
	old := pathFn
	pathFn = func() string { return path }
	return func() { pathFn = old }
}

// Load reads the askAiProviders section. Returns (nil, nil) when the file or
// section is absent (feature disabled). Returns an error only on parse failure.
func Load() (*Config, error) {
	data, err := os.ReadFile(pathFn())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	section, ok := raw["askAiProviders"]
	if !ok {
		return nil, nil
	}
	var cfg Config
	if err := json.Unmarshal(section, &cfg); err != nil {
		return nil, err
	}
	cfg.Default = strings.TrimSpace(cfg.Default)
	cfg.SearchEngine = strings.TrimSpace(cfg.SearchEngine)
	for i := range cfg.Providers {
		cfg.Providers[i].ID = strings.TrimSpace(cfg.Providers[i].ID)
		cfg.Providers[i].BaseURL = strings.TrimSpace(cfg.Providers[i].BaseURL)
		cfg.Providers[i].APIKey = strings.TrimSpace(cfg.Providers[i].APIKey)
		cfg.Providers[i].Model = strings.TrimSpace(cfg.Providers[i].Model)
	}
	return &cfg, nil
}

// Save writes the askAiProviders section back to config.local.json, preserving
// all other top-level keys via a raw-JSON round-trip. Atomic write.
func Save(cfg Config) error {
	var raw map[string]any
	if data, err := os.ReadFile(pathFn()); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &raw)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	raw["askAiProviders"] = cfg
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(pathFn(), out, 0o644)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend-go/internal/askaiconfig/`
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/askaiconfig/askaiconfig.go backend-go/internal/askaiconfig/askaiconfig_test.go
git commit -m "feat(askaiconfig): load/save ask-ai providers with raw-merge"
```

---

### Task 2: `askaiprovider` types + OpenAI-compatible client (TDD)

**Files:**
- Create: `backend-go/internal/askaiprovider/askaiprovider.go`
- Create: `backend-go/internal/askaiprovider/openai.go`
- Test: `backend-go/internal/askaiprovider/openai_test.go`

**Interfaces:**
- Produces: `type Provider`, `type Message`, `type Frame`, `func Stream(ctx, Provider, system string, msgs []Message, onFrame func(Frame)) error`, `func Complete(ctx, Provider, system string, msgs []Message) (string, error)`. OpenAI client: `POST <baseURL>/chat/completions`.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/askaiprovider/openai_test.go
package askaiprovider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamOpenAINormalizes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var got []Frame
	err := Stream(context.Background(), Provider{Kind: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"}, "", []Message{{Role: "user", Content: "hi"}}, func(f Frame) { got = append(got, f) })
	if err != nil {
		t.Fatal(err)
	}
	if !framesContain(got, Frame{Type: "text", Content: "Hel"}) || !framesContain(got, Frame{Type: "text", Content: "lo"}) {
		t.Errorf("missing text frames: %+v", got)
	}
	if !framesContain(got, Frame{Type: "done"}) {
		t.Errorf("missing done frame: %+v", got)
	}
}

func TestStreamOpenAIReasoning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"think\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ans\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var got []Frame
	_ = Stream(context.Background(), Provider{Kind: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m", Reasoning: true}, "", []Message{{Role: "user", Content: "hi"}}, func(f Frame) { got = append(got, f) })
	if !framesContain(got, Frame{Type: "thinking", Content: "think"}) {
		t.Errorf("missing thinking frame: %+v", got)
	}
}

func TestCompleteOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"pong"}}]}`)
	}))
	defer srv.Close()

	got, err := Complete(context.Background(), Provider{Kind: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"}, "", []Message{{Role: "user", Content: "ping"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "pong" {
		t.Errorf("got %q", got)
	}
}

func framesContain(fs []Frame, want Frame) bool {
	for _, f := range fs {
		if f == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/askaiprovider/`
Expected: build failure — types/functions undefined.

- [ ] **Step 3: Write the dispatch + OpenAI client**

```go
// backend-go/internal/askaiprovider/askaiprovider.go
// Package askaiprovider implements OpenAI-compatible and Anthropic streaming +
// non-streaming chat clients, normalized to one frame shape.
package askaiprovider

import (
	"context"
	"fmt"
)

// Provider is the connection config (mirrors askaiconfig.Provider).
type Provider struct {
	ID        string
	Kind      string // "openai" | "anthropic"
	Name      string
	BaseURL   string
	APIKey    string
	Model     string
	Reasoning bool
	Thinking  bool
}

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// Frame is the normalized streaming unit sent to the client.
type Frame struct {
	Type    string `json:"type"` // "text" | "thinking" | "done" | "error"
	Content string `json:"content"`
}

// Stream calls the provider and invokes onFrame for each normalized frame.
func Stream(ctx context.Context, p Provider, system string, msgs []Message, onFrame func(Frame)) error {
	switch p.Kind {
	case "anthropic":
		return streamAnthropic(ctx, p, system, msgs, onFrame)
	default:
		return streamOpenAI(ctx, p, system, msgs, onFrame)
	}
}

// Complete returns the full assistant text (non-streaming), for summaries.
func Complete(ctx context.Context, p Provider, system string, msgs []Message) (string, error) {
	switch p.Kind {
	case "anthropic":
		return completeAnthropic(ctx, p, system, msgs)
	default:
		return completeOpenAI(ctx, p, system, msgs)
	}
}

// asHTTPProvider is unused here but reserved for future injection points.
var _ = fmt.Sprint
```

```go
// backend-go/internal/askaiprovider/openai.go
package askaiprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func openAIBody(p Provider, system string, msgs []Message, stream bool) ([]byte, error) {
	out := make([]Message, 0, len(msgs)+1)
	if system != "" {
		out = append(out, Message{Role: "system", Content: system})
	}
	out = append(out, msgs...)
	return json.Marshal(map[string]any{
		"model":    p.Model,
		"messages": out,
		"stream":   stream,
	})
}

func streamOpenAI(ctx context.Context, p Provider, system string, msgs []Message, onFrame func(Frame)) error {
	body, err := openAIBody(p, system, msgs, true)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai: status %d: %s", resp.StatusCode, string(b))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			onFrame(Frame{Type: "done"})
			return nil
		}
		if payload == "" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, ch := range chunk.Choices {
			if p.Reasoning && ch.Delta.ReasoningContent != "" {
				onFrame(Frame{Type: "thinking", Content: ch.Delta.ReasoningContent})
			}
			if ch.Delta.Content != "" {
				onFrame(Frame{Type: "text", Content: ch.Delta.Content})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	onFrame(Frame{Type: "done"})
	return nil
}

func completeOpenAI(ctx context.Context, p Provider, system string, msgs []Message) (string, error) {
	body, err := openAIBody(p, system, msgs, false)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai: status %d: %s", resp.StatusCode, string(b))
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if len(res.Choices) == 0 {
		return "", nil
	}
	return res.Choices[0].Message.Content, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend-go/internal/askaiprovider/`
Expected: 3 tests PASS. (Anthropic dispatch returns an error since `anthropic.go` doesn't exist yet — but no test exercises that path here. The build must still succeed, so Task 3 must land before any anthropic test runs. This task's tests only cover openai, so they pass now.)

> Note: because `streamAnthropic`/`completeAnthropic` are referenced in `askaiprovider.go`, the package will NOT compile until Task 3 adds `anthropic.go`. Either implement Task 3 before running tests, or temporarily stub the two functions. The recommended order is: write Task 2 files AND Task 3's `anthropic.go` together, then run tests for both. The tasks are split for review clarity, not for independent compilation.

- [ ] **Step 5: Commit** (after Task 3 compiles)

```bash
git add backend-go/internal/askaiprovider/
git commit -m "feat(askaiprovider): openai-compatible + anthropic streaming clients"
```
(This commit is shared with Task 3 since the package only compiles with both files.)

---

### Task 3: Anthropic client (TDD)

**Files:**
- Create: `backend-go/internal/askaiprovider/anthropic.go`
- Test: `backend-go/internal/askaiprovider/anthropic_test.go`

**Interfaces:**
- Produces: `streamAnthropic`, `completeAnthropic` (package-private, called by the dispatch in Task 2). Anthropic client: `POST <baseURL>/v1/messages`, `x-api-key` + `anthropic-version: 2023-06-01`.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/askaiprovider/anthropic_test.go
package askaiprovider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamAnthropicNormalizes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"hmm\"}}\n\n")
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	var got []Frame
	err := Stream(context.Background(), Provider{Kind: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", Thinking: true}, "", []Message{{Role: "user", Content: "hi"}}, func(f Frame) { got = append(got, f) })
	if err != nil {
		t.Fatal(err)
	}
	if !framesContain(got, Frame{Type: "thinking", Content: "hmm"}) {
		t.Errorf("missing thinking frame: %+v", got)
	}
	if !framesContain(got, Frame{Type: "text", Content: "Hi"}) {
		t.Errorf("missing text frame: %+v", got)
	}
	if !framesContain(got, Frame{Type: "done"}) {
		t.Errorf("missing done frame: %+v", got)
	}
}

func TestCompleteAnthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"content":[{"type":"text","text":"pong"}]}`)
	}))
	defer srv.Close()

	got, err := Complete(context.Background(), Provider{Kind: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m"}, "", []Message{{Role: "user", Content: "ping"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "pong" {
		t.Errorf("got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/askaiprovider/`
Expected: build failure — `streamAnthropic`/`completeAnthropic` undefined (and the package doesn't compile).

- [ ] **Step 3: Write minimal implementation**

```go
// backend-go/internal/askaiprovider/anthropic.go
package askaiprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func anthropicBody(p Provider, system string, msgs []Message, stream bool) ([]byte, error) {
	req := map[string]any{
		"model":      p.Model,
		"max_tokens": 4096,
		"stream":     stream,
		"messages":   msgs,
	}
	if system != "" {
		req["system"] = system
	}
	if p.Thinking {
		req["thinking"] = map[string]any{"type": "enabled", "budget_tokens": 1024}
		req["max_tokens"] = 5120 // must exceed budget_tokens
	}
	return json.Marshal(req)
}

func streamAnthropic(ctx context.Context, p Provider, system string, msgs []Message, onFrame func(Frame)) error {
	body, err := anthropicBody(p, system, msgs, true)
	if err != nil {
		return err
	}
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, string(b))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		var ev struct {
			Type  string          `json:"type"`
			Delta json.RawMessage `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "content_block_delta":
			var d struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				Thinking string `json:"thinking"`
			}
			if json.Unmarshal(ev.Delta, &d) == nil {
				switch d.Type {
				case "thinking_delta":
					if p.Thinking && d.Thinking != "" {
						onFrame(Frame{Type: "thinking", Content: d.Thinking})
					}
				case "text_delta":
					if d.Text != "" {
						onFrame(Frame{Type: "text", Content: d.Text})
					}
				}
			}
		case "message_stop":
			onFrame(Frame{Type: "done"})
			return nil
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	onFrame(Frame{Type: "done"})
	return nil
}

func completeAnthropic(ctx context.Context, p Provider, system string, msgs []Message) (string, error) {
	// Disable thinking for a concise summary.
	p.Thinking = false
	body, err := anthropicBody(p, system, msgs, false)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, string(b))
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, b := range res.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String(), nil
}
```

- [ ] **Step 4: Run all provider tests**

Run: `go test ./backend-go/internal/askaiprovider/`
Expected: openai + anthropic tests (5 total) PASS; package compiles.

- [ ] **Step 5: Commit** (with Task 2)

```bash
git add backend-go/internal/askaiprovider/
git commit -m "feat(askaiprovider): openai-compatible + anthropic streaming clients"
```

---

### Task 4: Extend `confusionstore` with the `Ask` exchange (TDD)

**Files:**
- Modify: `backend-go/internal/confusionstore/store.go` (struct + new methods)
- Test: `backend-go/internal/confusionstore/ask_test.go` (new)

**Interfaces:**
- Produces: `type Ask`, `type AskMessage`, field `Confusion.Ask *Ask`, `func (s *Store) AppendAskMessage(id string, msg AskMessage) (Confusion, error)`, `func (s *Store) SetAskSummary(id, summary, state string) (Confusion, error)`. `SetAskSummary` with `state=="done"` flips `Confusion.State` to `StateAsked`.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/confusionstore/ask_test.go
package confusionstore

import (
	"testing"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func useTempProjectsRoot(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	t.Cleanup(func() { workspace.SetProjectsRootForTest(old) })
}

func TestAppendAskMessageAndSummary(t *testing.T) {
	useTempProjectsRoot(t)
	s, err := New("proj")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Create(Confusion{QuoteSnapshot: "q"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendAskMessage(c.ID, AskMessage{Role: "user", Content: "why?"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendAskMessage(c.ID, AskMessage{Role: "assistant", Content: "because"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ask == nil || len(got.Ask.Messages) != 2 || got.Ask.Messages[1].Content != "because" {
		t.Fatalf("messages not stored: %+v", got.Ask)
	}

	updated, err := s.SetAskSummary(c.ID, "a short summary", "done")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Ask.Summary != "a short summary" || updated.Ask.SummaryState != "done" {
		t.Errorf("summary not stored: %+v", updated.Ask)
	}
	if updated.State != StateAsked {
		t.Errorf("state should be asked after done summary, got %s", updated.State)
	}

	// Reload to confirm persistence.
	s2, _ := New("proj")
	again, _ := s2.Get(c.ID)
	if again.Ask == nil || again.Ask.Summary != "a short summary" {
		t.Errorf("not persisted: %+v", again.Ask)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/confusionstore/`
Expected: build failure — `Ask`/`AskMessage`/`AppendAskMessage`/`SetAskSummary` undefined. (If `workspace.ProjectsRootForTest`/`SetProjectsRootForTest` don't exist with those exact names, see the Note below.)

> Note on the projects-root test helpers: `backend-go/internal/workspace/workspace.go` exposes `ProjectsRootForTest()` and `SetProjectsRootForTest(dir string)` (lines ~767–770). If a build error says otherwise, read that file and use the exact names it exports. The pattern (redirect the projects root to a tempdir) is already used by `promptassembly` tests.

- [ ] **Step 3: Add the types + methods**

In `backend-go/internal/confusionstore/store.go`, add these types (after the `Confusion` struct):

```go
// AskMessage is one turn of an Ask-AI exchange attached to a confusion.
type AskMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// Ask is the persisted Ask-AI exchange for a confusion.
type Ask struct {
	Messages     []AskMessage `json:"messages,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	SummaryState string       `json:"summaryState,omitempty"` // idle | pending | done | failed
	ProviderID   string       `json:"providerId,omitempty"`
	UpdatedAt    string       `json:"updatedAt,omitempty"`
}
```

Add the field to `Confusion` (after `State State`):

```go
	State            State  `json:"state"`
	CreatedAt        string `json:"createdAt"`
	Ask              *Ask   `json:"ask,omitempty"`
```

Add these methods (after `Get`):

```go
// AppendAskMessage appends a message to the confusion's ask exchange (creating
// the Ask object on first use) and persists. New messages reset SummaryState to
// "idle" so a stale summary is not shown after a new turn.
func (s *Store) AppendAskMessage(id string, msg AskMessage) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			if c.Ask == nil {
				c.Ask = &Ask{}
			}
			if msg.ID == "" {
				msg.ID = newID()
			}
			if msg.CreatedAt == "" {
				msg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			c.Ask.Messages = append(c.Ask.Messages, msg)
			c.Ask.SummaryState = "idle"
			c.Ask.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			s.data[i] = c
			s.dirty = true
			return c, s.save()
		}
	}
	return Confusion{}, os.ErrNotExist
}

// SetAskSummary stores the summary and state for a confusion's ask exchange.
// state "done" flips the confusion to StateAsked.
func (s *Store) SetAskSummary(id, summary, state string) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			if c.Ask == nil {
				c.Ask = &Ask{}
			}
			c.Ask.Summary = summary
			c.Ask.SummaryState = state
			c.Ask.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if state == "done" {
				c.State = StateAsked
			}
			s.data[i] = c
			s.dirty = true
			return c, s.save()
		}
	}
	return Confusion{}, os.ErrNotExist
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend-go/internal/confusionstore/`
Expected: PASS (including the new test + existing store tests).

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/confusionstore/store.go backend-go/internal/confusionstore/ask_test.go
git commit -m "feat(confusionstore): add Ask exchange (messages + summary)"
```

---

### Task 5: Ask-AI settings endpoints (GET/PUT/probe)

**Files:**
- Create: `backend-go/internal/server/routes_askai.go`
- Modify: `backend-go/internal/server/router.go` (register 3 routes)

**Interfaces:**
- Consumes: `askaiconfig.Load/Save`, `askaiconfig.Config.Enabled/Find`, `askaiprovider.Complete`.
- Produces: handlers `handleGetAskAiSettings`, `handlePutAskAiSettings`, `handleProbeAskAi` and routes `GET/PUT /api/settings/ask-ai`, `POST /api/settings/ask-ai/probe`. GET masks keys as `"••••"`; PUT preserves existing keys when the client sends the mask back.

- [ ] **Step 1: Write the handlers**

```go
// backend-go/internal/server/routes_askai.go
package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/httpx"
)

const maskedKey = "••••"

func maskProviders(ps []askaiconfig.Provider) []askaiconfig.Provider {
	out := make([]askaiconfig.Provider, len(ps))
	for i, p := range ps {
		if p.APIKey != "" {
			p.APIKey = maskedKey
		}
		out[i] = p
	}
	return out
}

func (s *Server) handleGetAskAiSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := askaiconfig.Load()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg == nil {
		cfg = &askaiconfig.Config{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"default":      cfg.Default,
		"searchEngine": cfg.SearchEngine,
		"providers":    maskProviders(cfg.Providers),
	})
}

func (s *Server) handlePutAskAiSettings(w http.ResponseWriter, r *http.Request) {
	var in askaiconfig.Config
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	// Preserve real keys for providers the client echoed back masked.
	old, _ := askaiconfig.Load()
	oldByKey := map[string]askaiconfig.Provider{}
	if old != nil {
		for _, p := range old.Providers {
			oldByKey[p.ID] = p
		}
	}
	for i, p := range in.Providers {
		if p.APIKey == maskedKey {
			if prev, ok := oldByKey[p.ID]; ok {
				in.Providers[i].APIKey = prev.APIKey
			}
		}
	}
	if err := askaiconfig.Save(in); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProbeAskAi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProviderID string                `json:"providerId"`
		Inline     *askaiconfig.Provider `json:"inline"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	var pc askaiconfig.Provider
	if in.Inline != nil {
		pc = *in.Inline
	} else {
		cfg, err := askaiconfig.Load()
		if err != nil || cfg == nil {
			httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
			return
		}
		p := cfg.Find(in.ProviderID)
		if p == nil {
			httpx.Error(w, http.StatusBadRequest, "provider not found")
			return
		}
		pc = *p
	}
	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}
	_, err := askaiprovider.Complete(r.Context(), prov, "Reply with the single word: ok", []askaiprovider.Message{{Role: "user", Content: "ping"}})
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
```

- [ ] **Step 2: Register the routes**

In `backend-go/internal/server/router.go`, inside `Handler()`, after the existing agent-runtime settings routes (around line 91), add:

```go
	mux.HandleFunc("GET /api/settings/ask-ai", s.handleGetAskAiSettings)
	mux.HandleFunc("PUT /api/settings/ask-ai", s.handlePutAskAiSettings)
	mux.HandleFunc("POST /api/settings/ask-ai/probe", s.handleProbeAskAi)
```

- [ ] **Step 3: Verify build + run backend tests**

Run:
```bash
go build ./backend-go/...
go test ./backend-go/...
```
Expected: build OK; existing tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add backend-go/internal/server/routes_askai.go backend-go/internal/server/router.go
git commit -m "feat(server): ask-ai settings endpoints (get/put/probe)"
```

---

### Task 6: Ask-AI streaming endpoint (`ask-stream`)

**Files:**
- Modify: `backend-go/internal/server/routes_askai.go` (append handler + helper)
- Modify: `backend-go/internal/server/router.go` (register route)
- Test: `backend-go/internal/server/routes_askai_test.go` (new)

**Interfaces:**
- Consumes: `confusionstore.New/Get/AppendAskMessage`, `askaiconfig.Load/Enabled/Find`, `askaiprovider.Stream`, `workspace.ProjectRootForSlug/ValidateSlug`, `workspace.AtomicWriteFile`, `httpx.ReadJSON/Error`.
- Produces: `POST /api/projects/{id}/confusions/{confusionId}/ask-stream` returning `text/event-stream` of normalized frames; appends user message on entry and assistant message on `done`. `pageArtifactId` (project-relative path like `explain/pages/01-x.md`) optionally loads page context into the system prompt.

- [ ] **Step 1: Write the failing test**

```go
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
```

> Note: this handler uses the package-global `broadcaster` only indirectly (it does not emit on the stream path). `&Server{}` is sufficient because the handler reads config + confusion store off package globals/paths. If `handleAskAiStream` later needs a Server field, update `newTestServer`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/server/ -run TestAskAiStreamPersistsAndStreams`
Expected: build failure — `handleAskAiStream` undefined.

- [ ] **Step 3: Write the handler + helper**

Append to `backend-go/internal/server/routes_askai.go` (add imports `context`, `encoding/json`, `fmt`, `os`, `path/filepath`, `strings`, `time`, `workspace`, `confusionstore` as needed):

```go
func (s *Server) handleAskAiStream(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	cid := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var in struct {
		ProviderID     string `json:"providerId"`
		Content        string `json:"content"`
		PageArtifactID string `json:"pageArtifactId"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conf, err := store.Get(cid)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	cfg, err := askaiconfig.Load()
	if err != nil || !cfg.Enabled() {
		httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
		return
	}
	pc := cfg.Find(in.ProviderID)
	if pc == nil {
		httpx.Error(w, http.StatusBadRequest, "provider not found")
		return
	}

	// Append the user turn immediately so it persists even if the stream aborts.
	if _, err := store.AppendAskMessage(cid, confusionstore.AskMessage{Role: "user", Content: in.Content}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build the message history (prior turns + this user turn).
	var msgs []askaiprovider.Message
	if conf.Ask != nil {
		for _, m := range conf.Ask.Messages {
			msgs = append(msgs, askaiprovider.Message{Role: m.Role, Content: m.Content})
		}
	}
	msgs = append(msgs, askaiprovider.Message{Role: "user", Content: in.Content})

	system := ""
	if in.PageArtifactID != "" {
		if ctx, err := loadPageContext(slug, in.PageArtifactID); err == nil {
			system = "You are helping a learner studying the following page. Answer in context.\n\n" + ctx
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}
	var sb strings.Builder
	writeFrame := func(f askaiprovider.Frame) {
		b, _ := json.Marshal(f)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		if f.Type == "text" {
			sb.WriteString(f.Content)
		}
	}
	if err := askaiprovider.Stream(r.Context(), prov, system, msgs, writeFrame); err != nil {
		writeFrame(askaiprovider.Frame{Type: "error", Content: err.Error()})
		return
	}
	// Persist the assistant reply (best-effort).
	_, _ = store.AppendAskMessage(cid, confusionstore.AskMessage{Role: "assistant", Content: sb.String()})
}

// loadPageContext reads a project-relative artifact (e.g. "explain/pages/01.md")
// capped to keep prompts small.
func loadPageContext(slug, pageArtifactID string) (string, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pageArtifactID)))
	if err != nil {
		return "", err
	}
	s := string(data)
	const cap = 6000
	if len(s) > cap {
		s = s[:cap] + "\n…(truncated)"
	}
	return s, nil
}
```

Add the imports to the `routes_askai.go` import block: `"context"` is unused here (remove if the linter complains — actually it is not used in this task; only Task 7 uses context). Add only what each task uses: this task needs `"encoding/json"`, `"fmt"`, `"os"`, `"path/filepath"`, `"strings"`, plus `"github.com/xmz14/lll/backend-go/internal/confusionstore"` and `"github.com/xmz14/lll/backend-go/internal/workspace"`.

- [ ] **Step 4: Register the route**

In `router.go` `Handler()`, after the confusions routes (around line 121), add:

```go
	mux.HandleFunc("POST /api/projects/{id}/confusions/{confusionId}/ask-stream", s.handleAskAiStream)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./backend-go/internal/server/ -run TestAskAiStream`
Expected: PASS — streamed frames include `text` + `done`, assistant reply persisted.

- [ ] **Step 6: Commit**

```bash
git add backend-go/internal/server/routes_askai.go backend-go/internal/server/router.go backend-go/internal/server/routes_askai_test.go
git commit -m "feat(server): ask-ai streaming endpoint tied to a confusion"
```

---

### Task 7: Ask-AI summarize endpoint (`ask/summarize`)

**Files:**
- Modify: `backend-go/internal/server/routes_askai.go` (append handler)
- Modify: `backend-go/internal/server/router.go` (register route)

**Interfaces:**
- Consumes: `confusionstore.New/Get/SetAskSummary`, `askaiconfig.Load/Enabled/Find`, `askaiprovider.Complete`, `broadcaster.Emit`.
- Produces: `POST /api/projects/{id}/confusions/{confusionId}/ask/summarize` → `202 {status:"pending"}`. Sets `summaryState=pending`, then in a goroutine generates the summary (`<=250` chars prompt) from `quoteSnapshot + ask.messages`, stores it (`done`/`failed`), flips the confusion to `StateAsked` on `done`, and emits `confusion-updated {projectSlug, action:"summarize", id}` so the sidebar refreshes.

- [ ] **Step 1: Write the handler**

Append to `routes_askai.go` (add `"context"`, `"time"` to imports):

```go
func (s *Server) handleAskAiSummarize(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	cid := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conf, err := store.Get(cid)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}

	// Mark pending immediately + notify (sidebar shows "生成总结中...").
	store.SetAskSummary(cid, "", "pending")
	broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})

	go summarizeAskExchange(slug, cid, conf.QuoteSnapshot, conf.Ask)

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "pending"})
}

func summarizeAskExchange(slug, cid, quote string, ask *confusionstore.Ask) {
	cfg, err := askaiconfig.Load()
	if err != nil || !cfg.Enabled() {
		store, _ := confusionstore.New(slug)
		store.SetAskSummary(cid, "总结生成失败：未配置 Ask-AI 模型源。", "failed")
		broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
		return
	}
	pc := cfg.Find("")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}

	system := "结合用户的疑问点和下面的对话，生成一段不超过 250 个汉字的中文总结，帮助用户日后回忆这次答疑的结论。直接输出总结正文，不要寒暄或多余说明。"
	var msgs []askaiprovider.Message
	msgs = append(msgs, askaiprovider.Message{Role: "user", Content: "疑问原文：" + quote})
	if ask != nil {
		for _, m := range ask.Messages {
			msgs = append(msgs, askaiprovider.Message{Role: m.Role, Content: m.Content})
		}
	}
	summary, err := askaiprovider.Complete(ctx, prov, system, msgs)
	store, _ := confusionstore.New(slug)
	if err != nil || strings.TrimSpace(summary) == "" {
		store.SetAskSummary(cid, "总结生成失败。", "failed")
	} else {
		store.SetAskSummary(cid, strings.TrimSpace(summary), "done")
	}
	broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
}
```

- [ ] **Step 2: Register the route**

In `router.go` `Handler()`, after the ask-stream route, add:

```go
	mux.HandleFunc("POST /api/projects/{id}/confusions/{confusionId}/ask/summarize", s.handleAskAiSummarize)
```

- [ ] **Step 3: Verify build + tests**

Run:
```bash
go build ./backend-go/...
go test ./backend-go/...
```
Expected: build OK; all backend tests PASS.

- [ ] **Step 4: Commit**

```bash
git add backend-go/internal/server/routes_askai.go backend-go/internal/server/router.go
git commit -m "feat(server): ask-ai close->summarize endpoint"
```

---

## End-to-End Smoke (after all 7 tasks)

1. `go run ./backend-go/cmd/lll`.
2. Add a provider to `config.local.json` manually (or via the frontend settings page once Phase B-frontend lands):
   ```jsonc
   { "askAiProviders": { "default": "deepseek", "searchEngine": "google",
     "providers": [{ "id": "deepseek", "kind": "openai", "baseURL": "https://api.deepseek.com/v1", "apiKey": "sk-...", "model": "deepseek-chat" }] } }
   ```
3. `curl -s http://localhost:8787/api/settings/ask-ai` → keys masked.
4. `curl -s -X POST http://localhost:8787/api/settings/ask-ai/probe -H 'Content-Type: application/json' -d '{"providerId":"deepseek"}'` → `{"ok":true}`.
5. Create a confusion (via the existing `POST /api/projects/{id}/confusions`), then:
   `curl -N -X POST http://localhost:8787/api/projects/<slug>/confusions/<cid>/ask-stream -H 'Content-Type: application/json' -d '{"content":"用一句话解释这段"}'` → streamed `data: {"type":"text",...}` frames ending in `done`.
6. `curl -s -X POST http://localhost:8787/api/projects/<slug>/confusions/<cid>/ask/summarize` → `{"status":"pending"}`; a `confusion-updated` SSE event fires on `/api/events` when the summary is stored. `GET .../confusions` then shows `state:"asked"` + `ask.summary`.

## Notes / Honest Limitations (Phase B backend)

- No auth on the ask-ai endpoints beyond same-origin (local-only server, consistent with the rest of LLL). The settings GET masks keys; PUT preserves them via the mask-echo convention.
- `askaiprovider` uses `http.DefaultClient` with the request context for streaming (client disconnect cancels). No retry/backoff (YAGNI for a local learner tool).
- The summarize goroutine uses a fresh 60s context (independent of the HTTP request lifecycle, since the request returns 202 immediately).
- Page-context loading caps at 6000 chars to bound prompt size.
