package teachergateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	askaiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
)

func TestOpenAIToolArgumentsBufferUntilComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"reasoning_content":"raw hidden thought","reasoning_summary":"checked scope","tool_calls":[{"index":0,"id":"provider-call","function":{"name":"delegate_learning_work","arguments":"{\"taskType\":\"verify\","}}]}}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"objective\":\"check\"}"}}]}}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "openai", BaseURL: server.URL, APIKey: "key", Model: "model"}
	var events []Event
	err := streamOpenAI(context.Background(), provider, Request{}, func(event Event) { events = append(events, event) })
	if err != nil {
		t.Fatal(err)
	}
	var call *ToolCall
	var summaries []string
	for _, event := range events {
		if event.Type == "tool-call-ready" {
			call = event.ToolCall
		}
		if event.Type == "reasoning-summary-delta" {
			summaries = append(summaries, event.Delta)
		}
	}
	if call == nil || call.ProviderCallID != "provider-call" || call.Arguments["objective"] != "check" {
		t.Fatalf("tool call was not normalized: %#v", call)
	}
	if len(summaries) != 1 || summaries[0] != "checked scope" {
		t.Fatalf("raw reasoning leaked or summary missing: %#v", summaries)
	}
}

func TestOpenAIDisclosedReasoningContentStreamsWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"reasoning_content":"检查任务边界"}}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "openai", BaseURL: server.URL, APIKey: "key", Model: "model", Reasoning: true}
	var got string
	if err := streamOpenAI(context.Background(), provider, Request{}, func(event Event) {
		if event.Type == "reasoning-summary-delta" {
			got += event.Delta
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got != "检查任务边界" {
		t.Fatalf("enabled provider reasoning was not streamed: %q", got)
	}
}

func TestOpenAICompatibleReasoningTextStreamsWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"reasoning_text":"检查上下文"}}]}`)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "openai", BaseURL: server.URL, APIKey: "key", Model: "model", Reasoning: true}
	var got string
	err := streamOpenAI(context.Background(), provider, Request{}, func(event Event) {
		if event.Type == "reasoning-summary-delta" {
			got += event.Delta
		}
	})
	if err != nil || got != "检查上下文" {
		t.Fatalf("compatible reasoning text was not streamed: err=%v got=%q", err, got)
	}
}

func TestAnthropicDoesNotExposeThinkingDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"raw hidden thought"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"summary_delta","summary":"safe summary"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"message_stop"}`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "anthropic", BaseURL: server.URL, APIKey: "key", Model: "model"}
	var deltas []string
	err := streamAnthropic(context.Background(), provider, Request{}, func(event Event) {
		if event.Type == "reasoning-summary-delta" {
			deltas = append(deltas, event.Delta)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 1 || deltas[0] != "safe summary" {
		t.Fatalf("unexpected reasoning disclosure: %#v", deltas)
	}
}

func TestAnthropicThinkingIsRequestedAndStreamedWhenEnabled(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"先检查授权范围"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"message_stop"}`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "anthropic", BaseURL: server.URL, APIKey: "key", Model: "model", Thinking: true, Reasoning: true}
	var got string
	if err := streamAnthropic(context.Background(), provider, Request{}, func(event Event) {
		if event.Type == "reasoning-summary-delta" {
			got += event.Delta
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got != "先检查授权范围" || !strings.Contains(body, `"thinking":{"budget_tokens":2048,"type":"enabled"}`) {
		t.Fatalf("thinking request/stream mismatch: body=%s got=%q", body, got)
	}
}
