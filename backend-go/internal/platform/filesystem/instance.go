package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// AcquireWorkspaceInstance holds an OS lock until Close or process exit. The
// persistent lock file is intentionally not removed, so concurrent starters
// always compete for the same file even during shutdown.
func AcquireWorkspaceInstance(root string) (*os.File, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(root, ".learn-with-ai.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockInstance(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("workspace is already open in another Learn-With-AI instance: %w", err)
	}
	return file, nil
}
