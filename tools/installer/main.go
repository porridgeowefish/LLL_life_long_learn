// Command installer builds the per-user Windows installer. A trusted release
// ZIP is appended to this executable by tools/release/build-installers.js.
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var Version = "development"

func main() {
	silent := flag.Bool("silent", false, "install/uninstall without dialogs")
	uninstall := flag.Bool("uninstall", strings.EqualFold(filepath.Base(os.Args[0]), uninstallerName), "uninstall the application beside this executable")
	directory := flag.String("install-dir", "", "installation folder (default: per-user Programs)")
	identity := flag.String("integration-name", "learn-with-ai", "application integration name")
	flag.Parse()
	err := execute(*silent, *uninstall, *directory, *identity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if !*silent {
			showMessage("Learn-With-AI", err.Error(), false)
		}
		os.Exit(1)
	}
}

func execute(silent, uninstall bool, directory, identity string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if uninstall {
		root := filepath.Dir(self)
		manifest, err := readManifest(root)
		if err != nil {
			return err
		}
		if !silent && !showMessage("Learn-With-AI", "卸载 Learn-With-AI？学习数据会保留。", true) {
			return nil
		}
		if err := removePayload(root, manifest); err != nil {
			return err
		}
		if err := removeIntegration(manifest.Identity); err != nil {
			return err
		}
		if err := deleteAfterExit(self, root); err != nil {
			return err
		}
		if !silent {
			showMessage("Learn-With-AI", "已卸载。学习数据保持原样。", false)
		}
		return nil
	}
	if !identityPattern.MatchString(identity) {
		return fmt.Errorf("invalid integration name")
	}
	if directory == "" {
		directory = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", identity)
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return err
	}
	if os.Getenv("LOCALAPPDATA") == "" {
		return fmt.Errorf("LOCALAPPDATA is unavailable")
	}
	if !silent && !showMessage("Learn-With-AI 安装", "将安装到：\n"+directory+"\n\n安装仅对当前用户生效，并创建桌面和开始菜单入口。是否继续？", true) {
		return nil
	}
	payload, err := zip.OpenReader(self)
	if err != nil {
		return fmt.Errorf("installer payload is unavailable: %w", err)
	}
	defer payload.Close()
	if err := installPayload(&payload.Reader, directory, identity); err != nil {
		return err
	}
	input, err := os.Open(self)
	if err != nil {
		return err
	}
	defer input.Close()
	uninstaller := filepath.Join(directory, uninstallerName)
	output, err := os.Create(uninstaller)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("create uninstaller: %v / %v", copyErr, closeErr)
	}
	manifest, err := readManifest(directory)
	if err != nil {
		return err
	}
	found := false
	for _, name := range manifest.Files {
		if name == uninstallerName {
			found = true
		}
	}
	if !found {
		manifest.Files = append(manifest.Files, uninstallerName)
	}
	if err := writeManifest(directory, manifest); err != nil {
		return err
	}
	if err := createIntegration(directory, identity); err != nil {
		return err
	}
	if !silent && showMessage("Learn-With-AI", "安装完成。现在打开 Learn-With-AI？", true) {
		return launchApplication(directory)
	}
	return nil
}
