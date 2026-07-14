// Package sessionstore persists sessions + turns in memory and on disk.
package sessionstore

import (
	"sort"
	"sync"
	"time"
)

// SessionState enumerates the lifecycle states of a session.
type SessionState string

const (
	StatePreparing        SessionState = "preparing"
	StateLaunching        SessionState = "launching"
	StateRunning          SessionState = "running"
	StateCompleted        SessionState = "completed"
	StateCancelled        SessionState = "cancelled"
	StateFailed           SessionState = "failed"
	StateAwaitingFollowup SessionState = "awaiting-follow-up"
)

// Turn is one append-only conversational unit.
type Turn struct {
	ID        int       `json:"id"`
	Ordinal   int       `json:"ordinal"`
	Type      string    `json:"type"` // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	RunDirRel string    `json:"runDirRel,omitempty"` // empty for system turns
}

// Session captures one Claude terminal conversation.
type Session struct {
	ID          string       `json:"id"`
	ProjectSlug string       `json:"projectSlug"`
	ZoneName    string       `json:"zoneName"`
	AgentID     string       `json:"agentId"`
	State       SessionState `json:"state"`
	RunDirRel   string       `json:"runDirRel"`
	PromptPath  string       `json:"promptPath,omitempty"`
	Turns       []Turn       `json:"turns"`
	CreatedAt   time.Time    `json:"createdAt"`
	FinishedAt  *time.Time   `json:"finishedAt,omitempty"`
	ExitCode    *int         `json:"exitCode,omitempty"`
	LastMessage string       `json:"lastMessage,omitempty"`

	// Runtime-only fields (not serialized on disk):
	cancel chan struct{} `json:"-"`
}

// HasFollowups reports whether the session has had any follow-up turns
// after the initial user→assistant exchange. The initial exchange is
// turns [system, user, assistant]; any turn beyond that is a follow-up.
func (s *Session) HasFollowups() bool {
	return len(s.Turns) > 3
}

// LatestAssistantRunDir returns the run dir of the most recent assistant turn,
// or empty if none exists.
func (s *Session) LatestAssistantRunDir() string {
	for i := len(s.Turns) - 1; i >= 0; i-- {
		if s.Turns[i].Type == "assistant" && s.Turns[i].RunDirRel != "" {
			return s.Turns[i].RunDirRel
		}
	}
	return ""
}

// Store is the in-memory session registry. On-disk persistence goes to
// <project>/runs/_index/<sessionId>.json.
type Store struct {
	mu   sync.RWMutex
	byID map[string]*Session
}

// New returns an empty store.
func New() *Store {
	return &Store{byID: make(map[string]*Session)}
}

// Create adds a new session to the store and returns it.
func (s *Store) Create(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess.Turns == nil {
		sess.Turns = []Turn{}
	}
	s.byID[sess.ID] = sess
}

// Get returns the session by ID.
func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.byID[id]
	return sess, ok
}

// Update applies a mutation under the store's lock.
func (s *Store) Update(id string, fn func(*Session)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byID[id]
	if !ok {
		return false
	}
	fn(sess)
	return true
}

// List returns all sessions, optionally filtered by project slug.
func (s *Store) List(projectSlug string) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0)
	for _, sess := range s.byID {
		if projectSlug != "" && sess.ProjectSlug != projectSlug {
			continue
		}
		out = append(out, sess)
	}
	return out
}

// ListRecent returns the most recent N sessions sorted by CreatedAt descending.
// limit<=0 defaults to 10.
func (s *Store) ListRecent(limit int) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 10
	}
	all := make([]*Session, 0, len(s.byID))
	for _, sess := range s.byID {
		all = append(all, sess)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}

// ListActive returns sessions that are currently in flight.
func (s *Store) ListActive() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0)
	for _, sess := range s.byID {
		switch sess.State {
		case StatePreparing, StateLaunching, StateRunning, StateAwaitingFollowup:
			out = append(out, sess)
		}
	}
	return out
}

// HasActiveProject reports whether a project still has a run that may write
// artifacts. Project deletion uses this as a race-prevention boundary.
func (s *Store) HasActiveProject(projectSlug string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.byID {
		if sess.ProjectSlug != projectSlug {
			continue
		}
		switch sess.State {
		case StatePreparing, StateLaunching, StateRunning, StateAwaitingFollowup:
			return true
		}
	}
	return false
}

// RemoveProject removes completed session metadata after the canonical project
// directory (which contains the durable run records) has been deleted.
func (s *Store) RemoveProject(projectSlug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.byID {
		if sess.ProjectSlug == projectSlug {
			delete(s.byID, id)
		}
	}
}

// Stats returns aggregated counters for the dashboard.
// (total sessions, total turns across all sessions, in-flight sessions)
func (s *Store) Stats() (total int, turns int, active int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.byID {
		total++
		turns += len(sess.Turns)
		switch sess.State {
		case StatePreparing, StateLaunching, StateRunning, StateAwaitingFollowup:
			active++
		}
	}
	return
}

// AppendTurn appends a new turn with the next ordinal. Returns the new turn.
func (s *Store) AppendTurn(id string, turnType, content, runDirRel string) (Turn, bool) {
	var newTurn Turn
	ok := s.Update(id, func(sess *Session) {
		ordinal := len(sess.Turns) + 1
		newTurn = Turn{
			ID:        ordinal,
			Ordinal:   ordinal,
			Type:      turnType,
			Content:   content,
			CreatedAt: time.Now().UTC(),
			RunDirRel: runDirRel,
		}
		sess.Turns = append(sess.Turns, newTurn)
	})
	return newTurn, ok
}

// SetState updates a session's state.
func (s *Store) SetState(id string, state SessionState) bool {
	return s.Update(id, func(sess *Session) { sess.State = state })
}

// SetCancel registers a cancellation channel for a session.
func (s *Store) SetCancel(id string, ch chan struct{}) {
	s.Update(id, func(sess *Session) { sess.cancel = ch })
}

// Cancel signals the session's cancel channel (non-blocking).
func (s *Store) Cancel(id string) bool {
	s.mu.RLock()
	sess, ok := s.byID[id]
	s.mu.RUnlock()
	if !ok || sess.cancel == nil {
		return false
	}
	select {
	case <-sess.cancel:
		// already closed
	default:
		close(sess.cancel)
	}
	return true
}

// SetFinished marks a session as finished with state + exit code.
func (s *Store) SetFinished(id string, state SessionState, exitCode int) bool {
	now := time.Now().UTC()
	return s.Update(id, func(sess *Session) {
		sess.State = state
		sess.FinishedAt = &now
		sess.ExitCode = &exitCode
	})
}
