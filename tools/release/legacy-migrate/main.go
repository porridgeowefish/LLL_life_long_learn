// Command legacy-migrate is embedded inside the MSI. Only a confirmed migration
// invokes it; it never executes a command from the registry.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func validateLegacy(root string) error {
	data, err := os.ReadFile(filepath.Join(root, ".learn-with-ai-install.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Identity string   `json:"identity"`
		Files    []string `json:"files"`
	}
	if err = json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Identity != "learn-with-ai" || len(manifest.Files) == 0 {
		return fmt.Errorf("unowned legacy installation")
	}
	found := false
	for _, name := range manifest.Files {
		if name == "Uninstall-learn-with-ai.exe" {
			found = true
		}
		if name == "" || strings.ContainsAny(name, `\:`) || strings.HasPrefix(name, "/") {
			return fmt.Errorf("unsafe legacy file")
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." || part == "." || part == "" || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
				return fmt.Errorf("unsafe legacy path")
			}
		}
		for current := filepath.Join(root, filepath.FromSlash(name)); ; current = filepath.Dir(current) {
			info, e := os.Lstat(current)
			if e != nil && !os.IsNotExist(e) {
				return e
			}
			if e == nil && info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("linked legacy path")
			}
			if current == filepath.Dir(current) {
				break
			}
		}
	}
	if !found {
		return fmt.Errorf("legacy uninstaller is not owned")
	}
	return nil
}
func migrate() error {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return fmt.Errorf("LOCALAPPDATA missing")
	}
	root := filepath.Join(local, "Programs", "learn-with-ai")
	if err := validateLegacy(root); err != nil {
		return err
	}
	exe := filepath.Join(root, "Uninstall-learn-with-ai.exe")
	cmd := exec.Command(exe, "--uninstall", "--silent")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(exe); os.IsNotExist(err) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("legacy uninstaller could not remove itself")
}
func main() {
	if err := migrate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
