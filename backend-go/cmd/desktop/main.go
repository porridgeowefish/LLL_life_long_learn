// Command desktop runs Learn-With-AI in its own Windows application window.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xmz14/lll/backend-go/internal/app/desktop"
)

var Version = "0.5.0-dev"

func main() { os.Exit(run()) }
func run() int {
	version := flag.Bool("version", false, "print version")
	headless := flag.Bool("no-browser", false, "run private HTTP engine without a window")
	workspace := flag.String("workspace", "", "learner workspace")
	port := flag.Int("port", 0, "private IPv4 loopback port (default: assigned free port)")
	config := flag.String("config", "", "configuration file")
	smoke := flag.String("desktop-smoke-result", "", "write rendered native page evidence and exit (requires isolated --workspace)")
	flag.Parse()
	if *version {
		fmt.Println("Learn-With-AI " + Version)
		return 0
	}
	if *smoke != "" && (*workspace == "" || *headless) {
		fmt.Fprintln(os.Stderr, "desktop smoke requires --workspace and a native window")
		return 1
	}
	engine, err := desktop.Start(desktop.Options{Workspace: *workspace, Config: *config, Port: *port})
	if err != nil {
		if !*headless && *workspace != "" && desktop.FocusExisting(*workspace) {
			return 0
		}
		if !*headless && *workspace == "" { // Resolve the same per-user root used by Start without attaching to a server.
			if desktop.FocusExisting(desktop.DefaultWorkspace()) {
				return 0
			}
		}
		fmt.Fprintln(os.Stderr, err)
		if !*headless {
			desktop.ShowError(err.Error())
		}
		return 1
	}
	defer engine.Close()
	fmt.Printf("Learn-With-AI %s\nWorkspace: %s\nListening on %s\n", Version, engine.Workspace, engine.URL)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case <-signals:
			engine.Close()
		case <-engine.Done:
		}
	}()
	if *headless {
		<-engine.Done
		return 0
	}
	if err = desktop.RunWindow(engine, *smoke); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if *smoke == "" {
			desktop.ShowError(err.Error())
		}
		return 1
	}
	return 0
}
