package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platformconfig "github.com/xmz14/lll/backend-go/internal/platform/config"
)

func useSearchConfig(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.local.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := platformconfig.UsePathForTest(path)
	t.Cleanup(restore)
}

func TestConfiguredReturnsNilWithoutSection(t *testing.T) {
	useSearchConfig(t, `{}`)
	if Configured() != nil {
		t.Fatal("search must stay disabled without a webSearch section")
	}
}

func TestConfiguredResolvesAPIKeyEnv(t *testing.T) {
	useSearchConfig(t, `{"webSearch":{"provider":"zhipu","apiKeyEnv":"LLL_TEST_SEARCH_KEY","engine":"search_std"}}`)
	t.Setenv("LLL_TEST_SEARCH_KEY", "sk-test")
	if Configured() == nil {
		t.Fatal("search should enable via apiKeyEnv")
	}
}

func TestZhipuSearchPacksResults(t *testing.T) {
	var gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"search_result":[{"title":"Go 1.24 发布","link":"https://example.com/go124","content":"泛型别名落地。","media":"example.com"}]}`))
	}))
	defer server.Close()
	restore := setZhipuBaseURL(server.URL)
	defer restore()

	searcher := zhipu{apiKey: "sk-x", engine: "search_std"}
	results, err := searcher.Search(context.Background(), "go 1.24 发布")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer sk-x" || !strings.Contains(gotBody, `"search_query":"go 1.24 发布"`) || !strings.Contains(gotBody, `"search_engine":"search_std"`) {
		t.Fatalf("request wrong: auth=%q body=%s", gotAuth, gotBody)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/go124" {
		t.Fatalf("results wrong: %#v", results)
	}
	formatted := FormatResults(results)
	if !strings.Contains(formatted, "[1] Go 1.24 发布") || !strings.Contains(formatted, "https://example.com/go124") || !strings.Contains(formatted, "example.com") {
		t.Fatalf("format wrong: %s", formatted)
	}
}

func TestZhipuSearchMapsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"1002","message":"invalid key"}}`))
	}))
	defer server.Close()
	restore := setZhipuBaseURL(server.URL)
	defer restore()

	searcher := zhipu{apiKey: "bad", engine: "search_std"}
	if _, err := searcher.Search(context.Background(), "q"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected status error, got %v", err)
	}
}
