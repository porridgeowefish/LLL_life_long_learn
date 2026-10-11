package teacherservice

import (
	"encoding/json"
	"errors"
	aiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
	gateway "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/gateway"
	"unicode/utf8"
)

// Estimates are deliberately conservative; a configured window is a capacity
// constraint, not a claim that the local estimator is a provider tokenizer.
func estimateTextTokens(s string) int {
	ascii, other := 0, 0
	for _, r := range s {
		if r < 128 {
			ascii++
		} else {
			other++
		}
	}
	return (ascii+2)/3 + (other*3+1)/2
}

type requestBudget struct{ window, output, thinking, margin int }

var errContextBudgetExhausted = errors.New("teacher context budget exhausted; current learner message retained")

func budgetForProvider(id string) requestBudget {
	b := requestBudget{window: 32768, output: 8192, margin: 1024}
	if cfg, err := aiconfig.Load(); err == nil && cfg != nil {
		p := cfg.Resolve("teacher")
		if id != "" {
			p = cfg.Find(id)
		}
		if p != nil {
			if p.ContextWindowTokens > 0 {
				b.window = p.ContextWindowTokens
			}
			if p.Kind == "anthropic" && p.Thinking {
				b.thinking = 2048
			}
		}
	}
	if b.output > b.window/4 {
		b.output = b.window / 4
	}
	if b.output < 1 {
		b.output = 1
	}
	if b.margin > b.window/16 {
		b.margin = b.window / 16
	}
	return b
}
func (b requestBudget) input(req gateway.Request) int {
	n := estimateTextTokens(req.System) + 64
	for _, m := range req.Messages {
		raw, _ := json.Marshal(m)
		n += estimateTextTokens(string(raw)) + 24
	}
	for _, t := range req.Tools {
		raw, _ := json.Marshal(t)
		n += estimateTextTokens(string(raw)) + 32
	}
	return n
}
func (b requestBudget) available(req gateway.Request) int {
	return b.window - b.output - b.thinking - b.margin - b.input(req)
}
func (b requestBudget) check(req gateway.Request) error {
	if b.available(req) < 0 {
		return errContextBudgetExhausted
	}
	return nil
}

// CheckGenerationRequest uses the same complete serialized-input estimator as
// teaching turns, without dropping any mandatory generation instructions.
func CheckGenerationRequest(req gateway.Request, p aiconfig.Provider) (int, error) {
	b := requestBudget{window: 32768, output: 8192, margin: 1024}
	if p.ContextWindowTokens > 0 {
		b.window = p.ContextWindowTokens
	}
	if p.Kind == "anthropic" && p.Thinking {
		b.thinking = 2048
	}
	b.output = max(1, min(b.output, b.window/4))
	b.margin = min(b.margin, b.window/16)
	if req.MaxOutputTokens > 0 {
		b.output = req.MaxOutputTokens
	}
	return b.available(req), b.check(req)
}
func (b requestBudget) trimInitial(req *gateway.Request) error {
	// Only complete historical messages are removed, before any tool exchange.
	// The latest learner turn is mandatory and is never sliced or discarded.
	for b.available(*req) < 1024 && len(req.Messages) > 1 {
		req.Messages = req.Messages[1:]
	}
	return b.check(*req)
}
func truncateTokens(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if estimateTextTokens(s) <= limit {
		return s
	}
	end := 0
	n := 0
	for offset, r := range s {
		cost := 1
		if r >= 128 {
			cost = 2
		}
		if n+cost > limit {
			break
		}
		n += cost
		end = offset + utf8.RuneLen(r)
	}
	return s[:end]
}
