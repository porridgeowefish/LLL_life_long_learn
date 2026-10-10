// Command desktop runs Learn-With-AI in its own Windows application window.
package main

import (
	"errors"
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
	engine, err := desktop.Start(desktop.Options{Workspace: *workspace, Config: *config, Port: *port, Version: Version})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var busy *desktop.BusyError
		if errors.As(err, &busy) {
			// A second launch brings the running window forward instead of failing.
			if !*headless && *smoke == "" && desktop.FocusExisting(busy.Workspace) {
				return 0
			}
			if !*headless {
				desktop.ShowError("Learn-With-AI 已经在运行，但没有找到它的窗口。\n\n学习数据正被另一个 Learn-With-AI 程序使用（例如开发版服务）。请先关闭那个程序，再重新打开。\n\n数据目录：" + busy.Workspace)
			}
			return 1
		}
		if !*headless {
			desktop.ShowError("Learn-With-AI 启动失败：\n\n" + err.Error())
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
