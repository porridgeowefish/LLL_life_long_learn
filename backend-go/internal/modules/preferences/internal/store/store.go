// Package preferencestore owns the single workspace-global learner preference
// file. It is user-authored context: AI services may read a snapshot, but no AI
// execution path is allowed to write the canonical file.
package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

const Filename = "preferences.md"
const MaxBytes = 256 << 10

const defaultContent = `# 全局学习偏好

这份文件由你维护，LLL 的教师与助教只读。

可以记录你长期稳定的偏好，例如：

- 希望解释先给结论，再展开细节；
- 偏好的例子、语言、公式密度或代码风格；
- 希望教师如何提问、验证理解和安排练习。

不要在这里保存 API Key、密码或其他敏感信息。
`

type Snapshot struct {
	Path    string `json:"-"`
	Exists  bool   `json:"exists"`
	Content string `json:"-"`
}

func Path() string { return filepath.Join(paths.WORKSPACE, Filename) }

// Read returns one bounded snapshot without causing writes. This is the only
// operation available to AI-facing services.
func Read() (Snapshot, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{Path: path}, nil
	} else if err != nil {
		return Snapshot{}, err
	}
	if len(data) > MaxBytes {
		return Snapshot{}, errors.New("preferences file exceeds 256 KiB")
	}
	return Snapshot{Path: path, Exists: true, Content: string(data)}, nil
}

// Ensure is reserved for the explicit learner-facing editor. Opening that
// surface initializes the documented template when no preference file exists.
func Ensure() (Snapshot, error) {
	snapshot, err := Read()
	if err != nil || snapshot.Exists {
		return snapshot, err
	}
	return Write(defaultContent)
}

// Write is called only by the explicit learner-facing preferences endpoint.
func Write(content string) (Snapshot, error) {
	if len([]byte(content)) > MaxBytes {
		return Snapshot{}, errors.New("preferences file exceeds 256 KiB")
	}
	if strings.IndexByte(content, 0) >= 0 {
		return Snapshot{}, errors.New("preferences file contains NUL")
	}
	if err := workspace.AtomicWriteFile(Path(), []byte(content), 0o644); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Path: Path(), Exists: true, Content: content}, nil
}
