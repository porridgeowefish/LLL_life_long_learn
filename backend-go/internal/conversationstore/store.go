package conversationstore

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const SchemaVersion = 1

type Block struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Source      string `json:"source,omitempty"`
	ArtifactRef string `json:"artifactRef,omitempty"`
}

type Message struct {
	ID          string    `json:"id"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	Blocks      []Block   `json:"blocks"`
	OperationID string    `json:"operationId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

type Event struct {
	SchemaVersion int             `json:"schemaVersion"`
	Seq           uint64          `json:"seq"`
	EventID       string          `json:"eventId"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Data          json.RawMessage `json:"data"`
}

type Meta struct {
	SchemaVersion       int       `json:"schemaVersion"`
	ID                  string    `json:"id"`
	UnitID              string    `json:"unitId"`
	LatestSeq           uint64    `json:"latestSeq"`
	LatestEventID       string    `json:"latestEventId,omitempty"`
	EventCount          uint64    `json:"eventCount"`
	CompactedThroughSeq uint64    `json:"compactedThroughSeq"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type Unit struct {
	SchemaVersion   int       `json:"schemaVersion"`
	UnitID          string    `json:"unitId"`
	ConversationID  string    `json:"conversationId"`
	ProjectSlug     string    `json:"projectSlug"`
	ActiveAssetKeys []string  `json:"activeAssetKeys"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Projection struct {
	ConversationID string     `json:"conversationId"`
	UnitID         string     `json:"unitId"`
	LatestSeq      uint64     `json:"latestSeq"`
	PageThroughSeq uint64     `json:"pageThroughSeq"`
	HasMore        bool       `json:"hasMore"`
	Messages       []Message  `json:"messages"`
	TaskLinks      []TaskLink `json:"taskLinks"`
}

type TaskLink struct {
	MessageID  string `json:"messageId"`
	TaskID     string `json:"taskId"`
	ToolCallID string `json:"toolCallId,omitempty"`
}

type SequencedMessage struct {
	Seq     uint64
	EventID string
	Message Message
}

type ResponseRecord struct {
	ResponseID       string
	TeacherMessageID string
	Message          *Message
	Active           bool
}

type Store struct {
	slug string
	root string
	mu   *sync.Mutex
}

var locks sync.Map

func New(slug string) (*Store, error) {
	if !workspace.ValidateSlug(slug) {
		return nil, errors.New("invalid project slug")
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		return nil, err
	}
	if state.ProjectType != workspace.ProjectTypeSystemLearning {
		return nil, errors.New("teacher conversation requires system-learning project")
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	value, _ := locks.LoadOrStore(root, &sync.Mutex{})
	s := &Store{slug: slug, root: root, mu: value.(*sync.Mutex)}
	if err := s.ensure(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	convDir := filepath.Join(s.root, "conversation")
	if err := os.MkdirAll(convDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(s.root, "unit.json")
	metaPath := filepath.Join(convDir, "conversation.json")
	if _, err := os.Stat(unitPath); err == nil {
		if _, err := os.Stat(metaPath); err == nil {
			return nil
		}
	}
	now := time.Now().UTC()
	unit := Unit{SchemaVersion: SchemaVersion, UnitID: idgen.New("unit"), ConversationID: idgen.New("conv"), ProjectSlug: s.slug, ActiveAssetKeys: []string{"intro", "body", "practice"}, CreatedAt: now, UpdatedAt: now}
	if data, err := os.ReadFile(unitPath); err == nil {
		_ = json.Unmarshal(data, &unit)
	}
	meta := Meta{SchemaVersion: SchemaVersion, ID: unit.ConversationID, UnitID: unit.UnitID, CreatedAt: now, UpdatedAt: now}
	if err := writeJSON(unitPath, unit); err != nil {
		return err
	}
	if err := writeJSON(metaPath, meta); err != nil {
		return err
	}
	events := filepath.Join(convDir, "events.jsonl")
	f, err := os.OpenFile(events, os.O_CREATE, 0o644)
	if err == nil {
		err = f.Close()
	}
	return err
}

func (s *Store) Unit() (Unit, error) {
	var u Unit
	err := readJSON(filepath.Join(s.root, "unit.json"), &u)
	return u, err
}

func (s *Store) Meta() (Meta, error) {
	var m Meta
	err := readJSON(filepath.Join(s.root, "conversation", "conversation.json"), &m)
	return m, err
}

func (s *Store) Append(eventType string, data any) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendLocked(eventType, data)
}

func (s *Store) appendLocked(eventType string, data any) (Event, error) {
	meta, events, err := s.loadLocked()
	if err != nil {
		return Event{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, err
	}
	e := Event{SchemaVersion: SchemaVersion, Seq: meta.LatestSeq + 1, EventID: idgen.New("event"), Type: eventType, OccurredAt: time.Now().UTC(), Data: raw}
	line, err := json.Marshal(e)
	if err != nil {
		return Event{}, err
	}
	f, err := os.OpenFile(filepath.Join(s.root, "conversation", "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Event{}, err
	}
	if _, err = f.Write(append(line, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return Event{}, err
	}
	if closeErr != nil {
		return Event{}, closeErr
	}
	meta.LatestSeq = e.Seq
	meta.LatestEventID = e.EventID
	meta.EventCount = uint64(len(events) + 1)
	meta.UpdatedAt = e.OccurredAt
	if err := writeJSON(filepath.Join(s.root, "conversation", "conversation.json"), meta); err != nil {
		return Event{}, err
	}
	return e, nil
}

func (s *Store) AppendMessage(role, status, operationID string, blocks []Block) (Message, Event, error) {
	if role != "learner" && role != "teacher" && role != "system" {
		return Message{}, Event{}, errors.New("invalid message role")
	}
	for i := range blocks {
		if blocks[i].ID == "" {
			blocks[i].ID = idgen.New("blk")
		}
	}
	msg := Message{ID: idgen.New("msg"), Role: role, Status: status, Blocks: blocks, OperationID: operationID, CreatedAt: time.Now().UTC()}
	if status != "streaming" {
		msg.CompletedAt = msg.CreatedAt
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if operationID != "" {
		_, events, err := s.loadLocked()
		if err != nil {
			return Message{}, Event{}, err
		}
		for _, event := range events {
			if event.Type != "message-recorded" {
				continue
			}
			var existing Message
			if json.Unmarshal(event.Data, &existing) == nil && existing.OperationID == operationID {
				return existing, event, nil
			}
		}
	}
	e, err := s.appendLocked("message-recorded", msg)
	return msg, e, err
}

// RecordMessage persists a message whose identity was reserved before a
// streaming response started. It is also useful for importing durable history.
func (s *Store) RecordMessage(msg Message) (Event, error) {
	if msg.ID == "" || (msg.Role != "learner" && msg.Role != "teacher" && msg.Role != "system") {
		return Event{}, errors.New("invalid message")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	if msg.Status != "streaming" && msg.CompletedAt.IsZero() {
		msg.CompletedAt = time.Now().UTC()
	}
	for i := range msg.Blocks {
		if msg.Blocks[i].ID == "" {
			msg.Blocks[i].ID = idgen.New("blk")
		}
	}
	return s.Append("message-recorded", msg)
}

func (s *Store) Read(afterSeq uint64, limit int) (Projection, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, events, err := s.loadLocked()
	if err != nil {
		return Projection{}, err
	}
	proj := Projection{ConversationID: meta.ID, UnitID: meta.UnitID, LatestSeq: meta.LatestSeq, Messages: []Message{}, TaskLinks: []TaskLink{}}
	count := 0
	for _, event := range events {
		if event.Seq <= afterSeq {
			continue
		}
		if count >= limit {
			proj.HasMore = true
			break
		}
		proj.PageThroughSeq = event.Seq
		switch event.Type {
		case "message-recorded":
			var msg Message
			if json.Unmarshal(event.Data, &msg) == nil {
				proj.Messages = append(proj.Messages, msg)
			}
		case "task-linked":
			var link TaskLink
			if json.Unmarshal(event.Data, &link) == nil {
				proj.TaskLinks = append(proj.TaskLinks, link)
			}
		}
		count++
	}
	return proj, nil
}

// SnapshotThrough returns the complete durable projection up to and including
// throughSeq. It is used to seal assistant input: later conversation events
// must never leak into an already approved task.
func (s *Store) SnapshotThrough(throughSeq uint64) (Projection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, events, err := s.loadLocked()
	if err != nil {
		return Projection{}, err
	}
	if throughSeq > meta.LatestSeq {
		return Projection{}, errors.New("conversation cutoff exceeds latest sequence")
	}
	projection := Projection{ConversationID: meta.ID, UnitID: meta.UnitID, LatestSeq: throughSeq, PageThroughSeq: throughSeq, Messages: []Message{}, TaskLinks: []TaskLink{}}
	for _, event := range events {
		if event.Seq > throughSeq {
			break
		}
		switch event.Type {
		case "message-recorded":
			var message Message
			if json.Unmarshal(event.Data, &message) == nil {
				projection.Messages = append(projection.Messages, message)
			}
		case "task-linked":
			var link TaskLink
			if json.Unmarshal(event.Data, &link) == nil {
				projection.TaskLinks = append(projection.TaskLinks, link)
			}
		}
	}
	return projection, nil
}

func (s *Store) AllMessages() ([]Message, error) {
	sequenced, err := s.SequencedMessages()
	if err != nil {
		return nil, err
	}
	messages := make([]Message, 0, len(sequenced))
	for _, item := range sequenced {
		messages = append(messages, item.Message)
	}
	return messages, nil
}

func (s *Store) SequencedMessages() ([]SequencedMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, events, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	var messages []SequencedMessage
	for _, event := range events {
		if event.Type != "message-recorded" {
			continue
		}
		var message Message
		if json.Unmarshal(event.Data, &message) == nil {
			messages = append(messages, SequencedMessage{Seq: event.Seq, EventID: event.EventID, Message: message})
		}
	}
	return messages, nil
}

func (s *Store) FindMessage(id string) (Message, bool, error) {
	msgs, err := s.AllMessages()
	if err != nil {
		return Message{}, false, err
	}
	for _, msg := range msgs {
		if msg.ID == id {
			return msg, true, nil
		}
	}
	return Message{}, false, nil
}

// ResponseForLearner resolves the durable provider response associated with a
// learner message. It enables HTTP idempotency without replaying a provider
// request after a client reconnect.
func (s *Store) ResponseForLearner(learnerMessageID string) (ResponseRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, events, err := s.loadLocked()
	if err != nil {
		return ResponseRecord{}, false, err
	}
	var record ResponseRecord
	for _, event := range events {
		switch event.Type {
		case "teacher-response-started":
			var started struct {
				ResponseID                 string `json:"responseId"`
				TeacherMessageID           string `json:"teacherMessageId"`
				TriggeringLearnerMessageID string `json:"triggeringLearnerMessageId"`
			}
			if json.Unmarshal(event.Data, &started) == nil && started.TriggeringLearnerMessageID == learnerMessageID {
				record.ResponseID, record.TeacherMessageID, record.Active = started.ResponseID, started.TeacherMessageID, true
			}
		case "message-recorded":
			var message Message
			if json.Unmarshal(event.Data, &message) == nil && record.TeacherMessageID != "" && message.ID == record.TeacherMessageID {
				copy := message
				record.Message = &copy
			}
		case "teacher-response-finished":
			var finished struct {
				ResponseID string `json:"responseId"`
			}
			if json.Unmarshal(event.Data, &finished) == nil && record.ResponseID != "" && finished.ResponseID == record.ResponseID {
				record.Active = false
			}
		}
	}
	return record, record.ResponseID != "", nil
}

// ReconcileInterruptedResponses closes response-start records left active by a
// prior server process. Call it only during startup, before accepting turns;
// calling it while a provider is streaming would incorrectly interrupt work.
func (s *Store) ReconcileInterruptedResponses() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, events, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	type activeResponse struct {
		ResponseID       string
		TeacherMessageID string
		Message          *Message
		StartSeq         uint64
	}
	active := map[string]*activeResponse{}
	for _, event := range events {
		switch event.Type {
		case "teacher-response-started":
			var started struct {
				ResponseID       string `json:"responseId"`
				TeacherMessageID string `json:"teacherMessageId"`
			}
			if json.Unmarshal(event.Data, &started) == nil && started.ResponseID != "" {
				active[started.ResponseID] = &activeResponse{ResponseID: started.ResponseID, TeacherMessageID: started.TeacherMessageID, StartSeq: event.Seq}
			}
		case "message-recorded":
			var message Message
			if json.Unmarshal(event.Data, &message) == nil {
				for _, response := range active {
					if response.TeacherMessageID == message.ID {
						copy := message
						response.Message = &copy
					}
				}
			}
		case "teacher-response-finished":
			var finished struct {
				ResponseID string `json:"responseId"`
			}
			if json.Unmarshal(event.Data, &finished) == nil {
				delete(active, finished.ResponseID)
			}
		}
	}
	ordered := make([]*activeResponse, 0, len(active))
	for _, response := range active {
		ordered = append(ordered, response)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].StartSeq < ordered[j].StartSeq })
	for _, response := range ordered {
		status := "interrupted"
		if response.Message == nil {
			now := time.Now().UTC()
			message := Message{ID: response.TeacherMessageID, Role: "teacher", Status: status, Blocks: []Block{}, CreatedAt: now, CompletedAt: now}
			if _, err := s.appendLocked("message-recorded", message); err != nil {
				return 0, err
			}
		} else if response.Message.Status != "" {
			status = response.Message.Status
		}
		if _, err := s.appendLocked("teacher-response-finished", map[string]any{"responseId": response.ResponseID, "messageId": response.TeacherMessageID, "status": status, "recoveredAtStartup": true}); err != nil {
			return 0, err
		}
	}
	return len(ordered), nil
}

func ReconcileAllInterruptedResponses() {
	projects, err := workspace.IndexAll()
	if err != nil {
		return
	}
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		if store, err := New(project.Slug); err == nil {
			_, _ = store.ReconcileInterruptedResponses()
		}
	}
}

func (s *Store) loadLocked() (Meta, []Event, error) {
	var meta Meta
	if err := readJSON(filepath.Join(s.root, "conversation", "conversation.json"), &meta); err != nil {
		return Meta{}, nil, err
	}
	events, err := readEvents(filepath.Join(s.root, "conversation", "events.jsonl"))
	if err != nil {
		return Meta{}, nil, err
	}
	if len(events) > 0 {
		last := events[len(events)-1]
		if meta.LatestSeq != last.Seq || meta.EventCount != uint64(len(events)) {
			meta.LatestSeq = last.Seq
			meta.LatestEventID = last.EventID
			meta.EventCount = uint64(len(events))
			meta.UpdatedAt = last.OccurredAt
			if err := writeJSON(filepath.Join(s.root, "conversation", "conversation.json"), meta); err != nil {
				return Meta{}, nil, err
			}
		}
	}
	return meta, events, nil
}

func readEvents(path string) ([]Event, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var events []Event
	var seq uint64
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			var event Event
			if err := json.Unmarshal(line, &event); err != nil {
				if errors.Is(readErr, io.EOF) {
					break
				}
				return nil, fmt.Errorf("decode events.jsonl: %w", err)
			}
			if event.Seq != seq+1 {
				return nil, fmt.Errorf("conversation sequence gap at %d", event.Seq)
			}
			seq = event.Seq
			events = append(events, event)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	return events, nil
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(path, append(data, '\n'), 0o644)
}
