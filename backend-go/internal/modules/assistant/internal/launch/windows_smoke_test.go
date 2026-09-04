//go:build windows

package claudelauncher

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"
)

func TestWindowsWrapperPreservesLongQuotedPromptAndChineseWorkspace(t *testing.T) {
	repoRoot := findRepoRoot(t)
	temp := t.TempDir()
	receiver := filepath.Join(temp, "fake-agent.exe")
	build := exec.Command("go", "build", "-o", receiver, "./tests/smoke/fake-agent")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake receiver: %v\n%s", err, output)
	}

	workspaceDir := filepath.Join(temp, "中文项目")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prompt := strings.Repeat("\"", 62) + strings.Repeat("x", 4360)
	if len(prompt) != 4422 {
		t.Fatalf("test prompt length = %d", len(prompt))
	}
	promptPath := filepath.Join(temp, "prompt.md")
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		t.Fatal(err)
	}

	def, _ := agentruntime.DefinitionByID(agentruntime.RuntimeClaude)
	runtime := agentruntime.Runtime{Definition: def, Bin: receiver, Available: true, Mode: "native"}
	script := buildTaskWrapperPowerShell(runtime, workspaceDir, promptPath, nil)
	// Production wrappers wait for a key so the visible terminal remains
	// inspectable. The smoke runs headlessly and removes only that final wait.
	script = strings.ReplaceAll(script, `$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')`, "")
	wrapperPath := filepath.Join(temp, "wrapper.ps1")
	wrapperBytes := append([]byte{0xEF, 0xBB, 0xBF}, []byte(script)...)
	if err := os.WriteFile(wrapperPath, wrapperBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	gotWrapper, err := os.ReadFile(wrapperPath)
	if err != nil || len(gotWrapper) < 3 || string(gotWrapper[:3]) != string([]byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("wrapper missing UTF-8 BOM: %v %x", err, gotWrapper[:min(3, len(gotWrapper))])
	}

	recordPath := filepath.Join(temp, "invocation.json")
	command := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", wrapperPath)
	command.Env = append(os.Environ(), "LLL_SMOKE_OUTPUT="+recordPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run wrapper: %v\n%s", err, output)
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Args []string `json:"args"`
		CWD  string   `json:"cwd"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	wantPrompt := strings.ReplaceAll(prompt, `"`, "“")
	if len(record.Args) != 3 || record.Args[2] != wantPrompt {
		t.Fatalf("prompt argv mismatch: argc=%d lengths=%v", len(record.Args), argumentLengths(record.Args))
	}
	gotInfo, gotErr := os.Stat(record.CWD)
	wantInfo, wantErr := os.Stat(workspaceDir)
	if gotErr != nil || wantErr != nil || !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("receiver cwd = %q, want %q", record.CWD, workspaceDir)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func argumentLengths(args []string) []int {
	lengths := make([]int, len(args))
	for i := range args {
		lengths[i] = len(args[i])
	}
	return lengths
}
