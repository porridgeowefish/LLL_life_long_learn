package sessionstore

import (
	"testing"
)

func TestAppendTurn_AppendOnlyInvariant(t *testing.T) {
	store := New()
	sess := &Session{
		ID:          "s1",
		ProjectSlug: "p1",
		ZoneName:    "Explain",
		AgentID:     "explain",
		State:       StatePreparing,
	}
	store.Create(sess)

	// Append 4 turns: system, user, assistant, user follow-up.
	store.AppendTurn("s1", "system", "session created", "")
	t1, _ := store.AppendTurn("s1", "user", "Explain X", "runs/t1-explain")
	t2, _ := store.AppendTurn("s1", "assistant", "X is...", "runs/t1-explain")
	t3, _ := store.AppendTurn("s1", "user", "Compare with Y", "runs/t3-explain-followup")

	got, ok := store.Get("s1")
	if !ok {
		t.Fatal("session missing")
	}
	if len(got.Turns) != 4 {
		t.Fatalf("turn count = %d, want 4", len(got.Turns))
	}
	if t1.Ordinal != 2 || t2.Ordinal != 3 || t3.Ordinal != 4 {
		t.Errorf("ordinal mismatch: %+v", got.Turns)
	}
	// Prior turns must still be present (not replaced).
	if got.Turns[0].Content != "session created" {
		t.Errorf("turn 0 should still be system message, got %q", got.Turns[0].Content)
	}
	if got.Turns[2].Content != "X is..." {
		t.Errorf("turn 3 assistant content lost: %q", got.Turns[2].Content)
	}
}

func TestProjectSessionLifecycleHelpers(t *testing.T) {
	store := New()
	store.Create(&Session{ID: "active", ProjectSlug: "p1", State: StateRunning})
	store.Create(&Session{ID: "done", ProjectSlug: "p1", State: StateCompleted})
	store.Create(&Session{ID: "other", ProjectSlug: "p2", State: StateCompleted})
	if !store.HasActiveProject("p1") {
		t.Fatal("active project session was not detected")
	}
	store.SetState("active", StateCompleted)
	if store.HasActiveProject("p1") {
		t.Fatal("completed project reported as active")
	}
	store.RemoveProject("p1")
	if len(store.List("p1")) != 0 || len(store.List("p2")) != 1 {
		t.Fatalf("project session cleanup failed: p1=%d p2=%d", len(store.List("p1")), len(store.List("p2")))
	}
}

func TestHasFollowups(t *testing.T) {
	store := New()
	sess := &Session{ID: "s1"}
	store.Create(sess)
	if sess.HasFollowups() {
		t.Error("empty session should not have followups")
	}
	store.AppendTurn("s1", "system", "created", "")
	store.AppendTurn("s1", "user", "q1", "r1")
	store.AppendTurn("s1", "assistant", "a1", "r1")
	if sess.HasFollowups() {
		t.Error("session with only initial exchange should not report followups")
	}
	store.AppendTurn("s1", "user", "followup", "r2")
	if !sess.HasFollowups() {
		t.Error("session with follow-up should report HasFollowups=true")
	}
}

func TestSetState_Finished(t *testing.T) {
	store := New()
	store.Create(&Session{ID: "s1"})
	store.SetFinished("s1", StateCompleted, 0)
	got, _ := store.Get("s1")
	if got.State != StateCompleted {
		t.Errorf("state = %s, want completed", got.State)
	}
	if got.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("ExitCode should be 0, got %+v", got.ExitCode)
	}
}

func TestLatestAssistantRunDir(t *testing.T) {
	store := New()
	store.Create(&Session{ID: "s1"})
	store.AppendTurn("s1", "system", "x", "")
	store.AppendTurn("s1", "user", "q", "runs/r1")
	store.AppendTurn("s1", "assistant", "a", "runs/r1")
	store.AppendTurn("s1", "user", "followup", "runs/r2")
	// Latest assistant is still r1 (r2 is a user turn that hasn't been answered).
	sess, _ := store.Get("s1")
	if got := sess.LatestAssistantRunDir(); got != "runs/r1" {
		t.Errorf("LatestAssistantRunDir = %q, want runs/r1", got)
	}
}
