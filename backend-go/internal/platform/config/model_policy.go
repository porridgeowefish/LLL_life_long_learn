package config

import (
	"net/url"
	"strings"
)

// ModelOutputTokens is the user-selected ceiling for every model request.
// It is a ceiling, not a requested response length. Never lower it silently.
const ModelOutputTokens = 100000

func ModelOutputLimitKey(baseURL, model string) string {
	endpoint, _ := url.Parse(baseURL)
	if endpoint != nil && (endpoint.Hostname() == "api.openai.com" || endpoint.Hostname() == "api.groq.com") {
		return "max_completion_tokens"
	}
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "chat-latest" || strings.HasPrefix(model, "gpt-5") || strings.HasPrefix(model, "gpt-6") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4") {
		return "max_completion_tokens"
	}
	return "max_tokens"
}
