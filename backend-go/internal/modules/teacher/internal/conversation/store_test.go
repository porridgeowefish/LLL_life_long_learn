package conversationstore

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func TestAppendReadAndIdempotency(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("math", "数学", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	s, err := New("math")
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := s.AppendMessage("learner", "completed", "op-1", []Block{{Type: "markdown", Source: "你好"}})
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := s.AppendMessage("learner", "completed", "op-1", []Block{{Type: "markdown", Source: "重复"}})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != again.ID {
		t.Fatalf("idempotent append created another message: %s != %s", first.ID, again.ID)
	}
	proj, err := s.Read(0, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(proj.Messages) != 1 || proj.Messages[0].Blocks[0].Source != "你好" {
		t.Fatalf("unexpected projection: %#v", proj)
	}
}

func TestReadPagesExposeAdvancingSequence(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, err := New("topic")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, _, err := store.AppendMessage("learner", "completed", fmt.Sprintf("page-op-%d", i), []Block{{Type: "markdown", Source: "x"}}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.Read(0, 2)
	if err != nil || !first.HasMore || first.PageThroughSeq != 2 || len(first.Messages) != 2 {
		t.Fatalf("unexpected first page: %#v %v", first, err)
	}
	second, err := store.Read(first.PageThroughSeq, 2)
	if err != nil || second.HasMore || second.PageThroughSeq != 4 || len(second.Messages) != 2 {
		t.Fatalf("unexpected second page: %#v %v", second, err)
	}
}

func TestReadRecentPagesFromTailAndKeepsTaskLinks(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("recent", "最近对话", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, _ := New("recent")
	var messageIDs []string
	for i := 0; i < 5; i++ {
		message, _, err := store.AppendMessage("teacher", "completed", fmt.Sprintf("recent-op-%d", i), []Block{{Type: "markdown", Source: fmt.Sprintf("message-%d", i)}})
		if err != nil {
			t.Fatal(err)
		}
		messageIDs = append(messageIDs, message.ID)
		if i == 4 {
			if _, err := store.Append("task-linked", TaskLink{MessageID: message.ID, TaskID: "task-tail"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	latest, err := store.ReadRecent(0, 2)
	if err != nil || !latest.HasPrevious || latest.TotalMessages != 5 || len(latest.Messages) != 2 {
		t.Fatalf("unexpected latest page: %#v %v", latest, err)
	}
	if latest.Messages[0].ID != messageIDs[3] || latest.Messages[1].ID != messageIDs[4] || len(latest.TaskLinks) != 1 {
		t.Fatalf("tail page lost ordering or task link: %#v", latest)
	}
	older, err := store.ReadRecent(latest.PageFromSeq, 2)
	if err != nil || !older.HasPrevious || len(older.Messages) != 2 || older.Messages[0].ID != messageIDs[1] || older.Messages[1].ID != messageIDs[2] {
		t.Fatalf("unexpected older page: %#v %v", older, err)
	}
	oldest, err := store.ReadRecent(older.PageFromSeq, 2)
	if err != nil || oldest.HasPrevious || len(oldest.Messages) != 1 || oldest.Messages[0].ID != messageIDs[0] {
		t.Fatalf("unexpected oldest page: %#v %v", oldest, err)
	}
}

func TestSnapshotThroughExcludesLaterMessagesWithoutPageLimit(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("cutoff", "截止快照", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	store, err := New("cutoff")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 510; i++ {
		if _, _, err := store.AppendMessage("learner", "completed", "", []Block{{Type: "markdown", Source: fmt.Sprintf("message-%d", i)}}); err != nil {
			t.Fatal(err)
		}
	}
	sequenced, err := store.SequencedMessages()
	if err != nil {
		t.Fatal(err)
	}
	cutoff := sequenced[504].Seq
	snapshot, err := store.SnapshotThrough(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Messages) != 505 {
		t.Fatalf("snapshot unexpectedly truncated: %d", len(snapshot.Messages))
	}
	if got := snapshot.Messages[len(snapshot.Messages)-1].Blocks[0].Source; got != "message-504" {
		t.Fatalf("snapshot crossed cutoff: %q", got)
	}
}

func TestTruncatedFinalLineIsIgnored(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("physics", "物理", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	s, _ := New("physics")
	_, _, _ = s.AppendMessage("learner", "completed", "op-1", []Block{{Type: "markdown", Source: "a"}})
	path := filepath.Join(root, "physics", "conversation", "events.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"schemaVersion":1`)
	_ = f.Close()
	proj, err := s.Read(0, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(proj.Messages) != 1 {
		t.Fatalf("expected valid prefix, got %#v", proj.Messages)
	}
}

func TestStartupReconcileClosesOrphanedTeacherResponse(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("orphan", "中断", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	store, _ := New("orphan")
	learner, _, _ := store.AppendMessage("learner", "completed", "op_orphan", []Block{{Type: "markdown", Source: "问题"}})
	_, err := store.Append("teacher-response-started", map[string]any{"responseId": "resp_orphan", "teacherMessageId": "msg_orphan", "triggeringLearnerMessageId": learner.ID})
	if err != nil {
		t.Fatal(err)
	}
	count, err := store.ReconcileInterruptedResponses()
	if err != nil || count != 1 {
		t.Fatalf("reconcile failed: count=%d err=%v", count, err)
	}
	record, found, err := store.ResponseForLearner(learner.ID)
	if err != nil || !found || record.Active || record.Message == nil || record.Message.Status != "interrupted" {
		t.Fatalf("orphan response not closed: %#v %v", record, err)
	}
	if again, err := store.ReconcileInterruptedResponses(); err != nil || again != 0 {
		t.Fatalf("reconcile is not idempotent: count=%d err=%v", again, err)
	}
}
