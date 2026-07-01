package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandleShutdownRequiresLoopback(t *testing.T) {
	s := &Server{}
	s.SetShutdownFunc(func() {})

	req := httptest.NewRequest(http.MethodPost, "/api/system/shutdown", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	rec := httptest.NewRecorder()

	s.handleShutdown(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleShutdownCallsConfiguredCallback(t *testing.T) {
	s := &Server{}
	var called atomic.Bool
	s.SetShutdownFunc(func() { called.Store(true) })

	req := httptest.NewRequest(http.MethodPost, "/api/system/shutdown", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	s.handleShutdown(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}

	deadline := time.Now().Add(time.Second)
	for !called.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !called.Load() {
		t.Fatal("shutdown callback was not called")
	}
}
