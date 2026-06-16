package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/xmz14/lll/backend-go/internal/claudelauncher"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/practicestore"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

var practiceEvaluationJobs sync.Map

func practiceStoreForRequest(w http.ResponseWriter, slug string) (*practicestore.Store, bool) {
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return nil, false
	}
	store, err := practicestore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return store, true
}

func (s *Server) handleGetPracticeTasks(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	tasks, err := store.ReadTasks()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"tasks": nil, "generated": false})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"schemaVersion": tasks.SchemaVersion,
		"setId":         tasks.SetID,
		"tasks":         tasks.Tasks,
		"generated":     true,
		"generatedAt":   tasks.GeneratedAt,
	})
}

func (s *Server) handleGetPracticeDraft(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	draft, err := store.ReadDraft()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"draft": draft})
}

func (s *Server) handlePutPracticeDraft(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	tasks, err := store.ReadTasks()
	if err != nil || tasks == nil {
		httpx.Error(w, http.StatusConflict, "practice tasks not available")
		return
	}
	var draft practicestore.DraftFile
	if err := httpx.ReadJSON(r, &draft); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if draft.SetID != "" && tasks.SetID != "" && draft.SetID != tasks.SetID {
		httpx.Error(w, http.StatusConflict, "draft does not match current task set")
		return
	}
	if draft.GeneratedAt != "" && tasks.GeneratedAt != "" && draft.GeneratedAt != tasks.GeneratedAt {
		httpx.Error(w, http.StatusConflict, "draft does not match current generatedAt")
		return
	}
	draft.SetID = tasks.SetID
	draft.GeneratedAt = tasks.GeneratedAt
	draft.TaskIDs = taskIDs(tasks)
	if err := store.WriteDraft(&draft); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"draft": &draft})
}

func (s *Server) handleCreatePracticeAttempt(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	tasks, err := store.ReadTasks()
	if err != nil || tasks == nil {
		httpx.Error(w, http.StatusConflict, "practice tasks not available")
		return
	}
	attempt, err := store.CreateAttempt(tasks.SetID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"attempt": attempt.Attempt, "setId": attempt.SetID})
}

