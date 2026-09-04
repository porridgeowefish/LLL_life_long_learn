// Package bootstrap is LLL's single composition root. It owns the order in
// which configuration, stores, services, background workers, and the HTTP
// transport are assembled. During the iteration-14 migration it delegates
// the concrete wiring to httpserver.New(); later waves move construction
// here module by module.
package bootstrap

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/app/integration"
	"github.com/xmz14/lll/backend-go/internal/artifactwatch"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/imageconfig"
	"github.com/xmz14/lll/backend-go/internal/iteration13migration"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/projectindex"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/transport/httpserver"
)

// App is the fully assembled application.
type App struct {
	Server *httpserver.Server
}

// Build assembles the application with production wiring.
func Build() *App {
	bin := envOr("CLAUDE_BIN", "claude")
	runtimeCfg, err := assistant.LoadRuntime()
	if err != nil {
		println("agent-runtime: load warning:", err.Error())
	}
	registry := assistant.NewRegistry()
	if err := registry.Load(); err != nil {
		println("agent-registry: load warning:", err.Error())
	}
	migration := iteration13migration.RunAll()
	teacher.ReconcileAllInterruptedResponses()
	imageCfg, err := imageconfig.Load()
	if err != nil {
		println("image-config: load error:", err.Error())
		imageCfg = nil
	}
	broadcaster := httpx.NewBroadcaster()
	watcher, err := artifactwatch.Start(paths.PROJECTS_ROOT, broadcaster.Emit)
	if err != nil {
		println("artifactwatch: start warning:", err.Error())
	}
	teacherService := teacher.New(nil)
	teacherService.Authorizer = integration.TeacherTaskAuthorizer{}
	server := httpserver.New(httpserver.Dependencies{
		ClaudeBin: bin, ClaudeAvailable: probeBin(bin, "--version"),
		Runtime: assistant.ResolveRuntime(runtimeCfg), RuntimeOptions: assistant.ListRuntimes(runtimeCfg),
		ImageConfig: imageCfg, ImageAvailable: imageCfg != nil && imageCfg.PythonBin != "" && probeBin(imageCfg.PythonBin, "--version"),
		RunProgress: runprogress.New(), Watcher: watcher, Teacher: teacherService, Broadcaster: broadcaster,
		Agents: registry, Sessions: sessionstore.New(), Cache: projectindex.New(),
		MigrationReady: migration.Ready, MigrationFailed: migration.FailedProjects,
	})
	execution := assistant.NewExecution(server.RuntimeSnapshot)
	dispatcher := assistant.NewDispatcher(execution, broadcaster)
	server.AttachExecution(execution, dispatcher)
	teacherService.OnTask = func(projectSlug string, task teacher.DelegatedTask) {
		broadcaster.Emit("assistant-task-updated", map[string]any{"projectSlug": projectSlug, "task": task})
		dispatcher.Notify()
	}
	dispatcher.Start()
	return &App{Server: server}
}

func probeBin(bin string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
