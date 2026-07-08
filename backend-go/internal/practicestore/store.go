// Package practicestore manages public questions, private answer keys,
// attempts, submissions, and evaluations for the Practice zone.
package practicestore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const (
	TypeTrueFalse      = "true-false"
	TypeSingleChoice   = "single-choice"
	TypeMultipleChoice = "multiple-choice"
	TypeShortAnswer    = "short-answer"
	TypeEssay          = "essay"
	TypeCode           = "code"
)

type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Task struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	Type       string   `json:"type"`
	Difficulty int      `json:"difficulty"`
	Options    []Option `json:"options,omitempty"`
	SourceRefs []string `json:"sourceRefs,omitempty"`
}

func (t Task) IsObjective() bool {
	return t.Type == TypeTrueFalse || t.Type == TypeSingleChoice || t.Type == TypeMultipleChoice
}

type TasksFile struct {
	SchemaVersion int    `json:"schemaVersion"`
	SetID         string `json:"setId"`
	Tasks         []Task `json:"tasks"`
	GeneratedAt   string `json:"generatedAt"`
}

type AnswerKeyEntry struct {
	TaskID        string          `json:"taskId"`
	CorrectAnswer json.RawMessage `json:"correctAnswer"`
	Explanation   string          `json:"explanation"`
	SourceRefs    []string        `json:"sourceRefs,omitempty"`
}

type AnswerKeyFile struct {
	SchemaVersion int              `json:"schemaVersion"`
	SetID         string           `json:"setId"`
	Answers       []AnswerKeyEntry `json:"answers"`
}

type Submission struct {
	TaskID     string          `json:"taskId"`
	Answer     json.RawMessage `json:"answer"`
	SelfAssess int             `json:"selfAssess"`
}

type ObjectiveResult struct {
	TaskID        string          `json:"taskId"`
	Answer        json.RawMessage `json:"answer"`
	Correct       bool            `json:"correct"`
	CorrectAnswer json.RawMessage `json:"correctAnswer"`
	Explanation   string          `json:"explanation"`
	LockedAt      string          `json:"lockedAt"`
	GrowthDelta   int             `json:"growthDelta"`
}

type Attempt struct {
	Attempt          int                        `json:"attempt"`
	SetID            string                     `json:"setId"`
	Status           string                     `json:"status"`
	ObjectiveResults map[string]ObjectiveResult `json:"objectiveResults"`
	Submissions      []Submission               `json:"submissions,omitempty"`
	CreatedAt        string                     `json:"createdAt"`
	SubmittedAt      string                     `json:"submittedAt,omitempty"`
}

type DraftEntry struct {
	Answer     json.RawMessage `json:"answer"`
	SelfAssess int             `json:"selfAssess"`
}

type DraftFile struct {
	SchemaVersion int                   `json:"schemaVersion"`
	SetID         string                `json:"setId"`
	GeneratedAt   string                `json:"generatedAt"`
	TaskIDs       []string              `json:"taskIds"`
	Drafts        map[string]DraftEntry `json:"drafts"`
	Attempt       int                   `json:"attempt,omitempty"`
	UpdatedAt     string                `json:"updatedAt"`
}

type EvaluationResult struct {
	TaskID          string `json:"taskId"`
	Score           int    `json:"score"`
	Feedback        string `json:"feedback"`
	SuggestedAnswer string `json:"suggestedAnswer,omitempty"`
	Evidence        string `json:"evidence"`
	Passed          bool   `json:"passed"`
}

type Evaluation struct {
	Attempt      int                `json:"attempt"`
	Summary      string             `json:"summary,omitempty"`
	Results      []EvaluationResult `json:"results"`
	OverallScore float64            `json:"overallScore"`
	GeneratedAt  string             `json:"generatedAt"`
}

type Store struct {
	projectRoot string
	mu          *sync.Mutex
}

var projectLocks sync.Map

