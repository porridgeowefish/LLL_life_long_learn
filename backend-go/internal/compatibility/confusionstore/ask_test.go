// backend-go/internal/confusionstore/ask_test.go
package confusionstore

import (
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func useTempProjectsRoot(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	t.Cleanup(func() { workspace.SetProjectsRootForTest(old) })
}

func TestAppendAskMessageAndSummary(t *testing.T) {
	useTempProjectsRoot(t)
	s, err := New("proj")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Create(Confusion{QuoteSnapshot: "q"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendAskMessage(c.ID, AskMessage{Role: "user", Content: "why?"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendAskMessage(c.ID, AskMessage{Role: "assistant", Content: "because"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ask == nil || len(got.Ask.Messages) != 2 || got.Ask.Messages[1].Content != "because" {
		t.Fatalf("messages not stored: %+v", got.Ask)
	}

	updated, err := s.SetAskSummary(c.ID, "a short summary", "done")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Ask.Summary != "a short summary" || updated.Ask.SummaryState != "done" {
		t.Errorf("summary not stored: %+v", updated.Ask)
	}
	if updated.State != StateAsked {
		t.Errorf("state should be asked after done summary, got %s", updated.State)
	}

	// Reload to confirm persistence.
	s2, _ := New("proj")
	again, _ := s2.Get(c.ID)
	if again.Ask == nil || again.Ask.Summary != "a short summary" {
		t.Errorf("not persisted: %+v", again.Ask)
	}
}
