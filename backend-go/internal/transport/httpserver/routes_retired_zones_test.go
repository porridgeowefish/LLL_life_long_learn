package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func TestRetiredZoneFilesAreNotReadableOrWritable(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	root, err := workspace.ProjectRootForSlug("testproj")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root+"/summary", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root+"/summary/flashcards.json", []byte(`{"cards":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	read := httptest.NewRequest(http.MethodGet, "/files/projects/testproj/summary/flashcards.json", nil)
	read.SetPathValue("id", "testproj")
	readRec := httptest.NewRecorder()
	srv.handleReadFile(readRec, read)
	if readRec.Code != http.StatusForbidden {
		t.Fatalf("read retired summary: got %d, want %d", readRec.Code, http.StatusForbidden)
	}

	for _, path := range []string{"summary/flashcards.json", "extend/flower.json"} {
		req := httptest.NewRequest(http.MethodPost, "/files/projects/testproj/"+path, strings.NewReader("retired"))
		req.SetPathValue("id", "testproj")
		rec := httptest.NewRecorder()
		srv.handleWriteFile(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("write retired %s: got %d, want %d", path, rec.Code, http.StatusForbidden)
		}
	}
}
