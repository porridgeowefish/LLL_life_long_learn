package httpserver

import (
	"testing"

	progressstore "github.com/xmz14/lll/backend-go/internal/modules/learning"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func TestRecordLearningActivityPersistsAndBroadcastsOnlyNewEvents(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	if err := workspace.CreateProjectSkeletonWithInput("activity", "学习活动", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t)
	channel, unsubscribe := server.broadcaster.Subscribe()
	defer unsubscribe()
	event := progressstore.ProgressEvent{ID: "teacher-response:resp_1", SourceType: "teacher-response", SourceID: "resp_1", ActivityDelta: 1, Title: "完成教师对话"}

	if _, added, err := server.recordLearningActivity("activity", event); err != nil || !added {
		t.Fatalf("record learning activity: added=%v err=%v", added, err)
	}
	got := <-channel
	if got.Type != "learning-activity-updated" {
		t.Fatalf("event type = %q", got.Type)
	}
	if _, added, err := server.recordLearningActivity("activity", event); err != nil || added {
		t.Fatalf("duplicate activity: added=%v err=%v", added, err)
	}
	select {
	case duplicate := <-channel:
		t.Fatalf("duplicate activity broadcast: %#v", duplicate)
	default:
	}
}
