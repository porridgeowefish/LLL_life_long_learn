// Package agentruntime owns the learner-selected external CLI runtime.
package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/paths"
)

type ID string

const (
	RuntimeClaude    ID = "claude"
	RuntimeCodeBuddy ID = "codebuddy"
	RuntimeHermes    ID = "hermes"
	RuntimeCodex     ID = "codex"
	RuntimeTrae      ID = "trae"
)

type PromptDelivery string

const (
	PromptArg       PromptDelivery = "arg"
	PromptClipboard PromptDelivery = "clipboard"
)

type Definition struct {
	ID               ID             `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	DefaultBin       string         `json:"defaultBin"`
	BinEnv           string         `json:"binEnv"`
	ProbeArgs        []string       `json:"-"`
	PromptDelivery   PromptDelivery `json:"promptDelivery"`
	SupportsHeadless bool           `json:"supportsHeadless"`
}

type Runtime struct {
	Definition
	Bin       string `json:"bin"`
	Available bool   `json:"available"`
}

type Config struct {
	Selected ID                `json:"agentRuntime"`
	Bins     map[string]string `json:"agentRuntimeBins,omitempty"`
}

func Definitions() []Definition {
	return []Definition{
		{
			ID: RuntimeClaude, Name: "Claude Code CLI", DefaultBin: "claude", BinEnv: "CLAUDE_BIN",
			Description: "当前默认运行时；支持交互 TUI 和后台单次任务。",
			ProbeArgs:   []string{"--version"}, PromptDelivery: PromptArg, SupportsHeadless: true,
		},
		{
			ID: RuntimeCodeBuddy, Name: "CodeBuddy CLI", DefaultBin: "codebuddy", BinEnv: "CODEBUDDY_BIN",
			Description: "Tencent CodeBuddy 本地 Agent 入口；LLL 以可见终端和剪贴板 prompt 方式接入。",
			ProbeArgs:   []string{"--version"}, PromptDelivery: PromptClipboard, SupportsHeadless: false,
		},
		{
			ID: RuntimeHermes, Name: "Hermes CLI", DefaultBin: "hermes", BinEnv: "HERMES_BIN",
			Description: "Hermes Agent 终端界面；交互模式使用剪贴板预装 prompt，后台任务使用 hermes -z。",
			ProbeArgs:   []string{"--version"}, PromptDelivery: PromptClipboard, SupportsHeadless: true,
		},
		{
			ID: RuntimeCodex, Name: "Codex CLI", DefaultBin: "codex", BinEnv: "CODEX_BIN",
			Description: "OpenAI Codex 终端 Agent；交互模式支持命令行初始 prompt，后台任务使用 codex exec。",
			ProbeArgs:   []string{"--version"}, PromptDelivery: PromptArg, SupportsHeadless: true,
		},
		{
			ID: RuntimeTrae, Name: "Trae CLI", DefaultBin: "trae-cli", BinEnv: "TRAE_BIN",
			Description: "ByteDance Trae Agent 命令行入口；使用 trae-cli run 在当前项目目录执行任务。",
			ProbeArgs:   []string{"--help"}, PromptDelivery: PromptArg, SupportsHeadless: true,
		},
	}
}

func DefinitionByID(id ID) (Definition, bool) {
	for _, d := range Definitions() {
		if d.ID == id {
			return d, true
		}
	}
	return Definition{}, false
}

func Load() (Config, error) {
	cfg := Config{Selected: RuntimeClaude, Bins: map[string]string{}}
	if env := strings.TrimSpace(os.Getenv("LLL_AGENT_RUNTIME")); env != "" {
		cfg.Selected = normalizeID(ID(env))
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var disk Config
	if err := json.Unmarshal(data, &disk); err != nil {
		return cfg, err
	}
	if disk.Selected != "" {
		cfg.Selected = normalizeID(disk.Selected)
	}
	if disk.Bins != nil {
		cfg.Bins = disk.Bins
		if cfg.Bins[string(RuntimeCodeBuddy)] == "" && cfg.Bins["workbuddy"] != "" {
			cfg.Bins[string(RuntimeCodeBuddy)] = cfg.Bins["workbuddy"]
		}
	}
	if env := strings.TrimSpace(os.Getenv("LLL_AGENT_RUNTIME")); env != "" {
		cfg.Selected = normalizeID(ID(env))
	}
	return cfg, nil
}

func SaveSelected(id ID) error {
	if _, ok := DefinitionByID(id); !ok {
		return errors.New("unknown agent runtime")
	}
	path := configPath()
	raw := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	encoded, _ := json.Marshal(id)
	raw["agentRuntime"] = encoded
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func Resolve(cfg Config) Runtime {
	def, ok := DefinitionByID(cfg.Selected)
	if !ok {
		def, _ = DefinitionByID(RuntimeClaude)
	}
	bin := strings.TrimSpace(cfg.Bins[string(def.ID)])
	if env := strings.TrimSpace(os.Getenv(def.BinEnv)); env != "" {
		bin = env
	}
	if bin == "" {
		bin = def.DefaultBin
	}
	return Runtime{Definition: def, Bin: bin, Available: Probe(bin, def.ProbeArgs, 3*time.Second)}
}

func List(cfg Config) []Runtime {
	defs := Definitions()
	out := make([]Runtime, 0, len(defs))
	for _, def := range defs {
		bin := strings.TrimSpace(cfg.Bins[string(def.ID)])
		if env := strings.TrimSpace(os.Getenv(def.BinEnv)); env != "" {
			bin = env
		}
		if bin == "" {
			bin = def.DefaultBin
		}
		out = append(out, Runtime{Definition: def, Bin: bin, Available: Probe(bin, def.ProbeArgs, 3*time.Second)})
	}
	return out
}

func Probe(bin string, args []string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func configPath() string {
	root := paths.WORKSPACE
	if root == "" {
		root = paths.PROJECT_ROOT
	}
	return filepath.Join(root, "config.local.json")
}

func normalizeID(id ID) ID {
	if id == "workbuddy" {
		return RuntimeCodeBuddy
	}
	return id
}
