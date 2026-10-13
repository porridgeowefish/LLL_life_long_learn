package httpserver

import (
	"context"
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

func (s *Server) handleTeacherOpening(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	st, err := teacher.NewConversation(slug)
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	var input struct {
		ProviderID string `json:"providerId"`
	}
	if err = httpx.ReadJSON(r, &input); err != nil {
		httpx.Error(w, 400, "invalid opening request")
		return
	}
	lock := s.teacherTurnLock(slug)
	lock.Lock()
	if run := s.teacherResponses().Project(slug); run != nil {
		lock.Unlock()
		streamTeacherRun(w, r, run)
		return
	}
	projection, err := st.Read(0, 500)
	if err != nil {
		lock.Unlock()
		learningWorkspaceError(w, err)
		return
	}
	claimed := false
	for _, message := range projection.Messages {
		if message.OperationID != "teacher-welcome-v1" {
			claimed = true
		}
	}
	if claimed {
		lock.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	accepted, err := st.ClaimOpening()
	if err != nil {
		lock.Unlock()
		learningWorkspaceError(w, err)
		return
	}
	if !accepted {
		lock.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	run := s.startTeacherRunLocked(slug, func(ctx context.Context, emit func(teacher.StreamFrame)) error {
		return s.teacher.StreamOpening(ctx, slug, input.ProviderID, emit)
	})
	lock.Unlock()
	if run == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	streamTeacherRun(w, r, run)
}
