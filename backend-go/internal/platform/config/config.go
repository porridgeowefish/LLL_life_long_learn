// Package config owns the single typed application configuration loader.
//
// Precedence: compiled defaults < config.local.json < LLL_* environment
// < explicit CLI flags. Loading normalizes the historical flat keys in
// memory and never rewrites the user's file. Diagnostics redact secrets.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Section shapes. These mirror DATA_DESIGN.md canonical names; the JSON
// tags are the canonical sectioned keys.

type Server struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Workspace struct {
	Root string `json:"root"`
}

type Provider struct {
	ID                  string `json:"id"`
	Kind                string `json:"kind"` // "openai" | "anthropic"
	Name                string `json:"name"`
	BaseURL             string `json:"baseURL"`
	APIKey              string `json:"apiKey,omitempty"` // compatibility inline key; apiKeyEnv wins
	APIKeyEnv           string `json:"apiKeyEnv,omitempty"`
	Model               string `json:"model"`
	ContextWindowTokens int    `json:"contextWindowTokens,omitempty"`
	Reasoning           bool   `json:"reasoning,omitempty"`
	Thinking            bool   `json:"thinking,omitempty"`
}

type Binding struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
}

type AI struct {
	DefaultProvider string             `json:"defaultProvider"`
	SearchEngine    string             `json:"searchEngine"` // "google" | "bing"
	Providers       []Provider         `json:"providers"`
	Bindings        map[string]Binding `json:"bindings,omitempty"`
}

type Assistant struct {
	Runtime                 string            `json:"runtime"`
	MaxConcurrent           int               `json:"maxConcurrent"`
	MaxConcurrentPerProject int               `json:"maxConcurrentPerProject"`
	Bins                    map[string]string `json:"bins"`
}

type ImageBackup struct {
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey,omitempty"`
	APIKeyEnv string `json:"apiKeyEnv,omitempty"`
	Model     string `json:"model"`
}

type Image struct {
	Enabled     bool        `json:"enabled"`
	BaseURL     string      `json:"baseURL"`
	APIKey      string      `json:"apiKey,omitempty"`
	APIKeyEnv   string      `json:"apiKeyEnv,omitempty"`
	Model       string      `json:"model"`
	PromptModel string      `json:"promptModel"`
	PythonBin   string      `json:"pythonBin"`
	Backup      ImageBackup `json:"backup"`
}

type UI struct {
	Theme string `json:"theme"`
}

// Config is the resolved typed application configuration.
type Config struct {
	Server    Server    `json:"server"`
	Workspace Workspace `json:"workspace"`
	AI        AI        `json:"ai"`
	Assistant Assistant `json:"assistant"`
	Image     Image     `json:"image"`
	UI        UI        `json:"ui"`

	// Path is the config file that was loaded ("" when none existed).
	Path string `json:"-"`
	// Warnings carry non-secret diagnostics (e.g. old+new key conflicts).
	Warnings []string `json:"-"`
}

// source records where each resolved value came from, for diagnostics only.
type source struct {
	Section string `json:"section"`
	From    string `json:"from"` // "default" | "file" | "env" | "flag"
}

// Provenance maps section name → source. Never written back to disk.
type Provenance map[string]source

// Defaults returns the compiled default configuration.
func Defaults() Config {
	return Config{
		Server:    Server{Host: "127.0.0.1", Port: 8787},
		Workspace: Workspace{Root: ""},
		AI:        AI{SearchEngine: "google"},
		Assistant: Assistant{Runtime: "claude", MaxConcurrent: 5, MaxConcurrentPerProject: 2, Bins: map[string]string{}},
		Image:     Image{PythonBin: "python", APIKeyEnv: "LLL_IMAGE_API_KEY"},
		UI:        UI{Theme: "lychee-paper"},
	}
}

