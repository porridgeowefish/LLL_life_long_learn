package claudelauncher

import (
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
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
}
