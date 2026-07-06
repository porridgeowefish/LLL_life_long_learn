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
