package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/runprogress"
)

func TestRunStatusRejectsBadToken(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	srv.runProgress.Register("r1")
	body := `{"activity":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader(body))
	req.SetPathValue("runId", "r1")
	req.Header.Set("X-Run-Token", "wrong")
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: want 401, got %d", rec.Code)
	}
}

func TestRunStatusRejectsMissingToken(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	srv.runProgress.Register("r1")
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader(`{}`))
	req.SetPathValue("runId", "r1")
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing token: want 401, got %d", rec.Code)
	}
}

func TestRunStatusAcceptsAndDoneCompletes(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	tok := srv.runProgress.Register("r1")
	body := `{"activity":"wrote p","pagesDone":3,"pagesPlanned":5,"done":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader(body))
	req.SetPathValue("runId", "r1")
	req.Header.Set("X-Run-Token", tok)
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Status runprogress.Status `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status.PagesDone != 3 || resp.Status.PagesPlanned != 5 || !resp.Status.Done {
		t.Errorf("status not reflected: %+v", resp.Status)
	}
	// The done branch calls sessions.SetFinished(runId, completed, 0) against
	// the package-global store. For an unknown runId it is a no-op (returns
	// false) and must NOT fail the request. We assert the merged status is
	// returned + the store remembers Done.
	got, ok := srv.runProgress.Get("r1")
	if !ok || !got.Done || got.PagesDone != 3 {
		t.Errorf("store not updated: %+v ok=%v", got, ok)
	}
}

func TestRunStatusRejectsInvalidBody(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	tok := srv.runProgress.Register("r1")
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader("not json"))
	req.SetPathValue("runId", "r1")
	req.Header.Set("X-Run-Token", tok)
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid body: want 400, got %d", rec.Code)
	}
}
