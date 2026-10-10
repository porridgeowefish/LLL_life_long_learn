package usagestore

import (
	"fmt"
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

func TestProjectUsageIncludesAll25ProjectsAndSeparatesNotes(t *testing.T) {
	workspace.SetProjectsRootForTest(t.TempDir())
	defer workspace.SetProjectsRootForTest("")
	for i := 0; i < 25; i++ {
		slug := fmt.Sprintf("topic-%d", i)
		if err := workspace.CreateProjectSkeletonWithInput(slug, slug, "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
			t.Fatal(err)
		}
		r := Record{ConversationID: "c", UnitID: "u", ResponseID: "r", InputTokens: 10, OutputTokens: 20}
		if err := Append(slug, r); err != nil {
			t.Fatal(err)
		}
		r.InputTokens, r.OutputTokens = 3, 4
		if err := AppendNote(slug, r); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := ListProjects()
	if err != nil || len(rows) != 25 {
		t.Fatal("missing projects", len(rows), err)
	}
	for _, row := range rows {
		if row.InputTokens != 10 || row.OutputTokens != 20 || row.NoteInputTokens != 3 || row.NoteOutputTokens != 4 {
			t.Fatal("mixed token ledgers", row)
		}
	}
}
