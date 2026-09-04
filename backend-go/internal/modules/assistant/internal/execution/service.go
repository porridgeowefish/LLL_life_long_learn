// Package agentexecution owns the reusable domain service for starting an
// Agent CLI. HTTP routes, teachers, source processors, and task dispatchers do
// not launch terminals directly; they describe an execution to this service.
package agentexecution

import (
	"context"
	"errors"

	claudelauncher "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/launch"
	promptassembly "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/prompt"
	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
	agentruntime "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/runtime"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

type EventEmitter interface{ Emit(string, any) }

type RuntimeSource func() agentruntime.Runtime

// ProjectRequest starts an Agent against a canonical learning project. It is
// used by encyclopedia and legacy zone agents.
type ProjectRequest struct {
	ProjectSlug    string
	ContextName    string
	Agent          *agentregistry.Agent
	PromptPackage  *promptassembly.Package
	PermissionMode string
	Session        *sessionstore.Session
	SessionStore   *sessionstore.Store
	Events         EventEmitter
	RunProgress    *runprogress.Store
}

// TaskRequest starts the same visible, prompt-injected CLI surface inside an
// isolated assistant-task workspace.
type TaskRequest struct {
	WorkspacePath string
	PromptPath    string
	WrapperPath   string
	Environment   map[string]string
	OnExit        func(int)
}

type ResumeRequest struct {
	ProjectSlug  string
	ContextName  string
	RunDirName   string
	Session      *sessionstore.Session
	SessionStore *sessionstore.Store
	Events       EventEmitter
	RunProgress  *runprogress.Store
}

type HeadlessRequest struct {
	ProjectSlug   string
	AgentID       string
	Model         string
	PromptPackage *promptassembly.Package
}

type projectLauncher func(context.Context, claudelauncher.LaunchRequest) (*claudelauncher.RunResult, error)
type taskLauncher func(agentruntime.Runtime, string, string, string, map[string]string, func(int)) error
type resumeLauncher func(context.Context, claudelauncher.ResumeRequest) (*claudelauncher.RunResult, error)

// Service is the single domain entry point for visible Agent execution.
type Service struct {
	runtime       RuntimeSource
	launchProject projectLauncher
	launchTask    taskLauncher
	launchResume  resumeLauncher
}

func New(runtime RuntimeSource) *Service {
	return &Service{runtime: runtime, launchProject: claudelauncher.Launch, launchTask: claudelauncher.LaunchTaskTerminal, launchResume: claudelauncher.LaunchResume}
}

func (s *Service) Resume(ctx context.Context, req ResumeRequest) (*claudelauncher.RunResult, error) {
	if s == nil || s.runtime == nil || req.ProjectSlug == "" || req.RunDirName == "" {
		return nil, errors.New("invalid agent resume execution")
	}
	runtime := s.runtime()
	if !runtime.Available {
		return nil, errors.New("selected agent runtime is unavailable")
	}
	return s.launchResume(ctx, claudelauncher.ResumeRequest{
		ProjectSlug: req.ProjectSlug, ZoneName: workspace.ZoneName(req.ContextName), Session: req.Session,
		Store: req.SessionStore, Events: req.Events, Runtime: &runtime, RunDirName: req.RunDirName, RunProgress: req.RunProgress,
	})
}

func (s *Service) StartHeadless(ctx context.Context, req HeadlessRequest) error {
	if s == nil || s.runtime == nil || req.ProjectSlug == "" || req.AgentID == "" || req.PromptPackage == nil {
		return errors.New("invalid headless agent execution")
	}
	runtime := s.runtime()
	if !runtime.Available {
		return errors.New("selected agent runtime is unavailable")
	}
	return claudelauncher.LaunchHeadless(ctx, req.ProjectSlug, req.AgentID, runtime.Bin, req.Model, req.PromptPackage, &runtime)
}

func (s *Service) StartProject(ctx context.Context, req ProjectRequest) (*claudelauncher.RunResult, error) {
	if s == nil || s.runtime == nil || req.Agent == nil || req.PromptPackage == nil || req.Session == nil {
		return nil, errors.New("invalid project agent execution")
	}
	runtime := s.runtime()
	if !runtime.Available {
		return nil, errors.New("selected agent runtime is unavailable")
	}
	return s.launchProject(ctx, claudelauncher.LaunchRequest{
		ProjectSlug: req.ProjectSlug, ZoneName: workspace.ZoneName(req.ContextName), Agent: req.Agent,
		PromptPackage: req.PromptPackage, PermissionMode: req.PermissionMode, Session: req.Session,
		Store: req.SessionStore, Events: req.Events, Runtime: &runtime, RunProgress: req.RunProgress,
	})
}

func (s *Service) StartTask(_ context.Context, req TaskRequest) error {
	if s == nil || s.runtime == nil || req.WorkspacePath == "" || req.PromptPath == "" || req.WrapperPath == "" {
		return errors.New("invalid assistant task execution")
	}
	runtime := s.runtime()
	if !runtime.Available {
		return errors.New("selected agent runtime is unavailable")
	}
	return s.launchTask(runtime, req.WorkspacePath, req.PromptPath, req.WrapperPath, req.Environment, req.OnExit)
}
