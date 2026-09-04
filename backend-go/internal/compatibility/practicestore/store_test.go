package practicestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
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

func TestDraftRoundTrip(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	draft := &DraftFile{
		SetID:       "set-1",
		GeneratedAt: "2026-06-15T12:00:00Z",
		TaskIDs:     []string{"q1", "q2"},
		Drafts: map[string]DraftEntry{
			"q1": {Answer: json.RawMessage(`"learner answer"`), SelfAssess: 4},
		},
		Attempt: 3,
	}
	if err := store.WriteDraft(draft); err != nil {
		t.Fatal(err)
	}
	restored, err := store.ReadDraft()
	if err != nil {
		t.Fatal(err)
	}
	if restored == nil {
		t.Fatal("draft should round-trip")
	}
	if restored.SchemaVersion != 1 || restored.UpdatedAt == "" {
		t.Fatalf("draft metadata not populated: %#v", restored)
	}
	if got := string(restored.Drafts["q1"].Answer); got != `"learner answer"` {
		t.Fatalf("restored answer = %s", got)
	}
	if restored.Drafts["q1"].SelfAssess != 4 || restored.Attempt != 3 {
		t.Fatalf("restored draft fields = %#v", restored)
	}
}

func TestLatestAttemptAndEvaluationFeedbackArtifacts(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	first, err := store.CreateAttempt("set-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateAttempt("set-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SubmitAttempt(second.Attempt, []Submission{{
		TaskID: "q4", Answer: json.RawMessage(`"learner answer"`), SelfAssess: 3,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAttempt("set-1"); err != nil {
		t.Fatal(err)
	}
	latest, err := store.ReadLatestSubmittedAttempt()
	if err != nil {
		t.Fatal(err)
	}
	if latest.Attempt != second.Attempt || latest.Status != "submitted" {
		t.Fatalf("latest attempt = %#v, first=%d second=%d", latest, first.Attempt, second.Attempt)
	}

	ev := &Evaluation{
		Attempt: second.Attempt,
		Summary: "概念基础稳定，但迁移时遗漏边界条件。",
		Results: []EvaluationResult{{
			TaskID:          "q4",
			Score:           3,
			Feedback:        "方向正确，需要补充状态转换条件。",
			SuggestedAnswer: "先定义 token，再给出状态转换与接受状态。",
			Evidence:        "提交中未说明接受状态。",
			Passed:          true,
		}},
		OverallScore: 3,
		GeneratedAt:  NowISO(),
	}
	if err := store.WriteEvaluation(ev); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.practiceDir(), "evaluations", "2.md"))
	if err != nil {
		t.Fatal(err)
	}
	md := string(raw)
	for _, fragment := range []string{"总体反馈", ev.Summary, "参考回答", ev.Results[0].SuggestedAnswer} {
		if !strings.Contains(md, fragment) {
			t.Errorf("evaluation markdown missing %q:\n%s", fragment, md)
		}
	}
}

func TestLatestSubmittedAttemptSkipsEmptySubmission(t *testing.T) {
	store, cleanup := setupPracticeStore(t)
	defer cleanup()

	meaningful, err := store.CreateAttempt("set-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SubmitAttempt(meaningful.Attempt, []Submission{{
		TaskID: "q1", Answer: json.RawMessage(`"a real answer"`), SelfAssess: 3,
	}}); err != nil {
		t.Fatal(err)
	}
	empty, err := store.CreateAttempt("set-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SubmitAttempt(empty.Attempt, []Submission{{
		TaskID: "q1", Answer: json.RawMessage(`""`), SelfAssess: 0,
	}}); err != nil {
		t.Fatal(err)
	}

	latest, err := store.ReadLatestSubmittedAttempt()
	if err != nil {
		t.Fatal(err)
	}
	if latest.Attempt != meaningful.Attempt {
		t.Fatalf("latest meaningful attempt = %d, want %d", latest.Attempt, meaningful.Attempt)
	}
}
