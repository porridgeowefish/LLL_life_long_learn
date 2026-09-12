package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	preferencestore "github.com/xmz14/lll/backend-go/internal/modules/preferences"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

func TestPreferencesEndpointOwnsOneWorkspaceGlobalFile(t *testing.T) {
	old := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = old })
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := &Server{}

	put := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader("# 偏好\n\n先给结论。\n"))
	putRecorder := httptest.NewRecorder()
	srv.handlePutPreferences(putRecorder, put)
	if putRecorder.Code != http.StatusOK {
		t.Fatalf("put: %d %s", putRecorder.Code, putRecorder.Body.String())
	}

	getRecorder := httptest.NewRecorder()
	srv.handleGetPreferences(getRecorder, httptest.NewRequest(http.MethodGet, "/api/preferences", nil))
	var response struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Path != preferencestore.Filename || response.Content != "# 偏好\n\n先给结论。\n" {
		t.Fatalf("unexpected response: %#v", response)
	}
	data, err := os.ReadFile(filepath.Join(paths.WORKSPACE, preferencestore.Filename))
	if err != nil || string(data) != response.Content {
		t.Fatalf("unexpected file: %q, %v", data, err)
	}
}