// Validate reports actionable errors for invalid types, enums, and unsafe
// workspace paths. Secrets are never included in the message.
func (c *Config) Validate() error {
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port %d is out of range", c.Server.Port)
	}
	if c.Workspace.Root != "" {
		if !filepath.IsAbs(c.Workspace.Root) {
			return fmt.Errorf("workspace.root %q must be an absolute path", c.Workspace.Root)
		}
		if strings.Contains(c.Workspace.Root, "..") {
			return fmt.Errorf("workspace.root %q must not contain '..'", filepath.ToSlash(c.Workspace.Root))
		}
	}
	if c.AI.SearchEngine != "" && c.AI.SearchEngine != "google" && c.AI.SearchEngine != "bing" {
		return fmt.Errorf("ai.searchEngine %q is not one of google|bing", c.AI.SearchEngine)
	}
	seen := map[string]bool{}
	for i, p := range c.AI.Providers {
		if p.Kind != "openai" && p.Kind != "anthropic" {
			return fmt.Errorf("ai.providers[%d].kind %q is not one of openai|anthropic", i, p.Kind)
		}
		if p.ID == "" {
			return fmt.Errorf("ai.providers[%d].id is required", i)
		}
		if seen[p.ID] {
			return fmt.Errorf("ai.providers[%d].id %q is duplicated", i, p.ID)
		}
		seen[p.ID] = true
	}
	switch c.Assistant.Runtime {
	case "claude", "codebuddy", "hermes", "codex", "trae", "":
	default:
		return fmt.Errorf("assistant.runtime %q is not a supported runtime", c.Assistant.Runtime)
	}
	if c.Assistant.MaxConcurrent < 1 {
		return fmt.Errorf("assistant.maxConcurrent %d must be >= 1", c.Assistant.MaxConcurrent)
	}
	if c.Assistant.MaxConcurrentPerProject < 1 {
		return fmt.Errorf("assistant.maxConcurrentPerProject %d must be >= 1", c.Assistant.MaxConcurrentPerProject)
	}
	switch c.UI.Theme {
	case "lychee-paper", "mountain-mist", "wisteria-gray", "night-ink":
	default:
		return fmt.Errorf("ui.theme %q is not a supported theme", c.UI.Theme)
	}
	return nil
}

// ResolveAPIKey returns the provider API key with environment precedence:
// the variable named by apiKeyEnv wins over an inline apiKey.
func (p Provider) ResolveAPIKey() string {
	if p.APIKeyEnv != "" {
		if v := strings.TrimSpace(os.Getenv(p.APIKeyEnv)); v != "" {
			return v
		}
	}
	return p.APIKey
}

// Redacted returns a copy safe for logs: every secret-bearing field is
// replaced with a mask; nothing else changes.
func (c Config) Redacted() Config {
	out := c
	mask := "••••"
	out.AI.Providers = make([]Provider, len(c.AI.Providers))
	for i, p := range c.AI.Providers {
		if p.APIKey != "" {
			p.APIKey = mask
		}
		out.AI.Providers[i] = p
	}
	if c.Image.APIKey != "" {
		out.Image.APIKey = mask
	}
	if c.Image.Backup.APIKey != "" {
		out.Image.Backup.APIKey = mask
	}
	return out
}

// Options controls Load.
type Options struct {
	// Path overrides the config file location (CLI --config).
	Path string
	// WorkspaceRoot overrides the workspace root (CLI --workspace / env).
	WorkspaceRoot string
	// Port overrides the server port (CLI --port / env).
	Port int
}

// filePath resolves the config file location.
func filePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return filepath.Join(workspaceBase(), "config.local.json")
}

func workspaceBase() string {
	if v := strings.TrimSpace(os.Getenv("LLL_WORKSPACE_ROOT")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("WORKSPACE")); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
		dir = parent
	}
}

