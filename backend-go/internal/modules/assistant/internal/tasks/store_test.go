package assistanttask

import (
	"errors"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func testStore(t *testing.T, slug string) *Store {
	t.Helper()
	if err := workspace.CreateProjectSkeletonWithInput(slug, slug, "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	s, err := New(slug)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestIdempotencyAndSameTypeExclusion(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	s := testStore(t, "topic")
	in := CreateInput{Type: "consolidate", Objective: "整理正文", Origin: Origin{OperationID: "op_1"}}
	first, created, err := s.Create(in)
	if err != nil || !created {
		t.Fatalf("create failed: %#v %v", first, err)
	}
	second, created, err := s.Create(in)
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("idempotency failed: %#v %v", second, err)
	}
	_, _, err = s.Create(CreateInput{Type: "consolidate", Objective: "再次整理", Origin: Origin{OperationID: "op_2"}})
	var active *SameTypeActiveError
	if !errors.As(err, &active) || active.ExistingTaskID != first.ID {
		t.Fatalf("expected same-type conflict, got %v", err)
	}
	_, err = s.Update(first.ID, func(task *Task) error { task.Status = "succeeded"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := s.Create(CreateInput{Type: "consolidate", Objective: "再次整理", Origin: Origin{OperationID: "op_2"}}); err != nil || !created {
		t.Fatalf("terminal task should release exclusion: %v", err)
	}
}

func TestOperationConflict(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	s := testStore(t, "topic")
	_, _, _ = s.Create(CreateInput{Type: "verify", Objective: "核查 A", Origin: Origin{OperationID: "op_same"}})
	_, _, err := s.Create(CreateInput{Type: "verify", Objective: "核查 B", Origin: Origin{OperationID: "op_same"}})
	var conflict *OperationConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected operation conflict, got %v", err)
	}
}

func TestConsolidationPersistsWhetherPracticeWasRequested(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	s := testStore(t, "topic")
	task, created, err := s.Create(CreateInput{Type: "consolidate", Objective: "沉淀本轮教学稿", PracticeRequested: true, Origin: Origin{OperationID: "op_consolidate"}})
	if err != nil || !created || !task.PracticeRequested {
		t.Fatalf("practice request was not persisted: %#v %v", task, err)
	}
}
