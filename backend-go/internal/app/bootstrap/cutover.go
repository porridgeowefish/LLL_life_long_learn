package bootstrap

import (
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
		err := assistant.RetireForAPIGeneration(project.Slug)
		if err == nil && project.ProjectType == projects.ProjectTypeSystemLearning {
			_, err = assets.MigrateReview(project.Slug)
		}
		if err != nil {
			blocked[project.Slug] = true
			println("review cutover blocked:", project.Slug, err.Error())
		}
	}
	return blocked
}
