// Package runprogress holds in-memory, per-run progress reported by Claude Code
// hooks. It is deliberately separate from the gamification progressstore.
package runprogress

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Status is the latest reported state of one run.
type Status struct {
	Phase        string `json:"phase,omitempty"`
	Activity     string `json:"activity,omitempty"`
	PagesDone    int    `json:"pagesDone,omitempty"`
	PagesPlanned int    `json:"pagesPlanned,omitempty"`
	Done         bool   `json:"done,omitempty"`
	Failed       bool   `json:"failed,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// Store maps runId -> *Status with per-run auth tokens.
type Store struct {
	mu     sync.Mutex
	m      map[string]*Status
	tokens map[string]string
}

// New returns an empty store.
func New() *Store {
	return &Store{m: map[string]*Status{}, tokens: map[string]string{}}
}

func newToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Register creates an entry for runId and returns the auth token the launcher
// injects into the run's hooks. Re-registering a runId issues a fresh token
// and revokes the previous one.
func (s *Store) Register(runId string) string {
	tok := newToken()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[runId]; !ok {
		s.m[runId] = &Status{}
	}
	s.tokens[runId] = tok
	return tok
}

// Set validates the token and merges non-zero fields of in into the run's
// status. Returns the merged status and ok=false if the token is wrong or the
// run is unknown.
func (s *Store) Set(runId, token string, in Status) (Status, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	want, ok := s.tokens[runId]
	if !ok || want == "" || want != token {
		return Status{}, false
	}
	cur := s.m[runId]
	if in.Phase != "" {
		cur.Phase = in.Phase
	}
	if in.Activity != "" {
		cur.Activity = in.Activity
	}
	if in.PagesDone != 0 {
		cur.PagesDone = in.PagesDone
	}
	if in.PagesPlanned != 0 {
		cur.PagesPlanned = in.PagesPlanned
	}
	if in.Done {
		cur.Done = true
	}
	if in.Failed {
		cur.Failed = true
	}
	cur.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return *cur, true
}

// Get returns a copy of the status for runId.
func (s *Store) Get(runId string) (Status, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[runId]
	if !ok {
		return Status{}, false
	}
	return *st, true
}
