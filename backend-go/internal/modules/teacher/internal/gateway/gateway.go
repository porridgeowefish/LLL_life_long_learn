package teachergateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	askaiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
)

// Message is one provider-facing conversation turn. ToolCalls is set on
// assistant messages that carry tool invocations; ToolCallID is set on
// role-"tool" messages answering a specific provider call. Both kinds map to
// the openai tool_calls/role=tool shape and the anthropic tool_use/tool_result
// blocks when present.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
}

type ToolCall struct {
	CallID         string
	ProviderCallID string
	ToolName       string
	Arguments      map[string]any
}

type Event struct {
	Type     string
	Delta    string
	ToolCall *ToolCall
	Usage    map[string]int
}

type Request struct {
	System     string
	Messages   []Message
	Tools      []Tool
	ProviderID string
}

type Gateway interface {
	Stream(context.Context, Request, func(Event)) error
}

type Configured struct{}

func (Configured) Stream(ctx context.Context, in Request, emit func(Event)) error {
	cfg, err := askaiconfig.Load()
	if err != nil {
		return err
	}
	p := cfg.Resolve("teacher")
	if strings.TrimSpace(in.ProviderID) != "" {
		p = cfg.Find(strings.TrimSpace(in.ProviderID))
	}
	if p == nil || p.APIKey == "" || p.BaseURL == "" || p.Model == "" {
		return errors.New("teacher provider is not configured")
	}
	if p.Kind == "anthropic" {
		return streamAnthropic(ctx, *p, in, emit)
	}
	return streamOpenAI(ctx, *p, in, emit)
}

