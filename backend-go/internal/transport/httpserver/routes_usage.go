package httpserver

import (
	"net/http"
	"strconv"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

func (s *Server) handleListTeacherUsage(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	rows, err := teacher.ListTeacherUsage()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	start := (page - 1) * pageSize
	if start > len(rows) {
		start = len(rows)
	}
	end := start + pageSize
	if end > len(rows) {
		end = len(rows)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"page": page, "pageSize": pageSize, "total": len(rows), "conversations": rows[start:end]})
}