func taskIDs(tasks *practicestore.TasksFile) []string {
	if tasks == nil {
		return nil
	}
	ids := make([]string, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func (s *Server) handleGetLatestPracticeAttempt(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	attempt, err := store.ReadLatestSubmittedAttempt()
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "practice attempt not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"attempt": attempt})
}

func (s *Server) handleCheckObjective(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	store, ok := practiceStoreForRequest(w, slug)
	if !ok {
		return
	}
	attemptID, err := strconv.Atoi(r.PathValue("attempt"))
	if err != nil || attemptID < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid attempt")
		return
	}
	var body struct {
		Answer json.RawMessage `json:"answer"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil || len(body.Answer) == 0 {
		httpx.Error(w, http.StatusBadRequest, "answer is required")
		return
	}
	taskID := r.PathValue("taskId")
	result, err := store.CheckObjective(attemptID, taskID, body.Answer)
	if err != nil {
		httpx.Error(w, http.StatusConflict, err.Error())
		return
	}
	tasks, _ := store.ReadTasks()
	difficulty := taskDifficulty(tasks, taskID)
	progress, _ := progressstore.New(slug)
	delta := 0
	if progress != nil {
		_, added, summary, awardErr := progress.Award(progressstore.Event{
			ID:         fmt.Sprintf("practice-submit:%d:%s", attemptID, taskID),
			SourceType: "practice-submit",
			SourceID:   taskID,
			AttemptID:  strconv.Itoa(attemptID),
			Difficulty: difficulty,
			Outcome:    "submitted",
			Delta:      difficulty,
		})
		if awardErr == nil && added {
			delta += difficulty
		}
		if result.Correct {
			_, correctAdded, nextSummary, correctErr := progress.Award(progressstore.Event{
				ID:         fmt.Sprintf("practice-correct:%d:%s", attemptID, taskID),
				SourceType: "practice-correct",
				SourceID:   taskID,
				AttemptID:  strconv.Itoa(attemptID),
				Difficulty: difficulty,
				Outcome:    "correct",
				Delta:      difficulty,
			})
			if correctErr == nil {
				summary = nextSummary
				if correctAdded {
					delta += difficulty
				}
			}
		}
		result.GrowthDelta = delta
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"result": result, "growthDelta": delta, "progress": summary,
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"result": result, "growthDelta": 0})
}

func (s *Server) handleSubmitPracticeAttempt(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	store, ok := practiceStoreForRequest(w, slug)
	if !ok {
		return
	}
	attemptID, err := strconv.Atoi(r.PathValue("attempt"))
	if err != nil || attemptID < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid attempt")
		return
	}
	var body struct {
		Submissions []practicestore.Submission `json:"submissions"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	attempt, err := store.SubmitAttempt(attemptID, body.Submissions)
	if err != nil {
		httpx.Error(w, http.StatusConflict, err.Error())
		return
	}
	tasks, _ := store.ReadTasks()
	progress, _ := progressstore.New(slug)
	growthDelta := 0
	if progress != nil {
		for _, sub := range body.Submissions {
			task := findTask(tasks, sub.TaskID)
			if task == nil || task.IsObjective() || len(sub.Answer) == 0 {
				continue
			}
			_, added, _, awardErr := progress.Award(progressstore.Event{
				ID:         fmt.Sprintf("practice-submit:%d:%s", attemptID, sub.TaskID),
				SourceType: "practice-submit",
				SourceID:   sub.TaskID,
				AttemptID:  strconv.Itoa(attemptID),
				Difficulty: task.Difficulty,
				Outcome:    "submitted",
				Delta:      task.Difficulty,
			})
			if awardErr == nil && added {
				growthDelta += task.Difficulty
			}
		}
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"attempt": attempt.Attempt, "saved": len(body.Submissions), "growthDelta": growthDelta,
	})
}

// Compatibility endpoint: create and submit an attempt in one request.
func (s *Server) handleSubmitPractice(w http.ResponseWriter, r *http.Request) {
	store, ok := practiceStoreForRequest(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		Submissions []practicestore.Submission `json:"submissions"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil || len(body.Submissions) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no submissions")
		return
	}
	tasks, _ := store.ReadTasks()
	setID := "legacy"
	if tasks != nil {
		setID = tasks.SetID
	}
	attempt, err := store.CreateAttempt(setID)
	if err == nil {
		attempt, err = store.SubmitAttempt(attempt.Attempt, body.Submissions)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"attempt": attempt.Attempt, "saved": len(body.Submissions)})
}

func (s *Server) handleGetPracticeEvaluation(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	store, ok := practiceStoreForRequest(w, slug)
	if !ok {
		return
	}
	attemptID, err := strconv.Atoi(r.URL.Query().Get("attempt"))
	if err != nil || attemptID < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid attempt")
		return
	}
	ev, err := store.ReadEvaluation(attemptID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "evaluation not found")
		return
	}
	tasks, _ := store.ReadTasks()
	progress, _ := progressstore.New(slug)
	growthDelta := 0
	if progress != nil {
		for _, result := range ev.Results {
			task := findTask(tasks, result.TaskID)
			if task == nil || task.IsObjective() {
				continue
			}
			delta := int(math.Round(float64(task.Difficulty*result.Score) / 5))
			_, added, _, awardErr := progress.Award(progressstore.Event{
				ID:         fmt.Sprintf("practice-evaluation:%d:%s", attemptID, result.TaskID),
				SourceType: "practice-evaluation",
				SourceID:   result.TaskID,
				AttemptID:  strconv.Itoa(attemptID),
				Difficulty: task.Difficulty,
				Outcome:    "evaluated",
				Delta:      delta,
			})
			if awardErr == nil && added {
				growthDelta += delta
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"evaluation": ev, "growthDelta": growthDelta})
}

func (s *Server) handleRequestPracticeEvaluation(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	store, ok := practiceStoreForRequest(w, slug)
	if !ok {
		return
	}
	attemptID, err := strconv.Atoi(r.PathValue("attempt"))
	if err != nil || attemptID < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid attempt")
		return
	}
	attempt, err := store.ReadAttempt(attemptID)
	if err != nil || attempt.Status != "submitted" || !attemptHasEvaluableSubmission(attempt) {
		httpx.Error(w, http.StatusConflict, "submitted subjective answers are required")
		return
	}
	if _, err := store.ReadEvaluation(attemptID); err == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "complete", "attempt": attemptID})
		return
	}
	if !s.ClaudeAvailable {
		httpx.Error(w, http.StatusServiceUnavailable, "AI evaluation is unavailable")
		return
	}
	jobKey := fmt.Sprintf("%s:%d", slug, attemptID)
	if _, loaded := practiceEvaluationJobs.LoadOrStore(jobKey, true); loaded {
		httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "running", "attempt": attemptID})
		return
	}
	agent, found := agents.Get("practice")
	if !found {
		practiceEvaluationJobs.Delete(jobKey)
		httpx.Error(w, http.StatusInternalServerError, "practice agent not found")
		return
	}
	pkg, err := promptassembly.Build(promptassembly.Request{
		ProjectSlug:     slug,
		ZoneName:        workspace.ZonePractice,
		AgentID:         agent.ID,
		PracticeAttempt: attemptID,
	}, agents)
	if err != nil {
		practiceEvaluationJobs.Delete(jobKey)
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	go func() {
		defer practiceEvaluationJobs.Delete(jobKey)
		_ = claudelauncher.LaunchHeadless(context.Background(), slug, agent.ID, s.ClaudeBin, "", pkg)
	}()
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "queued", "attempt": attemptID})
}

func attemptHasEvaluableSubmission(attempt *practicestore.Attempt) bool {
	for _, submission := range attempt.Submissions {
		raw := string(bytes.TrimSpace(submission.Answer))
		if raw != "" && raw != `""` && raw != "null" && raw != "[]" {
			return true
		}
	}
	return false
}

func (s *Server) handleGetProgress(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := progressstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	summary, err := store.Summary()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"progress": summary})
}

func findTask(tasks *practicestore.TasksFile, taskID string) *practicestore.Task {
	if tasks == nil {
		return nil
	}
	for i := range tasks.Tasks {
		if tasks.Tasks[i].ID == taskID {
			return &tasks.Tasks[i]
		}
	}
	return nil
}

func taskDifficulty(tasks *practicestore.TasksFile, taskID string) int {
	if task := findTask(tasks, taskID); task != nil {
		return task.Difficulty
	}
	return 3
}
