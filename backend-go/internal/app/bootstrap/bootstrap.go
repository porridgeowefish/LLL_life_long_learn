// Package bootstrap is LLL's single composition root. It owns the order in
// which configuration, stores, services, background workers, and the HTTP
// transport are assembled. During the iteration-14 migration it delegates
// the concrete wiring to httpserver.New(); later waves move construction
// here module by module.
package bootstrap

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/compatibility/artifactwatch"
	"github.com/xmz14/lll/backend-go/internal/compatibility/iteration13migration"
	"github.com/xmz14/lll/backend-go/internal/compatibility/sessionstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/imageconfig"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	"github.com/xmz14/lll/backend-go/internal/modules/generation"
	runprogress "github.com/xmz14/lll/backend-go/internal/modules/learning"
	preferences "github.com/xmz14/lll/backend-go/internal/modules/preferences"
	projectindex "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"github.com/xmz14/lll/backend-go/internal/platform/command"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
	"github.com/xmz14/lll/backend-go/internal/transport/httpserver"
)

// App is the fully assembled application.
type App struct {
	Server *httpserver.Server
}

// Build assembles the application with production wiring.
func Build() *App {
	if err := preferences.BackupExisting(); err != nil {
		println("preferences: backup warning:", err.Error())
	}
	bin := envOr("CLAUDE_BIN", "claude")
	runtimeCfg, err := assistant.LoadRuntime()
	if err != nil {
		println("agent-runtime: load warning:", err.Error())
	}
	registry := assistant.NewRegistry()
	if err := registry.Load(); err != nil {
		println("agent-registry: load warning:", err.Error())
	}
	cutover := CutoverLearningAssets()
	migration := iteration13migration.RunExcept(cutover)
	for slug := range cutover {
		migration.FailedProjects = append(migration.FailedProjects, slug)
		migration.Ready = false
	}
	teacher.ReconcileAllInterruptedResponses()
	imageCfg, err := imageconfig.Load()
	if err != nil {
		println("image-config: load error:", err.Error())
		imageCfg = nil
	}
	broadcaster := httpx.NewBroadcaster()
	sources.ConfigureStructureEvents(func(slug string, job sources.StructureJob) {
		broadcaster.Emit("source-structure-job-updated", map[string]any{"projectSlug": slug, "sourceId": job.SourceID, "jobId": job.ID, "status": job.Status, "coverage": job.Coverage})
	})
	watcher, err := artifactwatch.Start(paths.PROJECTS_ROOT, broadcaster.Emit)
	if err != nil {
		println("artifactwatch: start warning:", err.Error())
	}
	teacherService := teacher.New(nil)
	teacherService.OnSourceReading = func(event map[string]any) {
		broadcaster.Emit("source-reading", event)
	}
	teacherService.OnSourceUpdated = func(slug string, source sources.Source) {
		broadcaster.Emit("source-updated", map[string]any{"projectSlug": slug, "source": source})
	}
	sourceProcessor := sources.NewProcessor(func(slug string, source sources.Source) {
		broadcaster.Emit("source-updated", map[string]any{"projectSlug": slug, "source": source})
	})
	sourceProcessor.ProjectBlocked = func(slug string) bool { return cutover[slug] }
	sourceProcessor.Start(context.Background())
	generationService := generation.New(nil)
	generationService.ProjectBlocked = func(slug string) bool { return cutover[slug] }
	generationService.OnUpdated = func(slug string, job generation.Job) {
		broadcaster.Emit("generation-job-updated", map[string]any{"projectSlug": slug, "jobId": job.ID, "status": job.Status, "phase": job.Phase, "progress": job.Progress})
	}
	if err := generationService.Start(context.Background()); err != nil {
		println("generation startup:", err.Error())
	}
	server := httpserver.New(httpserver.Dependencies{
		ClaudeBin: bin, ClaudeAvailable: probeBin(bin, "--version"),
		Runtime: assistant.ResolveRuntime(runtimeCfg), RuntimeOptions: assistant.ListRuntimes(runtimeCfg),
		ImageConfig: imageCfg, ImageAvailable: imageCfg != nil && imageCfg.PythonBin != "" && probeBin(imageCfg.PythonBin, "--version"),
		RunProgress: runprogress.NewRunStore(), Watcher: watcher, Teacher: teacherService, Broadcaster: broadcaster,
		Agents: registry, Sessions: sessionstore.New(), Cache: projectindex.NewIndex(),
		MigrationReady: migration.Ready, MigrationFailed: migration.FailedProjects,
		SourceProcessor: sourceProcessor,
		Generation:      generationService,
	})
	execution := assistant.NewExecution(server.RuntimeSnapshot)
	server.AttachExecution(execution, nil)
	notes := teacher.NewNoteService(nil)
	notes.ProjectBlocked = func(slug string) bool { return cutover[slug] }
	notes.OnInitializationFailure = func(slug string, err error) { println("teacher notes initialization:", slug, err.Error()) }
	notes.OnUpdated = func(slug, jobID, status, noteID string) {
		broadcaster.Emit("teacher-note-updated", map[string]any{"projectSlug": slug, "jobId": jobID, "status": status, "noteId": noteID})
	}
	if err := notes.Start(context.Background()); err != nil {
		println("teacher notes startup:", err.Error())
	}
	server.AttachNotes(notes)
	teacherService.OnCompleted = func(projectSlug, responseID string) {
		_, _, _ = server.RecordTeacherResponse(projectSlug, responseID)
		if err := notes.ObserveCompleted(projectSlug); err != nil {
			println("teacher notes:", projectSlug, err.Error())
		}
	}
	return &App{Server: server}
}

func probeBin(bin string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := command.Context(ctx, bin, args...).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
