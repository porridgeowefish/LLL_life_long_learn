// Package httpx provides HTTP helpers shared across route handlers.
package httpx

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// MaxBody limits JSON request bodies to 2MB to avoid runaway reads.
const MaxBody = 2 * 1024 * 1024

// WriteJSON serializes v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

// ReadJSON reads up to MaxBody bytes from r and decodes into v.
func ReadJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// Error sends a JSON error envelope.
func Error(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{"error": message})
}
