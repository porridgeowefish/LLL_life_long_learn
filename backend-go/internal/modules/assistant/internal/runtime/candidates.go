package agentruntime

import (
	"os"
	"path/filepath"
	"runtime"
)

func candidates(def Definition) []string {
	names := []string{def.DefaultBin}
	if def.ID == RuntimeCodeBuddy {
		names = append(names, "cbc", "codebuddy-code")
	}
	if def.ID == RuntimeWorkBuddy {
		names = append(names, "workbuddy")
	}
	out := append([]string(nil), names...)
	home, _ := os.UserHomeDir()
	dirs := []string{filepath.Join(home, ".local", "bin"), "/usr/local/bin", "/opt/homebrew/bin"}
	if runtime.GOOS == "windows" {
		dirs = []string{filepath.Join(os.Getenv("APPDATA"), "npm"), filepath.Join(os.Getenv("LOCALAPPDATA"), "codebuddy", "bin"), filepath.Join(home, ".local", "bin")}
	}
	for _, dir := range dirs {
		for _, name := range names {
			out = append(out, filepath.Join(dir, name))
			if runtime.GOOS == "windows" {
				out = append(out, filepath.Join(dir, name+".exe"), filepath.Join(dir, name+".cmd"))
			}
		}
	}
	return out
}
