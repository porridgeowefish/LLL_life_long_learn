package ocr

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	impl "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/store"
	platformconfig "github.com/xmz14/lll/backend-go/internal/platform/config"
)

func useOCRConfig(t *testing.T, baseURL string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.local.json")
	config := map[string]any{
		"ai": map[string]any{
			"defaultProvider": "vision",
			"providers": []map[string]any{{
				"id": "vision", "kind": "openai", "baseURL": baseURL, "apiKey": "sk-vision", "model": "glm-4v-plus",
			}},
			"bindings": map[string]any{
				"ocr": map[string]any{"providerId": "vision"},
			},
		},
	}
	raw, _ := json.Marshal(config)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	restore := platformconfig.UsePathForTest(path)
	t.Cleanup(restore)
}

func TestConfiguredRequiresExplicitBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	if err := os.WriteFile(path, []byte(`{"ai":{"defaultProvider":"p","providers":[{"id":"p","kind":"openai","baseURL":"https://x","apiKey":"k","model":"m"}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := platformconfig.UsePathForTest(path)
	defer restore()
	if Configured() {
		t.Fatal("ocr must require an explicit bindings.ocr entry")
	}
}

func TestProcessImageOCRCommitsDerivedMarkdown(t *testing.T) {
	var gotPayload map[string]any
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"# 讲义标题\n\n傅里叶变换：$F(\\omega)$"}}]}`))
	}))
	defer server.Close()
	useOCRConfig(t, server.URL)

	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("ocrproj", "图解", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, err := impl.New("ocrproj")
	if err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 1, 2, 3}
	source, revision, err := store.Add("讲义截图", "lecture.png", "image/png", int64(len(png)), strings.NewReader(string(png)), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessImageOCR("ocrproj", store, source.SourceID, revision.RevisionID); err != nil {
		t.Fatal(err)
	}
	after, afterRevision, err := store.Get(source.SourceID)
	if err != nil || after.Status != "ready" {
		t.Fatalf("source not ready: %#v err=%v", after, err)
	}
	content, err := store.ReadContent(source.SourceID)
	if err != nil || !strings.Contains(content, "傅里叶变换") {
		t.Fatalf("derived content wrong: %q err=%v", content, err)
	}
	if len(afterRevision.DerivedFiles) != 1 || afterRevision.DerivedFiles[0].Key != "content" {
		t.Fatalf("derived files wrong: %#v", afterRevision.DerivedFiles)
	}
	if gotAuth != "Bearer sk-vision" {
		t.Fatalf("auth header wrong: %s", gotAuth)
	}
	messages, _ := gotPayload["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("vision request must carry one user message: %#v", gotPayload)
	}
	userMessage := messages[0].(map[string]any)
	contentParts, _ := userMessage["content"].([]any)
	joined, _ := json.Marshal(contentParts)
	encodedImage := base64.StdEncoding.EncodeToString(png)
	if !strings.Contains(string(joined), encodedImage) || !strings.Contains(string(joined), "image_url") {
		t.Fatalf("image part missing from vision request: %s", joined)
	}
	if !strings.Contains(string(joined), "逐字提取") {
		t.Fatalf("extraction prompt missing: %s", joined)
	}
}

func TestProcessImageOCRFailurePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer server.Close()
	useOCRConfig(t, server.URL)

	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("ocrfail", "失败", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, _ := impl.New("ocrfail")
	png := []byte{0x89, 'P', 'N', 'G'}
	source, revision, err := store.Add("坏图", "bad.png", "image/png", int64(len(png)), strings.NewReader(string(png)), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessImageOCR("ocrfail", store, source.SourceID, revision.RevisionID); err == nil {
		t.Fatal("expected OCR failure")
	}
	after, _, err := store.Get(source.SourceID)
	if err != nil || after.Status != "stored" {
		t.Fatalf("caller owns failure transitions; source stays unprocessed: %#v err=%v", after, err)
	}
}
