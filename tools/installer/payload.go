package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const manifestName = ".learn-with-ai-install.json"
const uninstallerName = "Uninstall-learn-with-ai.exe"

type installation struct {
	Identity string   `json:"identity"`
	Version  string   `json:"version"`
	Files    []string `json:"files"`
}

var identityPattern = regexp.MustCompile(`^learn-with-ai(?:-[a-z0-9-]+)?$`)

func ownedPath(root, relative string) (string, error) {
	if relative == "" || strings.ContainsAny(relative, `\:`) || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("unsafe installation path %q", relative)
	}
	for _, segment := range strings.Split(relative, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return "", fmt.Errorf("unsafe installation path %q", relative)
		}
	}
	full := filepath.Join(root, filepath.FromSlash(relative))
	// Check every existing ancestor; a directory junction is also a link in Go.
	for current := full; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("linked installation path forbidden: %s", current)
		}
		if current == filepath.Dir(current) {
			break
		}
	}
	return full, nil
}

func readManifest(root string) (installation, error) {
	var value installation
	full, err := ownedPath(root, manifestName)
	if err != nil {
		return value, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}
	if !identityPattern.MatchString(value.Identity) || len(value.Files) == 0 {
		return value, fmt.Errorf("invalid installation manifest")
	}
	for _, name := range value.Files {
		if name == manifestName {
			return value, fmt.Errorf("reserved manifest entry")
		}
		if _, err := ownedPath(root, name); err != nil {
			return value, err
		}
	}
	return value, nil
}

func writeManifest(root string, value installation) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, manifestName), append(data, '\n'), 0600)
}

func installPayload(payload *zip.Reader, root, identity string) error {
	if !identityPattern.MatchString(identity) {
		return fmt.Errorf("invalid integration identity")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if root == filepath.Dir(root) {
		return fmt.Errorf("cannot install into a filesystem root")
	}
	old, oldErr := readManifest(root)
	if oldErr != nil && !os.IsNotExist(oldErr) {
		return oldErr
	}
	if oldErr == nil && old.Identity != identity {
		return fmt.Errorf("installation identity differs from existing application")
	}
	if oldErr != nil {
		entries, err := os.ReadDir(root)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("choose an empty folder or the existing learn-with-ai installation")
		}
	}
	oldOwned := make(map[string]bool)
	for _, name := range old.Files {
		oldOwned[strings.ToLower(name)] = true
	}
	next := installation{Identity: identity, Version: Version}
	seen := make(map[string]bool)
	// Validate the whole manifest before any destination file is overwritten.
	for _, file := range payload.File {
		if !strings.HasPrefix(file.Name, "payload/") || file.FileInfo().IsDir() || !file.Mode().IsRegular() {
			return fmt.Errorf("invalid payload entry %q", file.Name)
		}
		relative := strings.TrimPrefix(file.Name, "payload/")
		if relative == manifestName || relative == uninstallerName || seen[strings.ToLower(relative)] {
			return fmt.Errorf("reserved or duplicate payload entry %q", relative)
		}
		seen[strings.ToLower(relative)] = true
		full, err := ownedPath(root, relative)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(full); err == nil && !oldOwned[strings.ToLower(relative)] {
			return fmt.Errorf("will not overwrite an unowned file: %s", relative)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
		next.Files = append(next.Files, relative)
	}
	if len(next.Files) == 0 {
		return fmt.Errorf("empty installer payload")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	for index, file := range payload.File {
		full, _ := ownedPath(root, next.Files[index])
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.CreateTemp(filepath.Dir(full), ".install-*")
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		input.Close()
		closeErr := output.Close()
		if copyErr != nil || closeErr != nil {
			os.Remove(output.Name())
			return fmt.Errorf("extract %s: %v / %v", file.Name, copyErr, closeErr)
		}
		// Windows rename cannot replace an existing file. Removing an owned file
		// first also provides a clear failure when the running application locks it.
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			os.Remove(output.Name())
			return fmt.Errorf("close learn-with-ai before upgrading: %w", err)
		}
		if err := os.Rename(output.Name(), full); err != nil {
			os.Remove(output.Name())
			return err
		}
	}
	for _, name := range old.Files {
		if name == uninstallerName {
			next.Files = append(next.Files, name)
			continue
		}
		if seen[strings.ToLower(name)] {
			continue
		}
		full, _ := ownedPath(root, name)
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	sort.Strings(next.Files)
	return writeManifest(root, next)
}

func removePayload(root string, value installation) error {
	for _, relative := range value.Files {
		if relative == uninstallerName {
			continue
		} // Deleted after the uninstaller exits.
		full, err := ownedPath(root, relative)
		if err != nil {
			return err
		}
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("close learn-with-ai before uninstalling: %w", err)
		}
		for directory := filepath.Dir(full); directory != root && directory != filepath.Dir(directory); directory = filepath.Dir(directory) {
			if err := os.Remove(directory); err != nil {
				break
			}
		}
	}
	return os.Remove(filepath.Join(root, manifestName))
}