func projectLock(root string) *sync.Mutex {
	lock, _ := projectLocks.LoadOrStore(root, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func New(slug string) (*Store, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	return &Store{projectRoot: root, mu: projectLock(root)}, nil
}

func (s *Store) practiceDir() string { return filepath.Join(s.projectRoot, "practice") }

func (s *Store) ReadTasks() (*TasksFile, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "tasks.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tf TasksFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		return nil, fmt.Errorf("parse tasks.json: %w", err)
	}
	if tf.SchemaVersion == 0 {
		tf.SchemaVersion = 1
	}
	if tf.SetID == "" {
		tf.SetID = "legacy-" + strings.ReplaceAll(tf.GeneratedAt, ":", "-")
	}
	for i := range tf.Tasks {
		if tf.Tasks[i].Difficulty < 1 || tf.Tasks[i].Difficulty > 5 {
			tf.Tasks[i].Difficulty = 3
		}
		if tf.Tasks[i].Type == "" {
			tf.Tasks[i].Type = TypeEssay
		}
	}
	return &tf, nil
}

func (s *Store) WriteTasks(tf *TasksFile) error {
	if err := os.MkdirAll(s.practiceDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(s.practiceDir(), "tasks.json"), raw, 0o644)
}

func (s *Store) ReadDraft() (*DraftFile, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "draft.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var draft DraftFile
	if err := json.Unmarshal(raw, &draft); err != nil {
		_, _ = workspace.BackupCorruptFile(filepath.Join(s.practiceDir(), "draft.json"), err)
		return nil, nil
	}
	if draft.SchemaVersion == 0 {
		draft.SchemaVersion = 1
	}
	if draft.Drafts == nil {
		draft.Drafts = map[string]DraftEntry{}
	}
	return &draft, nil
}

func (s *Store) WriteDraft(draft *DraftFile) error {
	if draft == nil {
		return errors.New("draft is nil")
	}
	if err := os.MkdirAll(s.practiceDir(), 0o755); err != nil {
		return err
	}
	if draft.SchemaVersion == 0 {
		draft.SchemaVersion = 1
	}
	if draft.Drafts == nil {
		draft.Drafts = map[string]DraftEntry{}
	}
	draft.UpdatedAt = NowISO()
	raw, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(s.practiceDir(), "draft.json"), raw, 0o644)
}

func (s *Store) ReadAnswerKey() (*AnswerKeyFile, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "answer-key.json"))
	if err != nil {
		return nil, err
	}
	var key AnswerKeyFile
	if err := json.Unmarshal(raw, &key); err != nil {
		return nil, fmt.Errorf("parse answer-key.json: %w", err)
	}
	return &key, nil
}

func (s *Store) NextAttempt() int {
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(s.practiceDir(), "attempts", fmt.Sprintf("%d.json", i))); err == nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.practiceDir(), "submissions", fmt.Sprintf("%d.json", i))); err == nil {
			continue
		}
		return i
	}
}

func (s *Store) CreateAttempt(setID string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	attempt := &Attempt{
		Attempt:          s.NextAttempt(),
		SetID:            setID,
		Status:           "answering",
		ObjectiveResults: map[string]ObjectiveResult{},
		CreatedAt:        NowISO(),
	}
	return attempt, s.WriteAttempt(attempt)
}

func (s *Store) ReadAttempt(attempt int) (*Attempt, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "attempts", fmt.Sprintf("%d.json", attempt)))
	if err != nil {
		return nil, err
	}
	var out Attempt
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.ObjectiveResults == nil {
		out.ObjectiveResults = map[string]ObjectiveResult{}
	}
	return &out, nil
}

func (s *Store) ReadLatestSubmittedAttempt() (*Attempt, error) {
	for attemptID := s.NextAttempt() - 1; attemptID >= 1; attemptID-- {
		attempt, err := s.ReadAttempt(attemptID)
		if err != nil {
			continue
		}
		if attempt.Status == "submitted" && attemptHasResponses(attempt) {
			return attempt, nil
		}
	}
	return nil, os.ErrNotExist
}

func attemptHasResponses(attempt *Attempt) bool {
	if len(attempt.ObjectiveResults) > 0 {
		return true
	}
	for _, submission := range attempt.Submissions {
		if len(bytes.TrimSpace(submission.Answer)) == 0 {
			continue
		}
		if string(bytes.TrimSpace(submission.Answer)) != `""` &&
			string(bytes.TrimSpace(submission.Answer)) != "null" &&
			string(bytes.TrimSpace(submission.Answer)) != "[]" {
			return true
		}
	}
	return false
}

func (s *Store) WriteAttempt(attempt *Attempt) error {
	dir := filepath.Join(s.practiceDir(), "attempts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(attempt, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", attempt.Attempt)), raw, 0o644)
}

