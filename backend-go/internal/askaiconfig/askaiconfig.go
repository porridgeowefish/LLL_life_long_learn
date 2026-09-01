// Package askaiconfig loads/saves the Ask-AI provider config in config.local.json.
package askaiconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Provider is one configured Ask-AI model source.
type Provider struct {
	ID                  string `json:"id"`
	Kind                string `json:"kind"` // "openai" | "anthropic"
	Name                string `json:"name"`
	BaseURL             string `json:"baseURL"`
	APIKey              string `json:"apiKey"`
	Model               string `json:"model"`
	ContextWindowTokens int    `json:"contextWindowTokens,omitempty"`
	Reasoning           bool   `json:"reasoning,omitempty"` // openai-compatible: disclosed summary only
	Thinking            bool   `json:"thinking,omitempty"`  // provider capability; raw thinking is never exposed
}

// Config is the askAiProviders section.
type Config struct {
	Default      string             `json:"default"`
	SearchEngine string             `json:"searchEngine"` // "google" | "bing"
	Providers    []Provider         `json:"providers"`
	Bindings     map[string]Binding `json:"bindings,omitempty"`
}

type Binding struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
}

// Enabled reports whether at least one provider with key+baseURL+model exists.
func (c *Config) Enabled() bool {
	if c == nil {
		return false
	}
	for _, p := range c.Providers {
		if p.APIKey != "" && p.BaseURL != "" && p.Model != "" {
			return true
		}
	}
	return false
}

// Find returns the provider with id, else the default, else the first, else nil.
func (c *Config) Find(id string) *Provider {
	if c == nil {
		return nil
	}
	if id != "" {
		for i := range c.Providers {
			if c.Providers[i].ID == id {
				return &c.Providers[i]
			}
		}
	}
	for i := range c.Providers {
		if c.Providers[i].ID == c.Default {
			return &c.Providers[i]
		}
	}
	if len(c.Providers) > 0 {
		return &c.Providers[0]
	}
	return nil
}

// Resolve returns a copy of the provider selected for a logical AI service.
// Legacy configurations without bindings continue to use Find("").
func (c *Config) Resolve(service string) *Provider {
	if c == nil {
		return nil
	}
	binding, ok := c.Bindings[service]
	provider := c.Find(binding.ProviderID)
	if !ok {
		provider = c.Find("")
	}
	if provider == nil {
		return nil
	}
	resolved := *provider
	if ok && strings.TrimSpace(binding.Model) != "" {
		resolved.Model = strings.TrimSpace(binding.Model)
	}
	return &resolved
}

// pathFn resolves the config file path. It is a var so tests can redirect it.
var pathFn = func() string {
	base := paths.WORKSPACE
	if base == "" {
		base = paths.PROJECT_ROOT
	}
	return filepath.Join(base, "config.local.json")
}

// UseConfigPathForTest redirects the config file path and returns a restore
// function. For tests in OTHER packages (e.g. server handler tests) that
// cannot touch the unexported pathFn directly.
func UseConfigPathForTest(path string) (restore func()) {
	old := pathFn
	pathFn = func() string { return path }
	return func() { pathFn = old }
}

// Load reads the askAiProviders section. Returns (nil, nil) when the file or
// section is absent (feature disabled). Returns an error only on parse failure.
func Load() (*Config, error) {
	data, err := os.ReadFile(pathFn())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	section, ok := raw["askAiProviders"]
	if !ok {
		return nil, nil
	}
	var cfg Config
	if err := json.Unmarshal(section, &cfg); err != nil {
		return nil, err
	}
	cfg.Default = strings.TrimSpace(cfg.Default)
	cfg.SearchEngine = strings.TrimSpace(cfg.SearchEngine)
	for i := range cfg.Providers {
		cfg.Providers[i].ID = strings.TrimSpace(cfg.Providers[i].ID)
		cfg.Providers[i].BaseURL = strings.TrimSpace(cfg.Providers[i].BaseURL)
		cfg.Providers[i].APIKey = strings.TrimSpace(cfg.Providers[i].APIKey)
		cfg.Providers[i].Model = strings.TrimSpace(cfg.Providers[i].Model)
		if cfg.Providers[i].ContextWindowTokens < 0 {
			cfg.Providers[i].ContextWindowTokens = 0
		}
	}
	return &cfg, nil
}

// Save writes the askAiProviders section back to config.local.json, preserving
// all other top-level keys via a raw-JSON round-trip. Atomic write.
func Save(cfg Config) error {
	var raw map[string]any
	if data, err := os.ReadFile(pathFn()); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &raw)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	raw["askAiProviders"] = cfg
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(pathFn(), out, 0o644)
}
