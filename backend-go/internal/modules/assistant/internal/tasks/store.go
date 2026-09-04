package assistanttask

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const SchemaVersion = 1

var validTypes = map[string]bool{
	"consolidate":       true,
	"verify":            true,
	"produce-material":  true,
	"source-processing": true,
}

var terminalStatuses = map[string]bool{"succeeded": true, "partial": true, "failed": true, "cancelled": true}

type Origin struct {
	Kind              string `json:"kind"`
	OperationID       string `json:"operationId"`
	ProposalMessageID string `json:"proposalMessageId,omitempty"`
	ApprovalMessageID string `json:"approvalMessageId,omitempty"`
	ToolCallID        string `json:"toolCallId,omitempty"`
	SourceRevisionID  string `json:"sourceRevisionId,omitempty"`
}

type Failure struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	Suggestion string `json:"suggestion,omitempty"`
}

type Result struct {
	Summary      string            `json:"summary"`
	Deliverables []string          `json:"deliverables,omitempty"`
	AssetUpdates map[string]string `json:"assetUpdates,omitempty"`
}

type Lease struct {
	RunID              string    `json:"runId"`
	DispatcherInstance string    `json:"dispatcherInstance"`
	AcquiredAt         time.Time `json:"acquiredAt"`
	LastHeartbeatAt    time.Time `json:"lastHeartbeatAt"`
}