func (s *Store) CheckObjective(attemptID int, taskID string, answer json.RawMessage) (*ObjectiveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	attempt, err := s.ReadAttempt(attemptID)
	if err != nil {
		return nil, err
	}
	if existing, ok := attempt.ObjectiveResults[taskID]; ok {
		return &existing, nil
	}
	tasks, err := s.ReadTasks()
	if err != nil || tasks == nil {
		return nil, errors.New("tasks not found")
	}
	var task *Task
	for i := range tasks.Tasks {
		if tasks.Tasks[i].ID == taskID {
			task = &tasks.Tasks[i]
			break
		}
	}
	if task == nil || !task.IsObjective() {
		return nil, errors.New("task is not objective")
	}
	key, err := s.ReadAnswerKey()
	if err != nil {
		return nil, err
	}
	if key.SetID != "" && tasks.SetID != "" && key.SetID != tasks.SetID {
		return nil, errors.New("answer key does not match current task set")
	}
	var entry *AnswerKeyEntry
	for i := range key.Answers {
		if key.Answers[i].TaskID == taskID {
			entry = &key.Answers[i]
			break
		}
	}
	if entry == nil {
		return nil, errors.New("answer key entry not found")
	}
	result := ObjectiveResult{
		TaskID:        taskID,
		Answer:        cloneRaw(answer),
		Correct:       answersEqual(task.Type, answer, entry.CorrectAnswer),
		CorrectAnswer: cloneRaw(entry.CorrectAnswer),
		Explanation:   entry.Explanation,
		LockedAt:      NowISO(),
	}
	attempt.ObjectiveResults[taskID] = result
	if err := s.WriteAttempt(attempt); err != nil {
		return nil, err
	}
	return &result, nil
}

func answersEqual(taskType string, got, want json.RawMessage) bool {
	switch taskType {
	case TypeMultipleChoice:
		var a, b []string
		if json.Unmarshal(got, &a) != nil || json.Unmarshal(want, &b) != nil {
			return false
		}
		sort.Strings(a)
		sort.Strings(b)
		return strings.Join(a, "\x00") == strings.Join(b, "\x00")
	case TypeTrueFalse:
		var a, b bool
		if json.Unmarshal(got, &a) == nil && json.Unmarshal(want, &b) == nil {
			return a == b
		}
	}
	return bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want))
}

func cloneRaw(in json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), in...)
}

func (s *Store) SubmitAttempt(attemptID int, subs []Submission) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	attempt, err := s.ReadAttempt(attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.Status == "submitted" {
		return attempt, nil
	}
	attempt.Submissions = subs
	attempt.Status = "submitted"
	attempt.SubmittedAt = NowISO()
	if err := s.WriteAttempt(attempt); err != nil {
		return nil, err
	}
	if err := s.WriteSubmission(attemptID, subs); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *Store) WriteSubmission(attempt int, subs []Submission) error {
	dir := filepath.Join(s.practiceDir(), "submissions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", attempt)), raw, 0o644)
}

func (s *Store) ReadSubmission(attempt int) ([]Submission, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "submissions", fmt.Sprintf("%d.json", attempt)))
	if err != nil {
		return nil, err
	}
	var subs []Submission
	return subs, json.Unmarshal(raw, &subs)
}

func (s *Store) WriteEvaluation(ev *Evaluation) error {
	dir := filepath.Join(s.practiceDir(), "evaluations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	if err := workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", ev.Attempt)), raw, 0o644); err != nil {
		return err
	}
	var md strings.Builder
	fmt.Fprintf(&md, "# 练习评估（第 %d 次）\n\n- 总分：%.1f / 5\n- 时间：%s\n\n", ev.Attempt, ev.OverallScore, ev.GeneratedAt)
	if ev.Summary != "" {
		fmt.Fprintf(&md, "## 总体反馈\n\n%s\n\n", ev.Summary)
	}
	for _, r := range ev.Results {
		fmt.Fprintf(&md, "## 任务 %s\n\n- 得分：%d / 5\n- 反馈：%s\n\n", r.TaskID, r.Score, r.Feedback)
		if r.SuggestedAnswer != "" {
			fmt.Fprintf(&md, "### 参考回答\n\n%s\n\n", r.SuggestedAnswer)
		}
		if r.Evidence != "" {
			fmt.Fprintf(&md, "> %s\n\n", r.Evidence)
		}
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.md", ev.Attempt)), []byte(md.String()), 0o644)
}

func (s *Store) ReadEvaluation(attempt int) (*Evaluation, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "evaluations", strconv.Itoa(attempt)+".json"))
	if err != nil {
		return nil, err
	}
	var ev Evaluation
	return &ev, json.Unmarshal(raw, &ev)
}

func NowISO() string { return time.Now().UTC().Format(time.RFC3339) }
