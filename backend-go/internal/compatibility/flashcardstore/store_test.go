package flashcardstore

import (
	"os"
	"path/filepath"
	"testing"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

func setupStore(t *testing.T) (*Store, func()) {
	t.Helper()
	old := workspace.ProjectsRootForTest()
	workspace.SetProjectsRootForTest(t.TempDir())
	if err := workspace.CreateProjectSkeleton("cards", "Cards", ""); err != nil {
		t.Fatal(err)
	}
	store, err := New("cards")
	if err != nil {
		t.Fatal(err)
	}
	return store, func() { workspace.SetProjectsRootForTest(old) }
}

func TestReadFlashcardsSupportsVersionedAndLegacyFiles(t *testing.T) {
	store, cleanup := setupStore(t)
	defer cleanup()

	versioned := []byte(`{"version":1,"cards":[{"id":"fc-1","front":"为什么？","back":"因为。","category":"concept","sourceRefs":["explain/output.md"]}]}`)
	path := filepath.Join(store.summaryDir(), "flashcards.json")
	if err := os.WriteFile(path, versioned, 0o644); err != nil {
		t.Fatal(err)
	}
	cards, err := store.ReadFlashcards()
	if err != nil || len(cards) != 1 || cards[0].Category != "concept" {
		t.Fatalf("versioned cards = %#v, err=%v", cards, err)
	}

	legacy := []byte(`[{"id":"old-1","front":"Q","back":"A","zone":"Explain","sourceId":"output"}]`)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	cards, err = store.ReadFlashcards()
	if err != nil || len(cards) != 1 || cards[0].Zone != "Explain" {
		t.Fatalf("legacy cards = %#v, err=%v", cards, err)
	}
}

func TestReadFlashcardsNormalizesCommonAgentVariants(t *testing.T) {
	store, cleanup := setupStore(t)
	defer cleanup()

	path := filepath.Join(store.summaryDir(), "flashcards.json")
	variants := []struct {
		name string
		raw  []byte
	}{
		{
			name: "fenced canonical",
			raw:  []byte("```json\n{\"version\":1,\"cards\":[{\"id\":\"fc-1\",\"front\":\"Q\",\"back\":\"A\"}]}\n```"),
		},
		{
			name: "flashcards question answer aliases",
			raw:  []byte(`{"schemaVersion":1,"flashcards":[{"id":"fc-2","question":"Q2","answer":"A2"}]}`),
		},
		{
			name: "items prompt response aliases",
			raw:  []byte(`{"items":[{"id":"fc-3","prompt":"Q3","response":"A3"}]}`),
		},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, tc.raw, 0o644); err != nil {
				t.Fatal(err)
			}
			cards, err := store.ReadFlashcards()
			if err != nil {
				t.Fatalf("ReadFlashcards: %v", err)
			}
			if len(cards) != 1 || cards[0].Front == "" || cards[0].Back == "" {
				t.Fatalf("cards not normalized: %#v", cards)
			}
		})
	}
}

func TestWriteFlashcardsRejectsInvalidCards(t *testing.T) {
	store, cleanup := setupStore(t)
	defer cleanup()

	err := store.WriteFlashcards([]Flashcard{
		{ID: "same", Front: "Q1", Back: "A1"},
		{ID: "same", Front: "Q2", Back: "A2"},
	})
	if err == nil {
		t.Fatal("expected duplicate id validation error")
	}
}
