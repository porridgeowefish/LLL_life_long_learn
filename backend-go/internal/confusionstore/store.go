// Package confusionstore manages explain/confusions.json — the learner's
// confusion markers and their lifecycle (open → asked → resolved/deleted).
// File-backed (iter-03 file-first persistence); no DB.
package confusionstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// State enumerates the confusion lifecycle.
type State string

const (
	StateOpen     State = "open"
	StateAsked    State = "asked"
	StateResolved State = "resolved"
	StateDeleted  State = "deleted"
)

// Confusion is one learner-marked confusion point.
type Confusion struct {
	ID               string `json:"id"`
	SourceArtifactID string `json:"sourceArtifactId,omitempty"`
	ParagraphID      string `json:"paragraphId,omitempty"`
	CharStart        int    `json:"charStart"`
	CharEnd          int    `json:"charEnd"`
	QuoteSnapshot    string `json:"quoteSnapshot"`
	Notes            string `json:"notes,omitempty"`
	State            State  `json:"state"`
	CreatedAt        string `json:"createdAt"`
	Ask              *Ask   `json:"ask,omitempty"`
}

// AskMessage is one turn of an Ask-AI exchange attached to a confusion.
type AskMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// Ask is the persisted Ask-AI exchange for a confusion.
type Ask struct {
	Messages     []AskMessage `json:"messages,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	SummaryState string       `json:"summaryState,omitempty"` // idle | pending | done | failed
	ProviderID   string       `json:"providerId,omitempty"`
	UpdatedAt    string       `json:"updatedAt,omitempty"`
}

// Store is a file-backed confusion store for one project.
type Store struct {
	mu       sync.Mutex
	slug     string
	filePath string
	data     []Confusion
	dirty    bool
}

// New creates a Store for the given project slug.
func New(slug string) (*Store, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	fp := filepath.Join(root, "explain", "confusions.json")
	s := &Store{slug: slug, filePath: fp}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = []Confusion{}
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		s.data = []Confusion{}
		return nil
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *Store) save() error {
	if !s.dirty {
		return nil
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := workspace.AtomicWriteFile(s.filePath, raw, 0o644); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

// List returns all confusions optionally filtered by state.
func (s *Store) List(stateFilter State) []Confusion {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stateFilter == "" {
		out := make([]Confusion, 0, len(s.data))
		for _, c := range s.data {
			if c.State != StateDeleted {
				out = append(out, c)
			}
		}
		return out
	}
	var out []Confusion
	for _, c := range s.data {
		if c.State == stateFilter {
			out = append(out, c)
		}
	}
	return out
}

// Create adds a new confusion and persists.
func (s *Store) Create(c Confusion) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = newID()
	}
	if c.State == "" {
		c.State = StateOpen
	}
	if c.CreatedAt == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	s.data = append(s.data, c)
	s.dirty = true
	return c, s.save()
}

// Update modifies a confusion by ID. Returns the updated confusion or error.
func (s *Store) Update(id string, patch map[string]any) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			if v, ok := patch["notes"]; ok {
				c.Notes, _ = v.(string)
			}
			if v, ok := patch["state"]; ok {
				sv, _ := v.(string)
				c.State = State(sv)
			}
			s.data[i] = c
			s.dirty = true
			return c, s.save()
		}
	}
	return Confusion{}, os.ErrNotExist
}

// Delete permanently removes a saved summary by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			s.data = append(s.data[:i], s.data[i+1:]...)
			s.dirty = true
			return s.save()
		}
	}
	return os.ErrNotExist
}

// Get returns a single confusion by ID.
func (s *Store) Get(id string) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.data {
		if c.ID == id {
			return c, nil
		}
	}
	return Confusion{}, os.ErrNotExist
}

// AppendAskMessage appends a message to the confusion's ask exchange (creating
// the Ask object on first use) and persists. New messages reset SummaryState to
// "idle" so a stale summary is not shown after a new turn.
func (s *Store) AppendAskMessage(id string, msg AskMessage) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			if c.Ask == nil {
				c.Ask = &Ask{}
			}
			if msg.ID == "" {
				msg.ID = newID()
			}
			if msg.CreatedAt == "" {
				msg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			c.Ask.Messages = append(c.Ask.Messages, msg)
			c.Ask.SummaryState = "idle"
			c.Ask.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			s.data[i] = c
			s.dirty = true
			return c, s.save()
		}
	}
	return Confusion{}, os.ErrNotExist
}

// SetAskSummary stores the summary and state for a confusion's ask exchange.
// state "done" flips the confusion to StateAsked.
func (s *Store) SetAskSummary(id, summary, state string) (Confusion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data {
		if c.ID == id {
			if c.Ask == nil {
				c.Ask = &Ask{}
			}
			c.Ask.Summary = summary
			c.Ask.SummaryState = state
			c.Ask.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if state == "done" {
				c.State = StateAsked
			}
			s.data[i] = c
			s.dirty = true
			return c, s.save()
		}
	}
	return Confusion{}, os.ErrNotExist
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
