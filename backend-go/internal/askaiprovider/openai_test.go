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
