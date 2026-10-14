package httpserver

import (
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"net/http"
)

func (s *Server) handleConversationInstructions(w http.ResponseWriter, r *http.Request) {
	st, err := teacher.NewConversation(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if r.Method == http.MethodPut {
		var input struct {
			SystemPrompt *string `json:"systemPrompt"`
		}
		if err := httpx.ReadJSON(r, &input); err != nil || input.SystemPrompt == nil {
			httpx.Error(w, http.StatusBadRequest, "systemPrompt must be a string")
			return
		}
		if len(*input.SystemPrompt) > 32*1024 {
			httpx.Error(w, http.StatusBadRequest, "conversation instructions exceed 32768 UTF-8 bytes")
			return
		}
		if err := st.SetSystemPrompt(*input.SystemPrompt); err != nil {
			learningWorkspaceError(w, err)
			return
		}
	}
	value, err := st.SystemPrompt()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"systemPrompt": value})
}
