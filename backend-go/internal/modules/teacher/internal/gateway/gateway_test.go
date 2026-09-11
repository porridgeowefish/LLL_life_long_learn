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

func TestAnthropicStreamsUsageFromMessageEnvelope(t *testing.T) {
	// Per Anthropic SSE spec, message_start carries usage nested inside the
	// "message" object; only message_delta carries a top-level usage.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"message_start","message":{"usage":{"input_tokens":1384,"output_tokens":1}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"message_delta","delta":{},"usage":{"output_tokens":594}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"message_stop"}`)
	}))
	defer server.Close()
	provider := askaiconfig.Provider{Kind: "anthropic", BaseURL: server.URL, APIKey: "key", Model: "model"}
	usage := map[string]int{}
	err := streamAnthropic(context.Background(), provider, Request{}, func(event Event) {
		if event.Type == "usage" {
			for key, value := range event.Usage {
				usage[key] = value
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if usage["inputTokens"] != 1384 || usage["outputTokens"] != 594 {
		t.Fatalf("usage was not captured from message envelope: %#v", usage)
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

func TestAnthropicClassifiesDeltasByTheirContentBlock(t *testing.T) {
	tests := []struct {
		name        string
		blockType   string
		delta       string
		wantType    string
		wantContent string
	}{
		{
			name:        "thinking block wins over a mislabeled text delta",
			blockType:   "thinking",
			delta:       `{"type":"text_delta","text":"internal plan"}`,
			wantType:    "reasoning-summary-delta",
			wantContent: "internal plan",
		},
		{
			name:        "text block wins over a mislabeled thinking delta",
			blockType:   "text",
			delta:       `{"type":"thinking_delta","thinking":"learner-facing answer"}`,
			wantType:    "text-delta",
			wantContent: "learner-facing answer",
		},
		{
			name:        "text block wins over a mislabeled summary delta",
			blockType:   "text",
			delta:       `{"type":"summary_delta","summary":"final explanation"}`,
			wantType:    "text-delta",
			wantContent: "final explanation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprintf(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":%q}}\n\n", tt.blockType)
				fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":%s}\n\n", tt.delta)
				fmt.Fprintln(w, `data: {"type":"message_stop"}`)
			}))
			defer server.Close()

			provider := askaiconfig.Provider{Kind: "anthropic", BaseURL: server.URL, APIKey: "key", Model: "model", Thinking: true, Reasoning: true}
			var events []Event
			if err := streamAnthropic(context.Background(), provider, Request{}, func(event Event) {
				if event.Type == "text-delta" || event.Type == "reasoning-summary-delta" {
					events = append(events, event)
				}
			}); err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].Type != tt.wantType || events[0].Delta != tt.wantContent {
				t.Fatalf("delta crossed the reasoning/text boundary: %#v", events)
			}
		})
	}
}
