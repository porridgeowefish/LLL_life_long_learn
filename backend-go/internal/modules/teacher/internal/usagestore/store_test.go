package usagestore

import (
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func TestAppendAndListTeacherUsageByConversation(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "并行计算", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	if err := Append("topic", Record{ConversationID: "conv_1", UnitID: "unit_1", ResponseID: "resp_1", ProviderID: "teacher", InputTokens: 12, OutputTokens: 34}); err != nil {
		t.Fatal(err)
	}
	if err := Append("topic", Record{ConversationID: "conv_1", UnitID: "unit_1", ResponseID: "resp_2", ProviderID: "teacher", InputTokens: 8, OutputTokens: 21}); err != nil {
		t.Fatal(err)
	}
	rows, err := ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Title != "并行计算" || rows[0].InputTokens != 20 || rows[0].OutputTokens != 55 || len(rows[0].Turns) != 2 {
		t.Fatalf("unexpected usage rows: %#v", rows)
	}
}
