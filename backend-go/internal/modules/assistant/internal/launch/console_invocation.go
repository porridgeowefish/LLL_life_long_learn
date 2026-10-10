package claudelauncher

import (
	"errors"
	"strings"
)

// A terminal spawner exiting is not evidence that its CLI has finished.
func terminalInvocation(goos string, available func(string) bool, wrapper, workDir string) (string, []string, error) {
	if goos == "darwin" {
		if !available("osascript") {
			return "", nil, errors.New("macOS Terminal launcher unavailable")
		}
		command := "cd " + shQuote(workDir) + " && /bin/sh " + shQuote(wrapper)
		appleQuote := func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) }
		return "osascript", []string{"-e", `tell application "Terminal" to do script "` + appleQuote(command) + `"`, "-e", `tell application "Terminal" to activate`}, nil
	}
	if goos == "linux" {
		if available("gnome-terminal") {
			return "gnome-terminal", []string{"--", "/bin/sh", wrapper}, nil
		}
		if available("konsole") {
			return "konsole", []string{"--workdir", workDir, "-e", "/bin/sh", wrapper}, nil
		}
		if available("xfce4-terminal") {
			return "xfce4-terminal", []string{"--working-directory=" + workDir, "--command=/bin/sh " + shQuote(wrapper)}, nil
		}
		if available("xterm") {
			return "xterm", []string{"-e", "/bin/sh", wrapper}, nil
		}
		return "", nil, errors.New("no graphical terminal found; install gnome-terminal, konsole, xfce4-terminal or xterm")
	}
	return "", nil, errors.New("native terminal unsupported on this OS")
}
