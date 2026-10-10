package conversationstore

import (
	"encoding/json"
	"errors"
	"github.com/xmz14/lll/backend-go/internal/idgen"
	sources "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"time"
)

type ReadingCoverage struct {
	SourceID            string                `json:"sourceId"`
	RevisionID          string                `json:"revisionId"`
	ContentHash         string                `json:"contentHash"`
	CoveredRanges       []sources.SourceRange `json:"coveredRanges"`
	NextRange           *sources.SourceRange  `json:"nextRange,omitempty"`
	Scope               string                `json:"scope"`
	CompleteWithinScope bool                  `json:"completeWithinScope"`
	UnresolvedTopics    []string              `json:"unresolvedTopics,omitempty"`
}

// The completed message owns the durable evidence. The additional event is a
// replayable audit projection; a crash between writes cannot lose its pointers.
func (s *Store) RecordMessageWithEvidence(responseID string, msg Message) (Event, error) {
	if responseID == "" || msg.ID == "" || (msg.Role != "teacher" && msg.Role != "learner" && msg.Role != "system") {
		return Event{}, errors.New("invalid message")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	if msg.CompletedAt.IsZero() && msg.Status != "streaming" {
		msg.CompletedAt = time.Now().UTC()
	}
	for i := range msg.Blocks {
		if msg.Blocks[i].ID == "" {
			msg.Blocks[i].ID = idgen.New("blk")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, events, err := s.loadLocked()
	if err != nil {
		return Event{}, err
	}
	for _, event := range events {
		if event.Type == "message-recorded" {
			var prior Message
			if json.Unmarshal(event.Data, &prior) == nil && prior.ID == msg.ID {
				return event, s.ensureSourceEvidenceLocked(responseID, prior, events)
			}
		}
	}
	event, err := s.appendLocked("message-recorded", msg)
	if err != nil {
		return event, err
	}
	err = s.ensureSourceEvidenceLocked(responseID, msg, events)
	return event, err
}

func (s *Store) ensureSourceEvidenceLocked(responseID string, msg Message, events []Event) error {
	if msg.Status != "completed" || (len(msg.SourceCitations) == 0 && len(msg.ReadingCoverage) == 0) {
		return nil
	}
	for _, event := range events {
		if event.Type == "source-evidence-recorded" {
			var data struct {
				ResponseID string `json:"responseId"`
			}
			if json.Unmarshal(event.Data, &data) == nil && data.ResponseID == responseID {
				return nil
			}
		}
	}
	_, err := s.appendLocked("source-evidence-recorded", map[string]any{"responseId": responseID, "messageId": msg.ID, "citations": msg.SourceCitations, "coverage": msg.ReadingCoverage})
	return err
}
