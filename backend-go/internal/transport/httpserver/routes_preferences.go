package server

import (
	"io"
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/preferencestore"
)

func (s *Server) handleGetPreferences(w http.ResponseWriter, _ *http.Request) {
	snapshot, err := preferencestore.Ensure()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"path":     preferencestore.Filename,
		"content":  snapshot.Content,
		"maxBytes": preferencestore.MaxBytes,
	})
}

func (s *Server) handlePutPreferences(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, preferencestore.MaxBytes+1))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body) > preferencestore.MaxBytes {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "preferences file exceeds 256 KiB")
		return
	}
	snapshot, err := preferencestore.Write(string(body))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "path": preferencestore.Filename, "bytes": len([]byte(snapshot.Content))})
}