type Task struct {
	SchemaVersion         int       `json:"schemaVersion"`
	ID                    string    `json:"id"`
	UnitID                string    `json:"unitId"`
	ProjectSlug           string    `json:"projectSlug"`
	Type                  string    `json:"type"`
	Objective             string    `json:"objective"`
	SourceRefs            []string  `json:"sourceRefs"`
	Status                string    `json:"status"`
	Phase                 string    `json:"phase,omitempty"`
	Origin                Origin    `json:"origin"`
	ConversationCutoffSeq uint64    `json:"conversationCutoffSeq"`
	AttemptIDs            []string  `json:"attemptIds"`
	Result                *Result   `json:"result,omitempty"`
	Failure               *Failure  `json:"failure,omitempty"`
	Lease                 *Lease    `json:"lease,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Type                  string
	Objective             string
	SourceRefs            []string
	Origin                Origin
	ConversationCutoffSeq uint64
}

type SameTypeActiveError struct{ ExistingTaskID string }

func (e *SameTypeActiveError) Error() string               { return "same task type is already active" }
func (e *SameTypeActiveError) DelegationCode() string      { return "same_type_active" }
func (e *SameTypeActiveError) ExistingTaskIDValue() string { return e.ExistingTaskID }

type OperationConflictError struct{ ExistingTaskID string }

func (e *OperationConflictError) Error() string {
	return "operation id reused with different task input"
}
func (e *OperationConflictError) DelegationCode() string      { return "operation_conflict" }
func (e *OperationConflictError) ExistingTaskIDValue() string { return e.ExistingTaskID }

type Store struct {
	slug   string
	unitID string
	root   string
}

var workspaceLock sync.Mutex

func New(slug string) (*Store, error) {
	conversation, err := teacher.NewConversation(slug)
	if err != nil {
		return nil, err
	}
	unit, err := conversation.Unit()
	if err != nil {
		return nil, err
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	taskRoot := filepath.Join(root, "assistant-tasks")
	if err := os.MkdirAll(taskRoot, 0o755); err != nil {
		return nil, err
	}
	return &Store{slug: slug, unitID: unit.UnitID, root: taskRoot}, nil
}

func ValidType(taskType string) bool { return validTypes[taskType] }
func IsTerminal(status string) bool  { return terminalStatuses[status] }

func (s *Store) Create(in CreateInput) (Task, bool, error) {
	in.Objective = strings.TrimSpace(in.Objective)
	if !validTypes[in.Type] {
		return Task{}, false, errors.New("invalid task type")
	}
	if in.Objective == "" || len([]byte(in.Objective)) > 8<<10 {
		return Task{}, false, errors.New("invalid task objective")
	}
	if strings.TrimSpace(in.Origin.OperationID) == "" {
		return Task{}, false, errors.New("operation id is required")
	}
	workspaceLock.Lock()
	defer workspaceLock.Unlock()
	existing, err := s.listUnlocked()
	if err != nil {
		return Task{}, false, err
	}
	for _, task := range existing {
		if task.Origin.OperationID == in.Origin.OperationID {
			if sameCreate(task, in) {
				return task, false, nil
			}
			return Task{}, false, &OperationConflictError{ExistingTaskID: task.ID}
		}
		if task.Type == in.Type && !IsTerminal(task.Status) {
			return Task{}, false, &SameTypeActiveError{ExistingTaskID: task.ID}
		}
	}
	now := time.Now().UTC()
	task := Task{SchemaVersion: SchemaVersion, ID: idgen.New("task"), UnitID: s.unitID, ProjectSlug: s.slug, Type: in.Type, Objective: in.Objective, SourceRefs: dedupe(in.SourceRefs), Status: "queued", Origin: in.Origin, ConversationCutoffSeq: in.ConversationCutoffSeq, CreatedAt: now, UpdatedAt: now}
	dir := filepath.Join(s.root, task.ID)
	if err := os.MkdirAll(filepath.Join(dir, "attempts"), 0o755); err != nil {
		return Task{}, false, err
	}
	if err := writeJSON(filepath.Join(dir, "task.json"), task); err != nil {
		return Task{}, false, err
	}
	return task, true, nil
}

func (s *Store) Get(taskID string) (Task, error) {
	if !safeID(taskID, "task_") {
		return Task{}, os.ErrNotExist
	}
	workspaceLock.Lock()
	defer workspaceLock.Unlock()
	return readTask(filepath.Join(s.root, taskID, "task.json"))
}

func (s *Store) List(status string) ([]Task, error) {
	workspaceLock.Lock()
	defer workspaceLock.Unlock()
	all, err := s.listUnlocked()
	if err != nil {
		return nil, err
	}
	if status == "" {
		return all, nil
	}
	out := all[:0]
	for _, task := range all {
		if task.Status == status {
			out = append(out, task)
		}
	}
	return out, nil
}

func (s *Store) Update(taskID string, mutate func(*Task) error) (Task, error) {
	if !safeID(taskID, "task_") {
		return Task{}, os.ErrNotExist
	}
	workspaceLock.Lock()
	defer workspaceLock.Unlock()
	path := filepath.Join(s.root, taskID, "task.json")
	task, err := readTask(path)
	if err != nil {
		return Task{}, err
	}
	if err := mutate(&task); err != nil {
		return Task{}, err
	}
	if !validStatus(task.Status) {
		return Task{}, errors.New("invalid task status")
	}
	task.UpdatedAt = time.Now().UTC()
	return task, writeJSON(path, task)
}

func (s *Store) HasActiveSource(sourceID string) (bool, error) {
	tasks, err := s.List("")
	if err != nil {
		return false, err
	}
	for _, task := range tasks {
		if IsTerminal(task.Status) {
			continue
		}
		for _, ref := range task.SourceRefs {
			if ref == sourceID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Store) listUnlocked() ([]Task, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		task, err := readTask(filepath.Join(s.root, entry.Name(), "task.json"))
		if err == nil {
			out = append(out, task)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func sameCreate(task Task, in CreateInput) bool {
	return task.Type == in.Type && task.Objective == in.Objective && strings.Join(task.SourceRefs, "\x00") == strings.Join(dedupe(in.SourceRefs), "\x00")
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func validStatus(status string) bool {
	switch status {
	case "queued", "running", "succeeded", "partial", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func safeID(id, prefix string) bool {
	return strings.HasPrefix(id, prefix) && !strings.ContainsAny(id, `/\\`) && len(id) <= 96
}

func readTask(path string) (Task, error) {
	var task Task
	data, err := os.ReadFile(path)
	if err != nil {
		return task, err
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return task, fmt.Errorf("decode task: %w", err)
	}
	return task, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(path, append(data, '\n'), 0o644)
}
