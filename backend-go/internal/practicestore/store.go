// Package practicestore manages practice/tasks.md and submission/evaluation
// files under a project's practice/ directory. File-backed (iter-03).
package practicestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Task represents one practice question.
type Task struct {
	ID      string `json:"id"`
	Question string `json:"question"` // markdown
	Type    string `json:"type"`      // "short-answer" | "essay" | "code"
}

// TasksFile is the on-disk structure for practice/tasks.md (actually JSON for
// structured parsing, with a .md extension because it lives in the practice dir).
type TasksFile struct {
	Tasks     []Task  `json:"tasks"`
	GeneratedAt string `json:"generatedAt"`
}

// Submission represents one learner answer to a task.
type Submission struct {
	TaskID      string `json:"taskId"`
	Answer      string `json:"answer"`       // markdown
	SelfAssess  int    `json:"selfAssess"`    // 1-5 mastery scale
}

// EvaluationResult is per-task evaluation output.
type EvaluationResult struct {
	TaskID    string `json:"taskId"`
	Score     int    `json:"score"`      // 1-5
	Feedback  string `json:"feedback"`   // markdown
	Evidence  string `json:"evidence"`   // quoted answer snippets
	Passed    bool   `json:"passed"`
}

// Evaluation is the full evaluation for one attempt.
type Evaluation struct {
	Attempt     int                `json:"attempt"`
	Results     []EvaluationResult `json:"results"`
	OverallScore float64           `json:"overallScore"`
	GeneratedAt string             `json:"generatedAt"`
}

// Store reads/writes practice files for a project.
type Store struct {
	projectRoot string
}

// New creates a practice store for the given project slug.
func New(slug string) (*Store, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	return &Store{projectRoot: root}, nil
}

func (s *Store) practiceDir() string {
	return filepath.Join(s.projectRoot, "practice")
}

// ReadTasks reads practice/tasks.json. Returns nil if not found.
func (s *Store) ReadTasks() (*TasksFile, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "tasks.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var tf TasksFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		return nil, fmt.Errorf("parse tasks.json: %w", err)
	}
	return &tf, nil
}

// WriteTasks writes practice/tasks.json.
func (s *Store) WriteTasks(tf *TasksFile) error {
	dir := s.practiceDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return err
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, "tasks.json"), raw, 0o644)
}

// WriteSubmission writes a batch of answers for one attempt.
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

// ReadSubmission reads a submission by attempt number.
func (s *Store) ReadSubmission(attempt int) ([]Submission, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "submissions", fmt.Sprintf("%d.json", attempt)))
	if err != nil {
		return nil, err
	}
	var subs []Submission
	if err := json.Unmarshal(raw, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

// WriteEvaluation writes the evaluation result for one attempt.
func (s *Store) WriteEvaluation(ev *Evaluation) error {
	dir := filepath.Join(s.practiceDir(), "evaluations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Write JSON
	raw, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	if err := workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", ev.Attempt)), raw, 0o644); err != nil {
		return err
	}
	// Write human-readable markdown
	var md string
	md += fmt.Sprintf("# 练习评估（第 %d 次）\n\n", ev.Attempt)
	md += fmt.Sprintf("- 总分：%.1f / 5\n", ev.OverallScore)
	md += fmt.Sprintf("- 时间：%s\n\n", ev.GeneratedAt)
	for _, r := range ev.Results {
		pass := "❌"
		if r.Passed {
			pass = "✅"
		}
		md += fmt.Sprintf("## %s 任务 %s\n\n", pass, r.TaskID)
		md += fmt.Sprintf("- 得分：%d / 5\n", r.Score)
		md += fmt.Sprintf("- 反馈：%s\n\n", r.Feedback)
		if r.Evidence != "" {
			md += fmt.Sprintf("> %s\n\n", r.Evidence)
		}
	}
	return workspace.AtomicWriteFile(filepath.Join(dir, fmt.Sprintf("%d.md", ev.Attempt)), []byte(md), 0o644)
}

// ReadEvaluation reads an evaluation by attempt number.
func (s *Store) ReadEvaluation(attempt int) (*Evaluation, error) {
	raw, err := os.ReadFile(filepath.Join(s.practiceDir(), "evaluations", fmt.Sprintf("%d.json", attempt)))
	if err != nil {
		return nil, err
	}
	var ev Evaluation
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// NextAttempt returns the next attempt number (1-based).
func (s *Store) NextAttempt() int {
	for i := 1; ; i++ {
		p := filepath.Join(s.practiceDir(), "submissions", fmt.Sprintf("%d.json", i))
		if _, err := os.Stat(p); err != nil {
			return i
		}
	}
}

// Timestamp helper.
func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
