// Package agentruntime owns the learner-selected external CLI runtime.
package agentruntime

import (
	"context"
	command "github.com/xmz14/lll/backend-go/internal/platform/command"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	platformconfig "github.com/xmz14/lll/backend-go/internal/platform/config"
)

type ID string

const (
	RuntimeClaude    ID = "claude"
	RuntimeCodeBuddy ID = "codebuddy"
	RuntimeWorkBuddy ID = "workbuddy"
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
	Reason    string `json:"reason,omitempty"`
	Mode      string `json:"mode,omitempty"` // native | wsl
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
		{ID: RuntimeWorkBuddy, Name: "WorkBuddy CLI", DefaultBin: "wb", BinEnv: "WORKBUDDY_BIN", Description: "WorkBuddy 交互 CLI；以可见终端和剪贴板 prompt 接入。", ProbeArgs: []string{"--version"}, PromptDelivery: PromptClipboard, SupportsHeadless: false},
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
	loaded, _, err := platformconfig.Load(platformconfig.Options{})
	if err != nil {
		return Config{}, err
	}
	cfg := Config{Selected: normalizeID(ID(loaded.Assistant.Runtime)), Bins: loaded.Assistant.Bins}
	if cfg.Selected == "" {
		cfg.Selected = RuntimeClaude
	}
	if cfg.Bins == nil {
		cfg.Bins = map[string]string{}
	}
	return cfg, nil
}

func SaveSelected(id ID) error {
	if _, ok := DefinitionByID(id); !ok {
		return &unknownRuntimeError{id: id}
	}
	return platformconfig.SaveAssistantRuntime(string(id))
}

type unknownRuntimeError struct{ id ID }

func (e *unknownRuntimeError) Error() string { return "unknown agent runtime: " + string(e.id) }

func Resolve(cfg Config) Runtime {
	def, ok := DefinitionByID(cfg.Selected)
	if !ok {
		def, _ = DefinitionByID(RuntimeClaude)
	}
	bin, mode, available, reason := resolveCommand(def, cfg.Bins)
	return Runtime{Definition: def, Bin: bin, Available: available, Mode: mode, Reason: reason}
}

func List(cfg Config) []Runtime {
	defs := Definitions()
	out := make([]Runtime, len(defs))
	var wg sync.WaitGroup
	for i, def := range defs {
		wg.Add(1)
		go func(i int, def Definition) {
			defer wg.Done()
			bin, mode, available, reason := resolveCommand(def, cfg.Bins)
			out[i] = Runtime{Definition: def, Bin: bin, Available: available, Mode: mode, Reason: reason}
		}(i, def)
	}
	wg.Wait()
	return out
}

func Probe(bin string, args []string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := command.Context(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func resolveCommand(def Definition, bins map[string]string) (bin, mode string, available bool, reason string) {
	bin = strings.TrimSpace(bins[string(def.ID)])
	if value := strings.TrimSpace(os.Getenv(def.BinEnv)); value != "" {
		bin = value
	}
	if bin != "" {
		available, reason = probeIdentity(def, bin)
		return bin, "native", available, reason
	}
	reason = "not_found"
	for _, candidate := range candidates(def) {
		ok, why := probeIdentity(def, candidate)
		if ok {
			return candidate, "native", true, ""
		}
		if why != "not_found" {
			reason = why
		}
	}
	if runtime.GOOS == "windows" && canProbeWSL(def) && probeWSL(def.DefaultBin, def.ProbeArgs, 3*time.Second) {
		return def.DefaultBin, "wsl", true, ""
	}
	return def.DefaultBin, "native", false, reason
}
func probeIdentity(def Definition, bin string) (bool, string) {
	if _, err := exec.LookPath(bin); err != nil {
		return false, "not_found"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data, err := command.Context(ctx, bin, def.ProbeArgs...).CombinedOutput()
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return false, "probe_failed"
	}
	if def.ID != RuntimeCodeBuddy && def.ID != RuntimeWorkBuddy {
		return true, ""
	}
	matches := func(data []byte) bool {
		text := strings.ToLower(string(data))
		if def.ID == RuntimeCodeBuddy {
			return strings.Contains(text, "codebuddy")
		}
		return strings.Contains(text, "workbuddy") || strings.Contains(text, "work buddy")
	}
	if matches(data) {
		return true, ""
	}
	data, err = command.Context(ctx, bin, "--help").CombinedOutput()
	if err == nil && matches(data) {
		return true, ""
	}
	return false, "identity_mismatch"
}

func canProbeWSL(def Definition) bool {
	return def.ID == RuntimeClaude || def.ID == RuntimeCodex
}

func probeWSL(bin string, args []string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	parts := []string{"command -v " + shellQuote(bin) + " >/dev/null 2>&1"}
	if len(args) > 0 {
		quoted := make([]string, 0, len(args)+1)
		quoted = append(quoted, shellQuote(bin))
		for _, arg := range args {
			quoted = append(quoted, shellQuote(arg))
		}
		parts = append(parts, strings.Join(quoted, " "))
	}
	cmd := command.Context(ctx, "wsl.exe", "-e", "sh", "-lc", strings.Join(parts, " && "))
	out, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func normalizeID(id ID) ID { return id }