func streamOpenAI(ctx context.Context, p askaiconfig.Provider, in Request, emit func(Event)) error {
	var buffers toolBufferSet
	messages := make([]map[string]any, 0, len(in.Messages)+1)
	if in.System != "" {
		messages = append(messages, map[string]any{"role": "system", "content": in.System})
	}
	for _, message := range in.Messages {
		messages = append(messages, openAIMessage(message))
	}
	body := map[string]any{"model": p.Model, "messages": messages, "stream": true, "stream_options": map[string]any{"include_usage": true}}
	if len(in.Tools) > 0 {
		var tools []map[string]any
		for _, tool := range in.Tools {
			tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.Schema}})
		}
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	return streamJSONLines(ctx, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", p, body, func(payload []byte) error {
		if string(payload) == "[DONE]" {
			return io.EOF
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ReasoningSummary string `json:"reasoning_summary"`
					Reasoning        string `json:"reasoning"`
					ReasoningText    string `json:"reasoning_text"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(payload, &chunk); err != nil {
			return nil
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.ReasoningSummary != "" {
				emit(Event{Type: "reasoning-summary-delta", Delta: choice.Delta.ReasoningSummary})
			} else if p.Reasoning {
				summary := firstNonEmpty(choice.Delta.ReasoningContent, choice.Delta.Reasoning, choice.Delta.ReasoningText)
				if summary != "" {
					emit(Event{Type: "reasoning-summary-delta", Delta: summary})
				}
			}
			if choice.Delta.Content != "" {
				emit(Event{Type: "text-delta", Delta: choice.Delta.Content})
			}
			for _, call := range choice.Delta.ToolCalls {
				buffers.add(call.Index, call.ID, call.Function.Name, call.Function.Arguments)
			}
		}
		if chunk.Usage.PromptTokens+chunk.Usage.CompletionTokens > 0 {
			emit(Event{Type: "usage", Usage: map[string]int{"inputTokens": chunk.Usage.PromptTokens, "outputTokens": chunk.Usage.CompletionTokens}})
		}
		return nil
	}, func() {
		for _, buffered := range buffers.drain() {
			var args map[string]any
			if json.Unmarshal([]byte(buffered.arguments), &args) == nil {
				emit(Event{Type: "tool-call-ready", ToolCall: &ToolCall{CallID: idgen.New("call"), ProviderCallID: buffered.id, ToolName: buffered.name, Arguments: args}})
			}
		}
		emit(Event{Type: "response-completed"})
	})
}

// The provider stream is processed synchronously, so a request-local buffer is
// kept in the callback closure through this tiny value holder.
type openAIBuffer struct {
	index               int
	id, name, arguments string
}
type toolBufferSet struct{ values map[int]openAIBuffer }

func (s *toolBufferSet) add(index int, id, name, args string) {
	if s.values == nil {
		s.values = map[int]openAIBuffer{}
	}
	v := s.values[index]
	v.index = index
	if id != "" {
		v.id = id
	}
	if name != "" {
		v.name = name
	}
	v.arguments += args
	s.values[index] = v
}
func (s *toolBufferSet) drain() []openAIBuffer {
	var out []openAIBuffer
	for i := 0; i < len(s.values); i++ {
		if v, ok := s.values[i]; ok {
			out = append(out, v)
		}
	}
	s.values = nil
	return out
}

// openAIMessage maps one gateway message to the openai chat-completions shape,
// including tool_calls and role=tool continuations for the search loop.
func openAIMessage(message Message) map[string]any {
	out := map[string]any{"role": message.Role, "content": message.Content}
	if message.ToolCallID != "" {
		out["role"] = "tool"
		out["tool_call_id"] = message.ToolCallID
	}
	if len(message.ToolCalls) > 0 {
		calls := make([]map[string]any, 0, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			args, _ := json.Marshal(call.Arguments)
			calls = append(calls, map[string]any{"id": call.ProviderCallID, "type": "function", "function": map[string]any{"name": call.ToolName, "arguments": string(args)}})
		}
		out["tool_calls"] = calls
	}
	return out
}

// anthropicMessage maps one gateway message to the anthropic messages shape,
// using tool_use/tool_result content blocks for the search loop.
func anthropicMessage(message Message) map[string]any {
	if message.ToolCallID != "" {
		return map[string]any{"role": "user", "content": []map[string]any{{
			"type":        "tool_result",
			"tool_use_id": message.ToolCallID,
			"content":     []map[string]any{{"type": "text", "text": message.Content}},
		}}}
	}
	if len(message.ToolCalls) > 0 {
		blocks := make([]map[string]any, 0, len(message.ToolCalls)+1)
		if message.Content != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": message.Content})
		}
		for _, call := range message.ToolCalls {
			blocks = append(blocks, map[string]any{"type": "tool_use", "id": call.ProviderCallID, "name": call.ToolName, "input": call.Arguments})
		}
		return map[string]any{"role": "assistant", "content": blocks}
	}
	return map[string]any{"role": message.Role, "content": message.Content}
}

func streamAnthropic(ctx context.Context, p askaiconfig.Provider, in Request, emit func(Event)) error {
	var messages []map[string]any
	for _, message := range in.Messages {
		messages = append(messages, anthropicMessage(message))
	}
	body := map[string]any{"model": p.Model, "system": in.System, "messages": messages, "stream": true, "max_tokens": 8192}
	// Re-synthesized assistant messages cannot replay signed thinking blocks,
	// so tool-capable turns disable thinking rather than risk protocol errors.
	if p.Thinking && len(in.Tools) == 0 {
		body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": 2048}
		body["max_tokens"] = 10240
	}
	if len(in.Tools) > 0 {
		var tools []map[string]any
		for _, tool := range in.Tools {
			tools = append(tools, map[string]any{"name": tool.Name, "description": tool.Description, "input_schema": tool.Schema})
		}
		body["tools"] = tools
	}
	var toolName, providerID, arguments string
	blockTypes := map[int]string{}
	// shape records the block/delta type fingerprint of this provider stream.
	// GLM's anthropic-compat endpoint intermittently mislabels thinking or
	// body deltas (the 2026-09-12 mixed reasoning incidents); this log pins
	// the raw shape on the next occurrence without capturing any content.
	shape := newStreamShape(p.Model)
	defer shape.flush()
	return streamJSONLines(ctx, strings.TrimRight(p.BaseURL, "/")+"/v1/messages", p, body, func(payload []byte) error {
		var event struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type     string `json:"type"`
				ID       string `json:"id"`
				Name     string `json:"name"`
				Thinking string `json:"thinking"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				Summary     string `json:"summary"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
			Message struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(payload, &event) != nil {
			return nil
		}
		switch event.Type {
		case "message_start":
			// message_start carries usage nested inside its message envelope;
			// input token counts only exist there.
			if event.Message.Usage.InputTokens > 0 {
				emit(Event{Type: "usage", Usage: map[string]int{"inputTokens": event.Message.Usage.InputTokens}})
			}
		case "content_block_start":
			blockTypes[event.Index] = event.ContentBlock.Type
			shape.blockStart(event.Index, event.ContentBlock.Type)
			if event.ContentBlock.Type == "tool_use" {
				toolName, providerID, arguments = event.ContentBlock.Name, event.ContentBlock.ID, ""
			} else if p.Reasoning && event.ContentBlock.Type == "thinking" && event.ContentBlock.Thinking != "" {
				emit(Event{Type: "reasoning-summary-delta", Delta: event.ContentBlock.Thinking})
			}
		case "content_block_delta":
			shape.delta(event.Index, event.Delta.Type)
			switch event.Delta.Type {
			case "text_delta":
				if blockTypes[event.Index] == "thinking" {
					if p.Reasoning && event.Delta.Text != "" {
						emit(Event{Type: "reasoning-summary-delta", Delta: event.Delta.Text})
					}
				} else {
					emit(Event{Type: "text-delta", Delta: event.Delta.Text})
				}
			case "summary_delta":
				if blockTypes[event.Index] == "text" {
					emit(Event{Type: "text-delta", Delta: event.Delta.Summary})
				} else {
					emit(Event{Type: "reasoning-summary-delta", Delta: event.Delta.Summary})
				}
			case "thinking_delta":
				if blockTypes[event.Index] == "text" {
					if event.Delta.Thinking != "" {
						emit(Event{Type: "text-delta", Delta: event.Delta.Thinking})
					}
				} else if p.Reasoning && event.Delta.Thinking != "" {
					emit(Event{Type: "reasoning-summary-delta", Delta: event.Delta.Thinking})
				}
			case "input_json_delta":
				arguments += event.Delta.PartialJSON
			}
		case "content_block_stop":
			delete(blockTypes, event.Index)
			if toolName != "" {
				var args map[string]any
				if json.Unmarshal([]byte(arguments), &args) == nil {
					emit(Event{Type: "tool-call-ready", ToolCall: &ToolCall{CallID: idgen.New("call"), ProviderCallID: providerID, ToolName: toolName, Arguments: args}})
				}
				toolName, providerID, arguments = "", "", ""
			}
		case "message_delta":
			if event.Usage.OutputTokens > 0 {
				emit(Event{Type: "usage", Usage: map[string]int{"outputTokens": event.Usage.OutputTokens}})
			}
		case "message_stop":
			emit(Event{Type: "response-completed"})
			return io.EOF
		}
		return nil
	}, nil)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func streamJSONLines(ctx context.Context, url string, p askaiconfig.Provider, body any, onData func([]byte) error, onEnd func()) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Kind == "anthropic" {
		req.Header.Set("x-api-key", p.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("provider status %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		if err := onData([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))); err != nil {
			if errors.Is(err, io.EOF) {
				if onEnd != nil {
					onEnd()
				}
				return nil
			}
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if onEnd != nil {
		onEnd()
	}
	return nil
}

// streamShape fingerprints one anthropic stream's block/delta labeling so
// provider-side mislabeling (thinking content routed to the 正文 channel on
// 2026-09-12) leaves hard evidence in tmp/teacher-stream-shapes.log. It
// records types and counts only — never message content.
type streamShape struct {
	model  string
	blocks []string
	deltas []string
	total  int
}

func newStreamShape(model string) *streamShape { return &streamShape{model: model} }

func (s *streamShape) blockStart(index int, blockType string) {
	s.blocks = append(s.blocks, fmt.Sprintf("%d:%s", index, blockType))
}

func (s *streamShape) delta(index int, deltaType string) {
	s.total++
	s.deltas = append(s.deltas, fmt.Sprintf("%d:%s", index, deltaType))
}

func (s *streamShape) fingerprint() string {
	counts := map[string]int{}
	for _, d := range s.deltas {
		counts[d]++
	}
	pairs := make([]string, 0, len(counts))
	for _, d := range s.deltas { // preserve first-seen order deterministically
		if counts[d] > 0 {
			pairs = append(pairs, fmt.Sprintf("%s×%d", d, counts[d]))
			counts[d] = 0
		}
	}
	return "blocks=[" + strings.Join(s.blocks, ",") + "] deltas=[" + strings.Join(pairs, ",") + "]"
}

func (s *streamShape) flush() {
	if s.total == 0 {
		return
	}
	path := filepath.Join(paths.WORKSPACE, "tmp", "teacher-stream-shapes.log")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s model=%s %s\n", time.Now().UTC().Format(time.RFC3339), s.model, s.fingerprint())
}
