// Package assistant is the public boundary for assistant tasks and agent CLI execution.
package assistant

import (
	"context"
	"time"

	execution "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/execution"
	launch "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/launch"
	prompt "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/prompt"
	registry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
	runtime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"
	tasks "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/tasks"
)

const (
	GlobalConcurrency     = tasks.GlobalConcurrency
	UnitConcurrency       = tasks.UnitConcurrency
	TaskSchemaVersion     = tasks.SchemaVersion
	MaxRequiredPrimitives = registry.MaxRequiredPrimitives

	RuntimeClaude    = runtime.RuntimeClaude
	RuntimeCodeBuddy = runtime.RuntimeCodeBuddy
	RuntimeHermes    = runtime.RuntimeHermes
	RuntimeCodex     = runtime.RuntimeCodex
	RuntimeTrae      = runtime.RuntimeTrae
	PromptArg        = runtime.PromptArg
	PromptClipboard  = runtime.PromptClipboard
)

type (
	Task                   = tasks.Task
	Origin                 = tasks.Origin
	Result                 = tasks.Result
	Failure                = tasks.Failure
	Lease                  = tasks.Lease
	CreateInput            = tasks.CreateInput
	TaskStore              = tasks.Store
	Dispatcher             = tasks.Dispatcher
	Dependencies           = tasks.Dependencies
	AssetSnapshot          = tasks.AssetSnapshot
	AssetCommitInput       = tasks.AssetCommitInput
	SourceSnapshot         = tasks.SourceSnapshot
	SameTypeActiveError    = tasks.SameTypeActiveError
	OperationConflictError = tasks.OperationConflictError

	ProjectRequest   = execution.ProjectRequest
	TaskRequest      = execution.TaskRequest
	ResumeRequest    = execution.ResumeRequest
	HeadlessRequest  = execution.HeadlessRequest
	ExecutionService = execution.Service
	RuntimeSource    = execution.RuntimeSource

	Agent        = registry.Agent
	OutputTarget = registry.OutputTarget
	Primitives   = registry.Primitives
	Registry     = registry.Registry

	RuntimeID      = runtime.ID
	PromptDelivery = runtime.PromptDelivery
	RuntimeConfig  = runtime.Config
	Definition     = runtime.Definition
	Runtime        = runtime.Runtime

	LaunchRequest       = launch.LaunchRequest
	LaunchResumeRequest = launch.ResumeRequest
	RunResult           = launch.RunResult
	LaunchEventEmitter  = launch.EventEmitter

	PromptPackage       = prompt.Package
	PackageMeta         = prompt.PackageMeta
	PromptRequest       = prompt.Request
	ProjectAgentRequest = prompt.ProjectAgentRequest
)

func NewTaskStore(slug string) (*TaskStore, error) { return tasks.New(slug) }
func NewDispatcher(execution tasks.ExecutionService, events tasks.EventEmitter) *Dispatcher {
	return tasks.NewDispatcher(execution, events)
}
func IsTerminal(status string) bool  { return tasks.IsTerminal(status) }
func ValidType(taskType string) bool { return tasks.ValidType(taskType) }

func NewExecution(source RuntimeSource) *ExecutionService { return execution.New(source) }
func NewRegistry() *Registry                              { return registry.New() }
func AgentsRootForTest() string                           { return registry.AgentsRootForTest() }
func SetAgentsRootForTest(dir string)                     { registry.SetAgentsRootForTest(dir) }
func PrimitiveExists(name string) bool                    { return registry.PrimitiveExists(name) }
func PrimitivePath(name string) string                    { return registry.PrimitivePath(name) }

func LoadRuntime() (RuntimeConfig, error)                   { return runtime.Load() }
func SaveSelectedRuntime(id RuntimeID) error                { return runtime.SaveSelected(id) }
func RuntimeDefinitionByID(id RuntimeID) (Definition, bool) { return runtime.DefinitionByID(id) }
func RuntimeDefinitions() []Definition                      { return runtime.Definitions() }
func ListRuntimes(config RuntimeConfig) []Runtime           { return runtime.List(config) }
func ResolveRuntime(config RuntimeConfig) Runtime           { return runtime.Resolve(config) }
func ProbeRuntime(bin string, args []string, timeout time.Duration) bool {
	return runtime.Probe(bin, args, timeout)
}

func Launch(ctx context.Context, request LaunchRequest) (*RunResult, error) {
	return launch.Launch(ctx, request)
}
func LaunchResume(ctx context.Context, request LaunchResumeRequest) (*RunResult, error) {
	return launch.LaunchResume(ctx, request)
}
func LaunchHeadless(ctx context.Context, projectSlug, agentID, bin, model string, pkg *PromptPackage, rt *Runtime) error {
	return launch.LaunchHeadless(ctx, projectSlug, agentID, bin, model, pkg, rt)
}
func LaunchTaskTerminal(rt Runtime, attemptWorkspace, promptPath, wrapperPath string, env map[string]string, onExit func(int)) error {
	return launch.LaunchTaskTerminal(rt, attemptWorkspace, promptPath, wrapperPath, env, onExit)
}
func NormalizePermissionMode(mode string) string { return launch.NormalizePermissionMode(mode) }

func BuildPrompt(request PromptRequest, reg *Registry) (*PromptPackage, error) {
	return prompt.Build(request, reg)
}
func BuildProjectAgent(request ProjectAgentRequest, reg *Registry) (*PromptPackage, error) {
	return prompt.BuildProjectAgent(request, reg)
}
func MakeRunDirName(agentID string, at time.Time) string { return prompt.MakeRunDirName(agentID, at) }
func ClearPrimitiveCacheForTest()                        { prompt.ClearPrimitiveCacheForTest() }
func ExpandPrimitives(required, optional []string) (string, error) {
	return prompt.ExpandPrimitives(required, optional)
}
func LoadPrimitive(name string) (string, error) { return prompt.LoadPrimitive(name) }
