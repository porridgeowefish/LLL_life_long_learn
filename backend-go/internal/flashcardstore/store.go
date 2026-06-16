// Package flashcardstore manages summary/flashcards.json and
// summary/flashcard-progress.json. File-backed (iter-03).
package flashcardstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Flashcard is a single Q&A card.
type Flashcard struct {
	ID              string   `json:"id"`
	Front           string   `json:"front"`
	Back            string   `json:"back"`
	Category        string   `json:"category,omitempty"`
	SourceRefs      []string `json:"sourceRefs,omitempty"`
	GeneratedReason string   `json:"generatedReason,omitempty"`
	Zone            string   `json:"zone,omitempty"`     // legacy
	SourceID        string   `json:"sourceId,omitempty"` // legacy
}

type FlashcardsFile struct {
	Version int         `json:"version"`
	Cards   []Flashcard `json:"cards"`
}

type rawFlashcard struct {
	ID              string   `json:"id"`
	Front           string   `json:"front"`
	Back            string   `json:"back"`
	Question        string   `json:"question"`
	Answer          string   `json:"answer"`
	Prompt          string   `json:"prompt"`
	Response        string   `json:"response"`
	Category        string   `json:"category,omitempty"`
	SourceRefs      []string `json:"sourceRefs,omitempty"`
	GeneratedReason string   `json:"generatedReason,omitempty"`
	Zone            string   `json:"zone,omitempty"`
	SourceID        string   `json:"sourceId,omitempty"`
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
	cards, err := ParseFlashcards(raw)
	if err != nil {
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
	if err := validateFlashcards(cards); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(FlashcardsFile{Version: 1, Cards: cards}, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, "flashcards.json"), raw, 0o644)
}

// ParseFlashcards accepts the canonical versioned envelope plus common legacy
// and agent-output variants, then normalizes them into the canonical card
// shape used by the frontend.
func ParseFlashcards(raw []byte) ([]Flashcard, error) {
	clean := []byte(stripJSONFence(strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))))
	var file struct {
		Version       int            `json:"version"`
		SchemaVersion int            `json:"schemaVersion"`
		Cards         []rawFlashcard `json:"cards"`
		Flashcards    []rawFlashcard `json:"flashcards"`
		Items         []rawFlashcard `json:"items"`
	}
	if err := json.Unmarshal(clean, &file); err == nil {
		switch {
		case file.Cards != nil:
			return normalizeRawCards(file.Cards)
		case file.Flashcards != nil:
			return normalizeRawCards(file.Flashcards)
		case file.Items != nil:
			return normalizeRawCards(file.Items)
		}
	}
	var legacy []rawFlashcard
	if err := json.Unmarshal(clean, &legacy); err != nil {
		return nil, fmt.Errorf("parse flashcards.json: %w", err)
	}
	return normalizeRawCards(legacy)
}

func stripJSONFence(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return text
	}
	start := 1
	end := len(lines)
	if strings.HasPrefix(strings.TrimSpace(lines[end-1]), "```") {
		end--
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func normalizeRawCards(raw []rawFlashcard) ([]Flashcard, error) {
	cards := make([]Flashcard, 0, len(raw))
	for _, item := range raw {
		front := firstNonEmpty(item.Front, item.Question, item.Prompt)
		back := firstNonEmpty(item.Back, item.Answer, item.Response)
		cards = append(cards, Flashcard{
			ID:              item.ID,
			Front:           front,
			Back:            back,
			Category:        item.Category,
			SourceRefs:      item.SourceRefs,
			GeneratedReason: item.GeneratedReason,
			Zone:            item.Zone,
			SourceID:        item.SourceID,
		})
	}
	if err := validateFlashcards(cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateFlashcards(cards []Flashcard) error {
	seen := map[string]bool{}
	for i, card := range cards {
		if strings.TrimSpace(card.ID) == "" {
			return fmt.Errorf("flashcard %d missing id", i)
		}
		if seen[card.ID] {
			return fmt.Errorf("duplicate flashcard id: %s", card.ID)
		}
		seen[card.ID] = true
		if strings.TrimSpace(card.Front) == "" || strings.TrimSpace(card.Back) == "" {
			return fmt.Errorf("flashcard %s requires front and back", card.ID)
		}
	}
	return nil
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
