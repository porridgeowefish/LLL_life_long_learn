package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/practicestore"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

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
