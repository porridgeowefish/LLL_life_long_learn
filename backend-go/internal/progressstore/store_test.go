package progressstore

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestAwardIsIdempotent(t *testing.T) {
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(t.TempDir())
	defer workspace.SetProjectsRootForTest(old)
	if err := workspace.CreateProjectSkeleton("progress-test", "Progress Test", ""); err != nil {
		t.Fatal(err)
	}
	store, err := New("progress-test")
	if err != nil {
		t.Fatal(err)
	}
	event := Event{
		ID: "practice-submit:1:q1", SourceType: "practice-submit", SourceID: "q1",
		AttemptID: "1", Difficulty: 3, Outcome: "submitted", Delta: 3,
	}
	_, added, summary, err := store.Award(event)
	if err != nil || !added || summary.Total != 3 {
		t.Fatalf("first award: added=%v total=%d err=%v", added, summary.Total, err)
	}
	_, added, summary, err = store.Award(event)
	if err != nil || added || summary.Total != 3 {
		t.Fatalf("duplicate award: added=%v total=%d err=%v", added, summary.Total, err)
	}
}

func TestAwardIsIdempotentUnderConcurrency(t *testing.T) {
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(t.TempDir())
	defer workspace.SetProjectsRootForTest(old)
	if err := workspace.CreateProjectSkeleton("progress-concurrent", "Progress Concurrent", ""); err != nil {
		t.Fatal(err)
	}
	store, err := New("progress-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	event := Event{
		ID: "practice-correct:1:q1", SourceType: "practice-correct", SourceID: "q1",
		AttemptID: "1", Difficulty: 4, Outcome: "correct", Delta: 4,
	}

	var addedCount int32
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, added, _, awardErr := store.Award(event)
			if awardErr != nil {
				errs <- awardErr
				return
			}
			if added {
				atomic.AddInt32(&addedCount, 1)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for awardErr := range errs {
		t.Fatal(awardErr)
	}
	summary, err := store.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if addedCount != 1 || summary.Total != 4 {
		t.Fatalf("added=%d total=%d, want added=1 total=4", addedCount, summary.Total)
	}
}
