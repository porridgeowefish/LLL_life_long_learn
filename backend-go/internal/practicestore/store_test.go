package practicestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func setupPracticeStore(t *testing.T) (*Store, func()) {
	t.Helper()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(t.TempDir())
	if err := workspace.CreateProjectSkeleton("practice-test", "Practice Test", ""); err != nil {
		t.Fatal(err)
	}
	store, err := New("practice-test")
	if err != nil {
		t.Fatal(err)
	}
	return store, func() { workspace.SetProjectsRootForTest(old) }
}

func TestReadTasksLegacyDefaults(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	raw := []byte(`{"tasks":[{"id":"q1","type":"essay","question":"why"}],"generatedAt":"2026-06-14T00:00:00Z"}`)
	if err := os.WriteFile(filepath.Join(store.practiceDir(), "tasks.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	tasks, err := store.ReadTasks()
	if err != nil {
		t.Fatal(err)
	}
	if tasks.Tasks[0].Difficulty != 3 {
		t.Fatalf("legacy difficulty = %d, want 3", tasks.Tasks[0].Difficulty)
	}
	if tasks.SetID == "" {
		t.Fatal("legacy set id should be synthesized")
	}
}

func TestCheckObjectiveLocksAndComparesMultipleChoice(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	tasks := &TasksFile{
		SchemaVersion: 2,
		SetID:         "set-1",
		GeneratedAt:   NowISO(),
		Tasks: []Task{{
			ID: "q1", Type: TypeMultipleChoice, Question: "pick", Difficulty: 2,
		}},
	}
	if err := store.WriteTasks(tasks); err != nil {
		t.Fatal(err)
	}
	key := AnswerKeyFile{
		SchemaVersion: 1,
		SetID:         "set-1",
		Answers: []AnswerKeyEntry{{
			TaskID: "q1", CorrectAnswer: json.RawMessage(`["A","C"]`), Explanation: "because",
		}},
	}
	raw, _ := json.Marshal(key)
	if err := os.WriteFile(filepath.Join(store.practiceDir(), "answer-key.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	attempt, err := store.CreateAttempt("set-1")
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.CheckObjective(attempt.Attempt, "q1", json.RawMessage(`["C","A"]`))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Correct {
		t.Fatal("same multiple-choice set in different order should be correct")
	}
	locked, err := store.CheckObjective(attempt.Attempt, "q1", json.RawMessage(`["B"]`))
	if err != nil {
		t.Fatal(err)
	}
	if !locked.Correct {
		t.Fatal("second check must return the locked first result")
	}
}

func TestCreateAttemptAllocatesUniqueNumbersConcurrently(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	const count = 12
	ids := make(chan int, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attempt, err := store.CreateAttempt("set-1")
			if err != nil {
				errs <- err
				return
			}
			ids <- attempt.Attempt
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate attempt number %d", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Fatalf("created %d attempts, want %d", len(seen), count)
	}
}
