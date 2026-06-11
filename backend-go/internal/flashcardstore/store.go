// Package flashcardstore manages summary/flashcards.json and
// summary/flashcard-progress.json. File-backed (iter-03).
package flashcardstore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Flashcard is a single Q&A card.
type Flashcard struct {
	ID       string `json:"id"`
	Front    string `json:"front"`    // question / prompt (markdown)
	Back     string `json:"back"`     // answer / explanation (markdown)
	Zone     string `json:"zone"`     // source zone
	SourceID string `json:"sourceId"` // confusion / task / output reference
}

// CardProgress tracks learner's review history for one card.
type CardProgress struct {
	CardID     string `json:"cardId"`
	TimesSeen  int    `json:"timesSeen"`
	TimesRight int    `json:"timesRight"`
	LastGrade  string `json:"lastGrade"` // "forgot" | "fuzzy" | "got-it" | "easy"
}

// Store reads/writes flashcard files for a project.
type Store struct {
	projectRoot string
}

// New creates a flashcard store for the given project slug.
func New(slug string) (*Store, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	return &Store{projectRoot: root}, nil
}

func (s *Store) summaryDir() string {
	return filepath.Join(s.projectRoot, "summary")
}

// ReadFlashcards reads all flashcards. Returns empty slice if file missing.
func (s *Store) ReadFlashcards() ([]Flashcard, error) {
	raw, err := os.ReadFile(filepath.Join(s.summaryDir(), "flashcards.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cards []Flashcard
	if err := json.Unmarshal(raw, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// WriteFlashcards writes the flashcard deck.
func (s *Store) WriteFlashcards(cards []Flashcard) error {
	dir := s.summaryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, "flashcards.json"), raw, 0o644)
}

// ReadProgress reads review progress for all cards.
func (s *Store) ReadProgress() ([]CardProgress, error) {
	raw, err := os.ReadFile(filepath.Join(s.summaryDir(), "flashcard-progress.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var prog []CardProgress
	if err := json.Unmarshal(raw, &prog); err != nil {
		return nil, err
	}
	return prog, nil
}

// WriteProgress writes review progress.
func (s *Store) WriteProgress(prog []CardProgress) error {
	dir := s.summaryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(prog, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, "flashcard-progress.json"), raw, 0o644)
}

// GradeCard updates or creates progress for a single card.
func (s *Store) GradeCard(cardID, grade string) error {
	prog, err := s.ReadProgress()
	if err != nil {
		return err
	}
	found := false
	for i, p := range prog {
		if p.CardID == cardID {
			prog[i].TimesSeen++
			if grade == "got-it" || grade == "easy" {
				prog[i].TimesRight++
			}
			prog[i].LastGrade = grade
			found = true
			break
		}
	}
	if !found {
		p := CardProgress{
			CardID:    cardID,
			TimesSeen: 1,
			LastGrade: grade,
		}
		if grade == "got-it" || grade == "easy" {
			p.TimesRight = 1
		}
		prog = append(prog, p)
	}
	return s.WriteProgress(prog)
}
