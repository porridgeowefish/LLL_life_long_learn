// backend-go/internal/askaiconfig/askaiconfig_test.go
package askaiconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func useTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.local.json")
	t.Cleanup(UseConfigPathForTest(path))
	return path
}

func TestLoadAbsentReturnsNil(t *testing.T) {
	useTempConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when file absent, got %+v", cfg)
	}
}

func TestLoadParsesProviders(t *testing.T) {
	path := useTempConfig(t)
	os.WriteFile(path, []byte(`{"imageApiKey":"keep-me","askAiProviders":{
		"default":"deepseek","searchEngine":"google",
		"providers":[{"id":"deepseek","kind":"openai","baseURL":"https://x/v1","apiKey":"sk-1","model":"m","reasoning":true}]}}`), 0o644)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.Default != "deepseek" || len(cfg.Providers) != 1 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.Providers[0].APIKey != "sk-1" || !cfg.Providers[0].Reasoning {
		t.Errorf("provider not parsed: %+v", cfg.Providers[0])
	}
	if !cfg.Enabled() {
		t.Errorf("expected Enabled=true")
	}
}

func TestFindByIDThenDefaultThenFirst(t *testing.T) {
	cfg := &Config{Default: "b", Providers: []Provider{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if cfg.Find("c").ID != "c" {
		t.Errorf("Find by id failed")
	}
	if cfg.Find("").ID != "b" {
		t.Errorf("Find default failed")
	}
	cfg.Default = "zzz"
	if cfg.Find("").ID != "a" {
		t.Errorf("Find fallback-to-first failed")
	}
}

func TestResolveServiceBindingOverridesModel(t *testing.T) {
	cfg := &Config{Default: "a", Providers: []Provider{{ID: "a", Model: "fast"}, {ID: "b", Model: "base"}}, Bindings: map[string]Binding{"teacher": {ProviderID: "b", Model: "teacher-model"}}}
	resolved := cfg.Resolve("teacher")
	if resolved == nil || resolved.ID != "b" || resolved.Model != "teacher-model" {
		t.Fatalf("unexpected binding resolution: %#v", resolved)
	}
	if cfg.Providers[1].Model != "base" {
		t.Fatal("Resolve mutated stored provider")
	}
}

func TestSavePreservesOtherKeys(t *testing.T) {
	path := useTempConfig(t)
	os.WriteFile(path, []byte(`{"imageApiKey":"keep-me","agentRuntime":"claude"}`), 0o644)
	if err := Save(Config{Default: "p1", SearchEngine: "bing", Providers: []Provider{{ID: "p1", Kind: "openai", APIKey: "sk"}}}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg == nil || cfg.Default != "p1" || cfg.SearchEngine != "bing" {
		t.Fatalf("round-trip failed: %+v %v", cfg, err)
	}
	// Other keys preserved?
	raw, _ := os.ReadFile(path)
	s := string(raw)
	if !contains(s, "\"imageApiKey\":") || !contains(s, "\"keep-me\"") || !contains(s, "\"agentRuntime\":") {
		t.Errorf("Save clobbered other keys: %s", s)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
