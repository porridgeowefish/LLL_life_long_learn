package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.local.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultsResolveAndValidate(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	if cfg.Server.Port != 8787 || cfg.Assistant.Runtime != "claude" || cfg.Assistant.MaxConcurrent != 5 || cfg.UI.Theme != "lychee-paper" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestSectionedShapeLoads(t *testing.T) {
	path := writeConfig(t, `{
  "server": {"host": "0.0.0.0", "port": 9000},
  "workspace": {"root": "D:/tmp/ws"},
  "ai": {
    "defaultProvider": "primary",
    "searchEngine": "bing",
    "providers": [{"id": "primary", "kind": "openai", "name": "P", "baseURL": "https://x", "apiKeyEnv": "LLL_PRIMARY_AI_API_KEY", "model": "m"}],
    "bindings": {"teacher": {"providerId": "primary", "model": "gpt"}}
  },
  "assistant": {"runtime": "codex", "maxConcurrent": 3, "maxConcurrentPerProject": 1, "bins": {"codex": "cx"}},
  "image": {"enabled": true, "baseURL": "https://img", "apiKeyEnv": "LLL_IMAGE_API_KEY", "model": "gpt-image-2", "pythonBin": "python3"},
  "ui": {"theme": "night-ink"}
}`)
	cfg, prov, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 9000 {
		t.Fatalf("server: %+v", cfg.Server)
	}
	if cfg.Workspace.Root != "D:/tmp/ws" {
		t.Fatalf("workspace: %+v", cfg.Workspace)
	}
	if cfg.AI.DefaultProvider != "primary" || cfg.AI.SearchEngine != "bing" || len(cfg.AI.Providers) != 1 || cfg.AI.Bindings["teacher"].Model != "gpt" {
		t.Fatalf("ai: %+v", cfg.AI)
	}
	if cfg.Assistant.Runtime != "codex" || cfg.Assistant.MaxConcurrent != 3 || cfg.Assistant.Bins["codex"] != "cx" {
		t.Fatalf("assistant: %+v", cfg.Assistant)
	}
	if !cfg.Image.Enabled || cfg.Image.Model != "gpt-image-2" || cfg.Image.PythonBin != "python3" {
		t.Fatalf("image: %+v", cfg.Image)
	}
	if cfg.UI.Theme != "night-ink" {
		t.Fatalf("ui: %+v", cfg.UI)
	}
	for _, section := range []string{"server", "workspace", "ai", "assistant", "image", "ui"} {
		if prov[section].From != "file" {
			t.Fatalf("provenance[%s] = %+v, want file", section, prov[section])
		}
	}
}

// The compatibility contract: existing flat keys resolve to the same typed
// values as the sectioned form.
func TestFlatKeysResolveEquivalent(t *testing.T) {
	flat := writeConfig(t, `{
  "agentRuntime": "codex",
  "agentRuntimeBins": {"codex": "cx"},
  "imageApiKey": "sk-flat",
  "imageBaseURL": "https://img",
  "imageModel": "gpt-image-2",
  "imageBackupApiKey": "sk-backup",
  "imageBackupBaseURL": "https://backup",
  "imageBackupModel": "gpt-image-2b",
  "imagePromptModel": "opus",
  "pythonBin": "python3",
  "askAiProviders": {
    "default": "primary",
    "searchEngine": "bing",
    "providers": [{"id": "primary", "kind": "openai", "baseURL": "https://x", "apiKey": "sk-ai", "model": "m"}],
    "bindings": {"teacher": {"providerId": "primary", "model": "gpt"}}
  },
  "ui": {"theme": "wisteria-gray"}
}`)
	cfg, _, err := Load(Options{Path: flat})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Assistant.Runtime != "codex" || cfg.Assistant.Bins["codex"] != "cx" {
		t.Fatalf("assistant flat: %+v", cfg.Assistant)
	}
	if cfg.Image.APIKey != "sk-flat" || cfg.Image.BaseURL != "https://img" || cfg.Image.Model != "gpt-image-2" || cfg.Image.Backup.APIKey != "sk-backup" || cfg.Image.PromptModel != "opus" || cfg.Image.PythonBin != "python3" {
		t.Fatalf("image flat: %+v", cfg.Image)
	}
	if cfg.AI.DefaultProvider != "primary" || cfg.AI.SearchEngine != "bing" || len(cfg.AI.Providers) != 1 || cfg.AI.Providers[0].APIKey != "sk-ai" || cfg.AI.Bindings["teacher"].ProviderID != "primary" {
		t.Fatalf("ai flat: %+v", cfg.AI)
	}
	if cfg.UI.Theme != "wisteria-gray" {
		t.Fatalf("ui flat: %+v", cfg.UI)
	}
}

func TestSectionedWinsOverFlatWithWarning(t *testing.T) {
	path := writeConfig(t, `{
  "agentRuntime": "codex",
  "assistant": {"runtime": "hermes"},
  "askAiProviders": {"default": "old", "providers": []},
  "ai": {"defaultProvider": "new", "providers": []}
}`)
	cfg, _, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Assistant.Runtime != "hermes" {
		t.Fatalf("sectioned assistant should win, got %q", cfg.Assistant.Runtime)
	}
	if cfg.AI.DefaultProvider != "new" {
		t.Fatalf("sectioned ai should win, got %q", cfg.AI.DefaultProvider)
	}
	joined := strings.Join(cfg.Warnings, "\n")
	if !strings.Contains(joined, "askAiProviders") {
		t.Fatalf("expected conflict warning mentioning askAiProviders, got: %s", joined)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	path := writeConfig(t, `{"server": {"port": 9000}}`)
	t.Setenv("LLL_SERVER_PORT", "9100")
	cfg, prov, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9100 {
		t.Fatalf("env port should win, got %d", cfg.Server.Port)
	}
	if prov["server"].From != "env" {
		t.Fatalf("provenance server = %+v", prov["server"])
	}
}

func TestFlagOverridesEnv(t *testing.T) {
	path := writeConfig(t, `{"server": {"port": 9000}}`)
	t.Setenv("LLL_SERVER_PORT", "9100")
	cfg, prov, err := Load(Options{Path: path, Port: 9200})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9200 {
		t.Fatalf("flag port should win, got %d", cfg.Server.Port)
	}
	if prov["server"].From != "flag" {
		t.Fatalf("provenance server = %+v", prov["server"])
	}
}

func TestAPIKeyEnvWinsOverInline(t *testing.T) {
	path := writeConfig(t, `{"ai": {"providers": [{"id": "p", "kind": "openai", "baseURL": "https://x", "apiKey": "inline-key", "apiKeyEnv": "LLL_TEST_KEY", "model": "m"}]}}`)
	t.Setenv("LLL_TEST_KEY", "env-key")
	cfg, _, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.AI.Providers[0].ResolveAPIKey(); got != "env-key" {
		t.Fatalf("ResolveAPIKey = %q, want env-key", got)
	}
}

func TestRedactionMasksSecrets(t *testing.T) {
	path := writeConfig(t, `{
  "ai": {"providers": [{"id": "p", "kind": "openai", "baseURL": "https://x", "apiKey": "sk-secret", "model": "m"}]},
  "image": {"apiKey": "sk-img", "backup": {"apiKey": "sk-backup"}}
}`)
	cfg, _, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	redacted := cfg.Redacted()
	if redacted.AI.Providers[0].APIKey != "••••" || redacted.Image.APIKey != "••••" || redacted.Image.Backup.APIKey != "••••" {
		t.Fatalf("redaction failed: %+v", redacted)
	}
	blob, _ := json.Marshal(redacted)
	if strings.Contains(string(blob), "sk-secret") || strings.Contains(string(blob), "sk-img") {
		t.Fatalf("redacted config still carries a secret: %s", blob)
	}
	if cfg.AI.Providers[0].APIKey != "sk-secret" {
		t.Fatal("Redacted must not mutate the receiver")
	}
}

func TestInvalidValuesFailWithActionableDiagnostics(t *testing.T) {
	cases := []struct {
		json    string
		wantErr string
	}{
		{`{"ai": {"providers": [{"id": "p", "kind": "weird"}]}}`, "kind"},
		{`{"assistant": {"runtime": "nope"}}`, "runtime"},
		{`{"ui": {"theme": "neon-pink"}}`, "theme"},
		{`{"ai": {"searchEngine": "ddg"}}`, "searchEngine"},
	}
	for _, tc := range cases {
		path := writeConfig(t, tc.json)
		_, _, err := Load(Options{Path: path})
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("Load(%s) err = %v, want containing %q", tc.json, err, tc.wantErr)
		}
	}
}

func TestWorkspacePathSafety(t *testing.T) {
	path := writeConfig(t, `{"workspace": {"root": "D:/x/../y"}}`)
	_, _, err := Load(Options{Path: path})
	if err == nil || !strings.Contains(err.Error(), "workspace.root") {
		t.Fatalf("relative workspace root err = %v", err)
	}
}

func TestLoadNeverRewritesFile(t *testing.T) {
	content := "{\n  \"agentRuntime\": \"codex\"\n}\n"
	path := writeConfig(t, content)
	if _, _, err := Load(Options{Path: path}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != content {
		t.Fatalf("file rewritten during load:\nbefore: %q\nafter:  %q", content, string(after))
	}
}

func TestSaveUIPreservesOtherSections(t *testing.T) {
	restore := UsePathForTest(writeConfig(t, `{"agentRuntime": "codex", "ui": {"theme": "lychee-paper"}, "extra": 1}`))
	defer restore()
	if err := SaveUI("night-ink"); err != nil {
		t.Fatal(err)
	}
	raw := map[string]any{}
	data, err := os.ReadFile(activePath(""))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["extra"].(float64) != 1 {
		t.Fatal("SaveUI clobbered unrelated key 'extra'")
	}
	if raw["agentRuntime"] != "codex" {
		t.Fatal("SaveUI clobbered agentRuntime")
	}
	ui := raw["ui"].(map[string]any)
	if ui["theme"] != "night-ink" {
		t.Fatalf("theme = %v", ui["theme"])
	}
}

func TestSaveAssistantRuntimeUsesSectionWhenPresent(t *testing.T) {
	restore := UsePathForTest(writeConfig(t, `{"assistant": {"runtime": "claude"}, "ui": {"theme": "lychee-paper"}}`))
	defer restore()
	if err := SaveAssistantRuntime("codex"); err != nil {
		t.Fatal(err)
	}
	raw := map[string]any{}
	data, _ := os.ReadFile(activePath(""))
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	section := raw["assistant"].(map[string]any)
	if section["runtime"] != "codex" {
		t.Fatalf("assistant.runtime = %v", section["runtime"])
	}
	if _, exists := raw["agentRuntime"]; exists {
		t.Fatal("flat agentRuntime should not be created when sectioned form exists")
	}
}

func TestLoadAISectionedWinsOverLegacy(t *testing.T) {
	restore := UsePathForTest(writeConfig(t, `{
  "askAiProviders": {"default": "old", "providers": [{"id": "old", "kind": "openai", "baseURL": "u", "apiKey": "k", "model": "m"}]},
  "ai": {"defaultProvider": "new", "providers": [{"id": "new", "kind": "anthropic", "baseURL": "u2", "apiKeyEnv": "X", "model": "m2"}]}
}`))
	defer restore()
	settings, ok, err := LoadAI()
	if err != nil || !ok {
		t.Fatalf("LoadAI err=%v ok=%v", err, ok)
	}
	if settings.Default != "new" || len(settings.Providers) != 1 || settings.Providers[0].ID != "new" {
		t.Fatalf("LoadAI = %+v", settings)
	}
}
