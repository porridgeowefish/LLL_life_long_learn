// Command lll is the Local Learning Lab HTTP server entry point.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/server"
)

func main() {
	srv := server.New()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}
	addr := ":" + port

	fmt.Printf("LifeLongLearn server\n")
	fmt.Printf("  Workspace: %s\n", paths.WORKSPACE)
	fmt.Printf("  Projects:  %s\n", paths.PROJECTS_ROOT)
	fmt.Printf("  Agents:    %s\n", paths.AGENTS_ROOT)
	fmt.Printf("  Frontend:  %s\n", paths.FRONTEND_ROOT)
	fmt.Printf("  Claude:    %s (available=%t)\n", srv.ClaudeBin, srv.ClaudeAvailable)
	fmt.Printf("  Runtime:   %s / %s (available=%t)\n", srv.Runtime.ID, srv.Runtime.Bin, srv.Runtime.Available)
	fmt.Printf("Listening on http://localhost:%s\n", port)

	httpServer := &http.Server{Addr: addr, Handler: srv.Handler()}

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
