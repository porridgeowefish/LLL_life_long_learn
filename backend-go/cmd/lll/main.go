// Command lll is the Local Learning Lab HTTP server entry point.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/platform/config"
	"github.com/xmz14/lll/backend-go/internal/server"
)

func main() {
	var (
		port       = flag.Int("port", 0, "server port (overrides config and LLL_SERVER_PORT)")
		workspace  = flag.String("workspace", "", "workspace root (overrides config and LLL_WORKSPACE_ROOT)")
		configPath = flag.String("config", "", "config file path (default: <workspace>/config.local.json)")
	)
	flag.Parse()

	cfg, _, err := config.Load(config.Options{Path: *configPath, WorkspaceRoot: *workspace, Port: *port})
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	for _, warning := range cfg.Warnings {
		fmt.Println("config:", warning)
	}
	// Flag/env workspace wins over the file value for the runtime roots.
	if cfg.Workspace.Root != "" {
		os.Setenv("WORKSPACE", cfg.Workspace.Root)
	}

	srv := server.New()
	defer srv.Close()

	portEnv := fmt.Sprintf("%d", cfg.Server.Port)
	if *port != 0 {
		portEnv = fmt.Sprintf("%d", *port)
	}
	if portEnv == "" {
		portEnv = "8787"
	}
	addr := ":" + portEnv

	fmt.Printf("LifeLongLearn server\n")
	fmt.Printf("  Workspace: %s\n", paths.WORKSPACE)
	fmt.Printf("  Projects:  %s\n", paths.PROJECTS_ROOT)
	fmt.Printf("  Agents:    %s\n", paths.AGENTS_ROOT)
	fmt.Printf("  Frontend:  %s\n", paths.FRONTEND_ROOT)
	fmt.Printf("  Claude:    %s (available=%t)\n", srv.ClaudeBin, srv.ClaudeAvailable)
	fmt.Printf("  Runtime:   %s / %s (available=%t)\n", srv.Runtime.ID, srv.Runtime.Bin, srv.Runtime.Available)
	fmt.Printf("Listening on http://localhost:%s\n", portEnv)

	httpServer := &http.Server{Addr: addr, Handler: srv.Handler()}
	srv.SetShutdownFunc(func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Println("\nShutting down from UI...")
		_ = httpServer.Close()
	})

	// Graceful shutdown on Ctrl+C.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down...")
		httpServer.Close()
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