// diskShape is the raw sectioned document plus tolerated legacy flat keys.
type diskShape struct {
	Server    *json.RawMessage `json:"server"`
	Workspace *json.RawMessage `json:"workspace"`
	AI        *json.RawMessage `json:"ai"`
	Assistant *json.RawMessage `json:"assistant"`
	Image     *json.RawMessage `json:"image"`
	UI        *json.RawMessage `json:"ui"`

	// Legacy flat keys (read-compatible only).
	AskAiProviders     *json.RawMessage  `json:"askAiProviders"`
	AgentRuntime       *string           `json:"agentRuntime"`
	AgentRuntimeBins   map[string]string `json:"agentRuntimeBins"`
	ImageAPIKey        *string           `json:"imageApiKey"`
	ImageBaseURL       *string           `json:"imageBaseURL"`
	ImageModel         *string           `json:"imageModel"`
	ImageBackupAPIKey  *string           `json:"imageBackupApiKey"`
	ImageBackupBaseURL *string           `json:"imageBackupBaseURL"`
	ImageBackupModel   *string           `json:"imageBackupModel"`
	ImagePromptModel   *string           `json:"imagePromptModel"`
	PythonBin          *string           `json:"pythonBin"`
}

// legacyAI mirrors askaiconfig.Config.
type legacyAI struct {
	Default      string             `json:"default"`
	SearchEngine string             `json:"searchEngine"`
	Providers    []legacyProvider   `json:"providers"`
	Bindings     map[string]Binding `json:"bindings"`
}

type legacyProvider struct {
	ID                  string `json:"id"`
	Kind                string `json:"kind"`
	Name                string `json:"name"`
	BaseURL             string `json:"baseURL"`
	APIKey              string `json:"apiKey"`
	Model               string `json:"model"`
	ContextWindowTokens int    `json:"contextWindowTokens,omitempty"`
	Reasoning           bool   `json:"reasoning,omitempty"`
	Thinking            bool   `json:"thinking,omitempty"`
}

// Load resolves the typed configuration. It reads the file (when present),
// applies compatibility mapping, then environment, then explicit options.
// It never writes.
func Load(opts Options) (Config, Provenance, error) {
	cfg := Defaults()
	prov := Provenance{}
	for _, section := range []string{"server", "workspace", "ai", "assistant", "image", "ui"} {
		prov[section] = source{Section: section, From: "default"}
	}

	path := filePath(opts.Path)
	cfg.Path = path
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, prov, fmt.Errorf("read %s: %w", path, err)
	}
	if err == nil {
		var disk diskShape
		if err := json.Unmarshal(data, &disk); err != nil {
			return cfg, prov, fmt.Errorf("parse %s: %w", path, err)
		}
		applyDisk(&cfg, &disk, prov)
	}

	applyEnv(&cfg, prov)
	applyFlags(&cfg, opts, prov)

	if err := cfg.Validate(); err != nil {
		return cfg, prov, err
	}
	sort.Strings(cfg.Warnings)
	return cfg, prov, nil
}

func warnf(cfg *Config, format string, args ...any) {
	cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(format, args...))
}

