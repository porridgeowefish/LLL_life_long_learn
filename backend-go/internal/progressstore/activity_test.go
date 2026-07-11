package progressstore

import (
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func TestAggregateCombinesActivityGrowthAndLegacyPractice(t *testing.T) {
	dir := t.TempDir()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(dir)
	defer workspace.SetProjectsRootForTest(old)
	if err := workspace.CreateProjectSkeleton("math", "数学", ""); err != nil {
		t.Fatal(err)
	}
	store, err := New("math")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{
		{ID: "read", SourceType: "reading", SourceID: "p1", ActivityDelta: 2, Delta: 0, CreatedAt: "2026-07-10T10:00:00+08:00"},
		{ID: "submit", SourceType: "practice-submit", SourceID: "q1", Delta: 3, CreatedAt: "2026-07-11T10:00:00+08:00"},
		{ID: "correct", SourceType: "practice-correct", SourceID: "q1", Delta: 3, CreatedAt: "2026-07-11T10:01:00+08:00"},
	} {
		if _, _, _, err := store.Award(event); err != nil {
			t.Fatal(err)
		}
	}
	loc := time.FixedZone("CST", 8*60*60)
	summary, err := Aggregate("", 26, time.Date(2026, 7, 11, 18, 0, 0, 0, loc))
	if err != nil {
		t.Fatal(err)
	}
	if summary.ActiveDays != 2 || summary.CurrentStreak != 2 || summary.LongestStreak != 2 {
		t.Fatalf("unexpected streak summary: %+v", summary)
	}
	if summary.TotalActions != 2 || summary.TotalGrowth != 6 {
		t.Fatalf("unexpected totals: actions=%d growth=%d", summary.TotalActions, summary.TotalGrowth)
	}
	last := summary.Days[len(summary.Days)-1]
	if last.Activity != 1 || last.Growth != 6 || len(last.Events) != 2 {
		t.Fatalf("unexpected final day: %+v", last)
	}
}

func TestAggregateRejectsUnsupportedRange(t *testing.T) {
	if _, err := Aggregate("", 12, time.Now()); err == nil {
		t.Fatal("expected invalid weeks error")
	}
}
