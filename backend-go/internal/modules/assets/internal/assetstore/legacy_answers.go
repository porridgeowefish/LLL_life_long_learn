package assetstore

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

type LegacyAnswer struct {
	Answer      json.RawMessage `json:"answer"`
	Explanation string          `json:"explanation,omitempty"`
}
type legacyTasks struct {
	SetID string           `json:"setId"`
	Tasks []map[string]any `json:"tasks"`
}

var legacyJSONFence = regexp.MustCompile("(?s)```(?:json)?[ \\t]*\\r?\\n(.*?)\\r?\\n```")

// A historical answer file is useful only for the exact retained question set.
// Never attach a newly edited/different set's answers to an older version.
func (s *Store) legacyAnswers(content string) map[string]LegacyAnswer {
	if rejectLinks(s.root, "practice") != nil {
		return nil
	}
	var original legacyTasks
	if readJSON(filepath.Join(s.root, "practice", "tasks.json"), &original) != nil || original.SetID == "" || len(original.Tasks) == 0 {
		return nil
	}
	expected, _ := json.Marshal(original.Tasks)
	candidates := []string{strings.TrimSpace(content)}
	for _, match := range legacyJSONFence.FindAllStringSubmatch(content, -1) {
		candidates = append(candidates, match[1])
	}
	matched := false
	for _, candidate := range candidates {
		var snapshot legacyTasks
		if json.Unmarshal([]byte(candidate), &snapshot) != nil || snapshot.SetID != original.SetID {
			continue
		}
		raw, _ := json.Marshal(snapshot.Tasks)
		if bytes.Equal(raw, expected) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}
	var key struct {
		SetID   string `json:"setId"`
		Answers []struct {
			TaskID        string          `json:"taskId"`
			CorrectAnswer json.RawMessage `json:"correctAnswer"`
			Explanation   string          `json:"explanation"`
		} `json:"answers"`
	}
	if readJSON(filepath.Join(s.root, "practice", "answer-key.json"), &key) != nil || key.SetID != original.SetID {
		return nil
	}
	ids := map[string]bool{}
	for _, task := range original.Tasks {
		if id, ok := task["id"].(string); ok {
			ids[id] = true
		}
	}
	answers := map[string]LegacyAnswer{}
	for _, entry := range key.Answers {
		if ids[entry.TaskID] && len(entry.CorrectAnswer) > 0 {
			answers[entry.TaskID] = LegacyAnswer{Answer: entry.CorrectAnswer, Explanation: entry.Explanation}
		}
	}
	return answers
}
