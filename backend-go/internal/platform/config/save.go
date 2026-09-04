// Section-preserving save for explicit settings flows. The loader never
// writes; only these APIs write, and only behind an explicit user action
// (Settings PUT routes). They round-trip unknown top-level keys untouched.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// pathOverride lets tests redirect the file location.
var pathOverride string

// UsePathForTest redirects the config file path and returns a restore func.
func UsePathForTest(path string) (restore func()) {
	old := pathOverride
	pathOverride = path
	return func() { pathOverride = old }
}

func activePath(explicit string) string {
	if pathOverride != "" {
		return pathOverride
	}
	return filePath(explicit)
}

// SaveUI writes the ui section, preserving every other top-level key.
func SaveUI(theme string) error {
	raw, err := readRaw(activePath(""))
	if err != nil {
		return err
	}
	raw["ui"] = map[string]any{"theme": theme}
	return writeRaw(activePath(""), raw)
}

// SaveAssistantRuntime writes assistant.runtime (sectioned when the
// assistant section exists, otherwise the legacy flat agentRuntime key so
// older builds stay readable during the compatibility period).
func SaveAssistantRuntime(runtime string) error {
	raw, err := readRaw(activePath(""))
	if err != nil {
		return err
	}
	if _, ok := raw["assistant"]; ok {
		section := map[string]any{}
		if existing, ok := raw["assistant"].(map[string]any); ok {
			section = existing
		}
		section["runtime"] = runtime
		raw["assistant"] = section
	} else {
		raw["agentRuntime"] = runtime
	}
	return writeRaw(activePath(""), raw)
}

// AISettings mirrors askaiconfig.Config for the settings flows.
type AISettings struct {
	Default      string             `json:"default"`
	SearchEngine string             `json:"searchEngine"`
	Providers    []Provider         `json:"providers"`
	Bindings     map[string]Binding `json:"bindings,omitempty"`
}

// SaveAI writes the askAiProviders section under its legacy key so the
// existing local Settings flow and older builds keep reading it. The
// canonical 'ai' section is read-preferred during the compatibility period.
func SaveAI(settings AISettings) error {
	raw, err := readRaw(activePath(""))
	if err != nil {
		return err
	}
	raw["askAiProviders"] = settings
	return writeRaw(activePath(""), raw)
}

// LoadAI reads the effective AI settings: sectioned 'ai' wins over the
// legacy 'askAiProviders' key.
func LoadAI() (AISettings, bool, error) {
	var out AISettings
	data, err := os.ReadFile(activePath(""))
	if err != nil {
		if os.IsNotExist(err) {
			return out, false, nil
		}
		return out, false, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return out, false, err
	}
	if section, ok := raw["ai"]; ok {
		var ai AI
		if err := json.Unmarshal(section, &ai); err != nil {
			return out, false, err
		}
		out = AISettings{Default: ai.DefaultProvider, SearchEngine: ai.SearchEngine, Providers: ai.Providers, Bindings: ai.Bindings}
		return out, true, nil
	}
	if section, ok := raw["askAiProviders"]; ok {
		if err := json.Unmarshal(section, &out); err != nil {
			return out, false, err
		}
		return out, true, nil
	}
	return out, false, nil
}

func readRaw(path string) (map[string]any, error) {
	raw := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return raw, nil
}

func writeRaw(path string, raw map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWrite(path, append(out, '\n')); err != nil {
		return err
	}
	return nil
}

// atomicWrite is the sibling-temp+rename primitive (Windows-safe: same
// volume). Mirrors workspace.AtomicWriteFile until platform/filesystem
// lands in a later wave; then this delegates.
func atomicWrite(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// SortedKeys is a small helper for deterministic section iteration in tests.
func SortedKeys(raw map[string]any) []string {
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ = strings.TrimSpace
