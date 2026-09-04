package agentexecution

import (
	"context"
	"testing"

	claudelauncher "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/launch"
	promptassembly "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/prompt"
	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
)

func availableRuntime() agentruntime.Runtime {
	def, _ := agentruntime.DefinitionByID(agentruntime.RuntimeClaude)
	return agentruntime.Runtime{Definition: def, Bin: "claude", Available: true, Mode: "native"}
}

func TestProjectAndTaskUseOneExecutionService(t *testing.T) {
	service := New(availableRuntime)
	projectCalled, taskCalled := false, false
	service.launchProject = func(_ context.Context, req claudelauncher.LaunchRequest) (*claudelauncher.RunResult, error) {
		projectCalled = req.Agent.ID == "encyclopedia" && req.Runtime != nil
		return &claudelauncher.RunResult{}, nil
	}
	service.launchTask = func(rt agentruntime.Runtime, workspace, prompt, wrapper string, _ map[string]string, _ func(int)) error {
		taskCalled = rt.ID == agentruntime.RuntimeClaude && workspace == "workspace" && prompt == "prompt.md" && wrapper == "wrapper.ps1"
		return nil
	}
	_, err := service.StartProject(context.Background(), ProjectRequest{
		ProjectSlug: "physics", ContextName: "学科总览", Agent: &agentregistry.Agent{ID: "encyclopedia"},
		PromptPackage: &promptassembly.Package{}, Session: &sessionstore.Session{}, SessionStore: sessionstore.New(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.StartTask(context.Background(), TaskRequest{WorkspacePath: "workspace", PromptPath: "prompt.md", WrapperPath: "wrapper.ps1"}); err != nil {
		t.Fatal(err)
	}
	if !projectCalled || !taskCalled {
		t.Fatalf("execution service did not route both modes: project=%v task=%v", projectCalled, taskCalled)
	}
}