func applyDisk(cfg *Config, disk *diskShape, prov Provenance) {
	mark := func(section string) { prov[section] = source{Section: section, From: "file"} }

	if disk.Server != nil {
		var s Server
		if err := json.Unmarshal(*disk.Server, &s); err == nil {
			if s.Host != "" {
				cfg.Server.Host = s.Host
			}
			if s.Port != 0 {
				cfg.Server.Port = s.Port
			}
			mark("server")
		} else {
			warnf(cfg, "config: server section ignored: %v", err)
		}
	}
	if disk.Workspace != nil {
		var w Workspace
		if err := json.Unmarshal(*disk.Workspace, &w); err == nil && strings.TrimSpace(w.Root) != "" {
			cfg.Workspace.Root = strings.TrimSpace(w.Root)
			mark("workspace")
		}
	}
	if disk.AI != nil {
		var ai AI
		if err := json.Unmarshal(*disk.AI, &ai); err == nil {
			mergeAI(cfg, ai)
			mark("ai")
		} else {
			warnf(cfg, "config: ai section ignored: %v", err)
		}
	}
	if disk.Assistant != nil {
		var a Assistant
		if err := json.Unmarshal(*disk.Assistant, &a); err == nil {
			if a.Runtime != "" {
				cfg.Assistant.Runtime = a.Runtime
			}
			if a.MaxConcurrent > 0 {
				cfg.Assistant.MaxConcurrent = a.MaxConcurrent
			}
			if a.MaxConcurrentPerProject > 0 {
				cfg.Assistant.MaxConcurrentPerProject = a.MaxConcurrentPerProject
			}
			if a.Bins != nil {
				cfg.Assistant.Bins = a.Bins
			}
			mark("assistant")
		} else {
			warnf(cfg, "config: assistant section ignored: %v", err)
		}
	}
	if disk.Image != nil {
		var img Image
		if err := json.Unmarshal(*disk.Image, &img); err == nil {
			mergeImage(cfg, img)
			mark("image")
		} else {
			warnf(cfg, "config: image section ignored: %v", err)
		}
	}
	if disk.UI != nil {
		var ui UI
		if err := json.Unmarshal(*disk.UI, &ui); err == nil && ui.Theme != "" {
			cfg.UI.Theme = ui.Theme
			mark("ui")
		}
	}

	// Legacy flat keys fill only unset canonical fields.
	if disk.AskAiProviders != nil && disk.AI == nil {
		var legacy legacyAI
		if err := json.Unmarshal(*disk.AskAiProviders, &legacy); err == nil {
			ai := AI{DefaultProvider: legacy.Default, SearchEngine: legacy.SearchEngine, Bindings: legacy.Bindings}
			for _, p := range legacy.Providers {
				ai.Providers = append(ai.Providers, Provider{ID: p.ID, Kind: p.Kind, Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Model: p.Model, ContextWindowTokens: p.ContextWindowTokens, Reasoning: p.Reasoning, Thinking: p.Thinking})
			}
			mergeAI(cfg, ai)
			mark("ai")
		}
	} else if disk.AskAiProviders != nil && disk.AI != nil {
		warnf(cfg, "config: both 'ai' and legacy 'askAiProviders' present; sectioned 'ai' wins")
	}
	if disk.AgentRuntime != nil && *disk.AgentRuntime != "" && disk.Assistant == nil {
		cfg.Assistant.Runtime = *disk.AgentRuntime
		mark("assistant")
	}
	if disk.AgentRuntimeBins != nil && disk.Assistant == nil {
		cfg.Assistant.Bins = disk.AgentRuntimeBins
	}
	applyLegacyImage(cfg, disk, mark)
	if disk.UI == nil {
		// ui.theme is the one nested key the flat era shared with sectioned form.
	}
}

// flatTheme reads {"ui":{"theme":...}} which both eras share.
func applyLegacyImage(cfg *Config, disk *diskShape, mark func(string)) {
	if disk.Image != nil {
		return
	}
	touched := false
	set := func(dst *string, v *string) {
		if v != nil && *v != "" {
			*dst = strings.TrimSpace(*v)
			touched = true
		}
	}
	set(&cfg.Image.APIKey, disk.ImageAPIKey)
	set(&cfg.Image.BaseURL, disk.ImageBaseURL)
	set(&cfg.Image.Model, disk.ImageModel)
	set(&cfg.Image.Backup.APIKey, disk.ImageBackupAPIKey)
	set(&cfg.Image.Backup.BaseURL, disk.ImageBackupBaseURL)
	set(&cfg.Image.Backup.Model, disk.ImageBackupModel)
	set(&cfg.Image.PromptModel, disk.ImagePromptModel)
	set(&cfg.Image.PythonBin, disk.PythonBin)
	if cfg.Image.APIKeyEnv == "" && disk.ImageAPIKey != nil {
		cfg.Image.APIKeyEnv = "LLL_IMAGE_API_KEY"
	}
	if cfg.Image.Backup.APIKeyEnv == "" && disk.ImageBackupAPIKey != nil {
		cfg.Image.Backup.APIKeyEnv = "LLL_IMAGE_BACKUP_API_KEY"
	}
	if touched {
		mark("image")
	}
}

