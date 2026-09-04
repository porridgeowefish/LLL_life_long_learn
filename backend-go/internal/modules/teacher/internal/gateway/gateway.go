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
	"strings"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	askaiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
)

type Message struct {
	Role    string
	Content string
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
		messages = append(messages, map[string]any{"role": message.Role, "content": message.Content})
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

func streamAnthropic(ctx context.Context, p askaiconfig.Provider, in Request, emit func(Event)) error {
	var messages []map[string]any
	for _, message := range in.Messages {
		messages = append(messages, map[string]any{"role": message.Role, "content": message.Content})
	}
	body := map[string]any{"model": p.Model, "system": in.System, "messages": messages, "stream": true, "max_tokens": 8192}
	if p.Thinking {
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
	return streamJSONLines(ctx, strings.TrimRight(p.BaseURL, "/")+"/v1/messages", p, body, func(payload []byte) error {
		var event struct {
			Type         string `json:"type"`
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
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(payload, &event) != nil {
			return nil
		}
		switch event.Type {
		case "content_block_start":
			if event.ContentBlock.Type == "tool_use" {
				toolName, providerID, arguments = event.ContentBlock.Name, event.ContentBlock.ID, ""
			} else if p.Reasoning && event.ContentBlock.Type == "thinking" && event.ContentBlock.Thinking != "" {
				emit(Event{Type: "reasoning-summary-delta", Delta: event.ContentBlock.Thinking})
			}
		case "content_block_delta":
			switch event.Delta.Type {
			case "text_delta":
				emit(Event{Type: "text-delta", Delta: event.Delta.Text})
			case "summary_delta":
				emit(Event{Type: "reasoning-summary-delta", Delta: event.Delta.Summary})
			case "thinking_delta":
				if p.Reasoning && event.Delta.Thinking != "" {
					emit(Event{Type: "reasoning-summary-delta", Delta: event.Delta.Thinking})
				}
			case "input_json_delta":
				arguments += event.Delta.PartialJSON
			}
		case "content_block_stop":
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
