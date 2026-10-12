package assetstore

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// PracticeResult is the learner's scored attempt for one practice round
// (one immutable practice version). Grading is deterministic and happens in
// the learner UI from the round's own answer key; the store keeps the
// submitted answers and score until the learner explicitly retakes the test.
type PracticeResult struct {
	SchemaVersion int                         `json:"schemaVersion"`
	VersionID     string                      `json:"versionId"`
	Answers       map[string][]string         `json:"answers"`
	Items         map[string]PracticeItemMark `json:"items"`
	Score         int                         `json:"score"`
	Total         int                         `json:"total"`
	SubmittedAt   time.Time                   `json:"submittedAt"`
}

// PracticeItemMark records the verdict and the key used for one question.
type PracticeItemMark struct {
	Correct  bool     `json:"correct"`
	Expected []string `json:"expected"`
}

var (
	ErrInvalidPracticeResult = errors.New("invalid practice result")
	ErrPracticeRoundNotFound = errors.New("practice round not found")
	practiceQuestionKey      = regexp.MustCompile(`^[1-9][0-9]{0,2}$`)
	practiceChoice           = regexp.MustCompile(`^(?:[A-Z]|true|false)$`)
)

const maxPracticeQuestions = 200

func validPracticeVersionID(id string) bool {
	return id != "" && id != "." && id != ".." && filepath.Base(id) == id && !strings.ContainsAny(id, "/\\")
}

func validChoices(values []string) bool {
	if len(values) > 26 {
		return false
	}
	for _, value := range values {
		if !practiceChoice.MatchString(value) {
			return false
		}
	}
	return true
}

func (r PracticeResult) validate() error {
	if r.Total < 1 || r.Total > maxPracticeQuestions || r.Score < 0 || r.Score > r.Total || len(r.Items) != r.Total || len(r.Answers) > maxPracticeQuestions {
		return ErrInvalidPracticeResult
	}
	correct := 0
	for key, item := range r.Items {
		if !practiceQuestionKey.MatchString(key) || len(item.Expected) == 0 || !validChoices(item.Expected) {
			return ErrInvalidPracticeResult
		}
		if item.Correct {
			correct++
		}
	}
	for key, answer := range r.Answers {
		if !practiceQuestionKey.MatchString(key) || !validChoices(answer) {
			return ErrInvalidPracticeResult
		}
	}
	if correct != r.Score {
		return ErrInvalidPracticeResult
	}
	return nil
}

func (s *Store) practiceResultsDir() string {
	return filepath.Join(s.root, "assets", "practice", "results")
}

// PracticeResults returns every saved round result keyed by practice version.
func (s *Store) PracticeResults() (map[string]PracticeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]PracticeResult{}
	entries, err := os.ReadDir(s.practiceResultsDir())
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var result PracticeResult
		if readJSON(filepath.Join(s.practiceResultsDir(), name), &result) != nil || result.validate() != nil {
			continue // A damaged result is ignored rather than blocking the round.
		}
		if strings.TrimSuffix(name, ".json") == result.VersionID {
			out[result.VersionID] = result
		}
	}
	return out, nil
}

// SavePracticeResult stores (or replaces) the scored attempt of an existing round.
func (s *Store) SavePracticeResult(versionID string, result PracticeResult) (PracticeResult, error) {
	if !validPracticeVersionID(versionID) {
		return PracticeResult{}, ErrPracticeRoundNotFound
	}
	result.SchemaVersion = 1
	result.VersionID = versionID
	if result.Answers == nil {
		result.Answers = map[string][]string{}
	}
	if err := result.validate(); err != nil {
		return PracticeResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(filepath.Join(s.root, "assets", "practice", "versions", versionID, "version.json")); err != nil {
		return PracticeResult{}, ErrPracticeRoundNotFound
	}
	if err := os.MkdirAll(s.practiceResultsDir(), 0o755); err != nil {
		return PracticeResult{}, err
	}
	result.SubmittedAt = time.Now().UTC()
	if err := writeJSON(filepath.Join(s.practiceResultsDir(), versionID+".json"), result); err != nil {
		return PracticeResult{}, err
	}
	return result, nil
}

// DeletePracticeResult clears a round's score when the learner retakes it.
func (s *Store) DeletePracticeResult(versionID string) error {
	if !validPracticeVersionID(versionID) {
		return ErrPracticeRoundNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(filepath.Join(s.practiceResultsDir(), versionID+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
