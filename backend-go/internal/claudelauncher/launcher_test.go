package claudelauncher

import (
	"os"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestNormalizePermissionModeDefaultsToAuto(t *testing.T) {
	if got := NormalizePermissionMode(""); got != "auto" {
		t.Fatalf("NormalizePermissionMode(\"\") = %q, want auto", got)
	}
	if got := NormalizePermissionMode(" acceptEdits "); got != "acceptEdits" {
		t.Fatalf("NormalizePermissionMode trims explicit mode = %q", got)
	}
}

func TestTUIWrapperLaunchesClaudeInAutoMode(t *testing.T) {
	script := buildTUIWrapperScript(
		`D:\workspace\project`,
		`D:\workspace\project\runs\prompt.md`,
		`D:\workspace\project\runs\stderr.log`,
		"", // no Phase-C hook settings → no --settings arg
		LaunchRequest{
			ZoneName: workspace.ZoneExplain,
			Agent: &agentregistry.Agent{
				ID:   "explain",
				Name: "Explain",
				Icon: "E",
			},
			ClaudeBin: "claude",
		},
	)

	if !strings.Contains(script, "$permissionMode = 'auto'") {
		t.Fatalf("wrapper did not set auto permission mode:\n%s", script)
	}
	if !strings.Contains(script, "& $agentExe --permission-mode $permissionMode $promptText") {
		t.Fatalf("wrapper did not pass permission mode to claude:\n%s", script)
	}
	if strings.Contains(script, "--settings") {
		t.Fatalf("wrapper must not pass --settings when no settings file is given:\n%s", script)
	}
}

func TestTUIWrapperInjectsSettingsForClaude(t *testing.T) {
	script := buildTUIWrapperScript(
		`D:\workspace\project`,
		`D:\workspace\project\runs\prompt.md`,
		`D:\workspace\project\runs\stderr.log`,
		`D:\workspace\project\runs\claude-settings.json`,
		LaunchRequest{
			ZoneName: workspace.ZoneExplain,
			Agent: &agentregistry.Agent{
				ID:   "explain",
				Name: "Explain",
				Icon: "E",
			},
			ClaudeBin: "claude",
		},
	)
	if !strings.Contains(script, "--settings 'D:\\workspace\\project\\runs\\claude-settings.json'") {
		t.Fatalf("claude wrapper must inject --settings <file>:\n%s", script)
	}
}

func TestWriteHookSettingsRegistersRunAndToken(t *testing.T) {
	dir := t.TempDir()
	store := runprogress.New()
	path, token, err := writeHookSettings(dir, "run-xyz", store)
	if err != nil {
		t.Fatalf("writeHookSettings error: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}
	if !strings.HasSuffix(path, "claude-settings.json") {
		t.Fatalf("unexpected settings path: %s", path)
	}
	// The registered token must authorize updates for this runId.
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read settings: %v", readErr)
	}
	body := string(data)
	if !strings.Contains(body, "X-Run-Token: "+token) {
		t.Errorf("settings JSON missing the run token:\n%s", body)
	}
	if !strings.Contains(body, "/api/runs/run-xyz/status") {
		t.Errorf("settings JSON missing the run-status URL:\n%s", body)
	}
	if !strings.Contains(body, "PostToolUse") || !strings.Contains(body, "Stop") {
		t.Errorf("settings JSON missing PostToolUse/Stop hooks:\n%s", body)
	}
	// Wrong token must be rejected by the store.
	if _, ok := store.Set("run-xyz", "wrong", runprogress.Status{Activity: "x"}); ok {
		t.Errorf("store accepted a wrong token")
	}
}

func TestResumeWrapperLaunchesClaudeContinue(t *testing.T) {
	script := buildResumeWrapperScript(
		`D:\workspace\project`,
		`D:\workspace\project\runs\resume\stderr.log`,
		ResumeRequest{
			ProjectSlug: "demo",
			ZoneName:    workspace.ZoneExplain,
			ClaudeBin:   "claude",
		},
	)

	if !strings.Contains(script, "& $agentExe -c") {
		t.Fatalf("resume wrapper did not call claude -c:\n%s", script)
	}
	if strings.Contains(script, "$promptText") {
		t.Fatalf("resume wrapper should not inject a new prompt:\n%s", script)
	}
}