func mergeAI(cfg *Config, ai AI) {
	if ai.DefaultProvider != "" {
		cfg.AI.DefaultProvider = ai.DefaultProvider
	}
	if ai.SearchEngine != "" {
		cfg.AI.SearchEngine = ai.SearchEngine
	}
	if ai.Providers != nil {
		cfg.AI.Providers = ai.Providers
	}
	if ai.Bindings != nil {
		cfg.AI.Bindings = ai.Bindings
	}
}

func mergeImage(cfg *Config, img Image) {
	if img.BaseURL != "" || img.APIKey != "" || img.APIKeyEnv != "" || img.Model != "" || img.PromptModel != "" || img.PythonBin != "" || img.Backup != (ImageBackup{}) || img.Enabled {
		cfg.Image.Enabled = img.Enabled || img.BaseURL != "" || img.APIKey != "" || img.APIKeyEnv != ""
		if img.BaseURL != "" {
			cfg.Image.BaseURL = img.BaseURL
		}
		if img.APIKey != "" {
			cfg.Image.APIKey = img.APIKey
		}
		if img.APIKeyEnv != "" {
			cfg.Image.APIKeyEnv = img.APIKeyEnv
		}
		if img.Model != "" {
			cfg.Image.Model = img.Model
		}
		if img.PromptModel != "" {
			cfg.Image.PromptModel = img.PromptModel
		}
		if img.PythonBin != "" {
			cfg.Image.PythonBin = img.PythonBin
		}
		if img.Backup.BaseURL != "" {
			cfg.Image.Backup.BaseURL = img.Backup.BaseURL
		}
		if img.Backup.APIKey != "" {
			cfg.Image.Backup.APIKey = img.Backup.APIKey
		}
		if img.Backup.APIKeyEnv != "" {
			cfg.Image.Backup.APIKeyEnv = img.Backup.APIKeyEnv
		}
		if img.Backup.Model != "" {
			cfg.Image.Backup.Model = img.Backup.Model
		}
	}
}

// applyEnv applies LLL_* environment overrides.
func applyEnv(cfg *Config, prov Provenance) {
	mark := func(section, key string) { prov[section] = source{Section: section, From: "env"} }

	if v := strings.TrimSpace(os.Getenv("LLL_SERVER_PORT")); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
			mark("server", "LLL_SERVER_PORT")
		} else {
			warnf(cfg, "config: LLL_SERVER_PORT=%q is not a number; ignored", v)
		}
	}
	if v := strings.TrimSpace(os.Getenv("LLL_WORKSPACE_ROOT")); v != "" {
		cfg.Workspace.Root = v
		mark("workspace", "LLL_WORKSPACE_ROOT")
	} else if v := strings.TrimSpace(os.Getenv("WORKSPACE")); v != "" {
		cfg.Workspace.Root = v
		mark("workspace", "WORKSPACE")
	}
	if v := strings.TrimSpace(os.Getenv("LLL_AGENT_RUNTIME")); v != "" {
		cfg.Assistant.Runtime = v
		mark("assistant", "LLL_AGENT_RUNTIME")
	}
	// Runtime binary environment variables (CLAUDE_BIN etc.) are consumed by
	// the assistant runtime resolver; the config layer records nothing here.
}

// applyFlags applies explicit CLI overrides last.
func applyFlags(cfg *Config, opts Options, prov Provenance) {
	if opts.WorkspaceRoot != "" {
		cfg.Workspace.Root = opts.WorkspaceRoot
		prov["workspace"] = source{Section: "workspace", From: "flag"}
	}
	if opts.Port != 0 {
		cfg.Server.Port = opts.Port
		prov["server"] = source{Section: "server", From: "flag"}
	}
}

// ProjectsRoot derives the projects directory from the resolved workspace.
func (c *Config) ProjectsRoot() string {
	root := c.WorkspaceRootResolved()
	return filepath.Join(root, "projects")
}

// WorkspaceRootResolved returns the effective workspace root: the resolved
// value or the discovered repository root.
func (c *Config) WorkspaceRootResolved() string {
	if c.Workspace.Root != "" {
		return c.Workspace.Root
	}
	return workspaceBase()
}
