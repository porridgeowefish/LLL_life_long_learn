package conversationstore

import (
	"encoding/json"
	"sort"
	"strings"
)

type NoteTurn struct {
	LearnerMessageID string `json:"learnerMessageId"`
	ResponseID       string `json:"responseId"`
	ProviderID       string `json:"providerId"`
	Model            string `json:"model"`
	CompletedSeq     uint64 `json:"completedSeq"`
	LearnerText      string `json:"learnerText"`
	TeacherText      string `json:"teacherText"`
}

func (s *Store) SuccessfulNoteTurns(afterSeq uint64) ([]NoteTurn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, events, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	messages := map[string]Message{}
	completed := map[string]struct {
		Seq               uint64
		ProviderID, Model string
	}{}
	for _, e := range events {
		switch e.Type {
		case "message-recorded":
			var m Message
			if err := json.Unmarshal(e.Data, &m); err != nil {
				return nil, err
			}
			messages[m.ID] = m
		case "teacher-response-finished":
			var r struct{ ResponseID, Status, ProviderID, Model string }
			if err := json.Unmarshal(e.Data, &r); err != nil {
				return nil, err
			}
			if r.Status == "completed" {
				completed[r.ResponseID] = struct {
					Seq               uint64
					ProviderID, Model string
				}{e.Seq, r.ProviderID, r.Model}
			}
		}
	}
	text := func(m Message) string {
		var out []string
		for _, b := range m.Blocks {
			if b.Type == "markdown" {
				out = append(out, b.Source)
			}
		}
		return strings.Join(out, "\n")
	}
	var turns []NoteTurn
	state := replayResponses(events)
	for id, info := range state.byID {
		finish, ok := completed[id]
		if !ok || finish.Seq <= afterSeq || info.Active || info.Superseded || info.Message == nil || info.Message.Status != "completed" {
			continue
		}
		learner, ok := messages[info.TriggeringLearnerMessageID]
		if !ok || learner.Role != "learner" || learner.Status != "completed" {
			continue
		}
		turns = append(turns, NoteTurn{learner.ID, id, finish.ProviderID, finish.Model, finish.Seq, text(learner), text(*info.Message)})
	}
	sort.Slice(turns, func(i, j int) bool { return turns[i].CompletedSeq < turns[j].CompletedSeq })
	return turns, nil
}
