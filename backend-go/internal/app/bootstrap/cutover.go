package bootstrap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/xmz14/lll/backend-go/internal/modules/assets"
	"github.com/xmz14/lll/backend-go/internal/modules/assistant"
	projects "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

// CutoverLearningAssets returns the projects blocked from all new work.
func CutoverLearningAssets() map[string]bool {
	blocked := map[string]bool{}
	index, err := projects.IndexAll()
	if err != nil {
		blocked["workspace-index"] = true
		return blocked
	}
	for _, project := range index {
		if project.ProjectType != projects.ProjectTypeSystemLearning {
			continue
		}
		root, err := projects.ProjectRootForSlug(project.Slug)
		if err == nil {
			var journal struct {
				State string `json:"state"`
			}
			data, readErr := os.ReadFile(filepath.Join(root, "migrations", "iteration-18.json"))
			if readErr == nil {
				err = json.Unmarshal(data, &journal)
			} else if !errors.Is(readErr, os.ErrNotExist) {
				err = readErr
			}
			if err == nil && journal.State != "completed" {
				err = assistant.RetireLegacyTasks(project.Slug)
			}
		}
		if err == nil {
			_, err = assets.MigrateReview(project.Slug)
		}
		if err != nil {
			blocked[project.Slug] = true
			println("review cutover blocked:", project.Slug, err.Error())
		}
	}
	return blocked
}
