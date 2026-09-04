package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	learningscope "github.com/xmz14/lll/backend-go/internal/modules/learning"
	progressstore "github.com/xmz14/lll/backend-go/internal/modules/learning"
	folderstore "github.com/xmz14/lll/backend-go/internal/modules/projects"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
)

// contentTypeFor returns the appropriate Content-Type for a file based on its extension.
func contentTypeFor(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "text/plain; charset=utf-8"
	}
}

// handleDeleteProject permanently removes one learning project and its local
// artifacts. The frontend owns the single user-confirmation step; the backend
// owns validation, active-run protection, and associated reference cleanup.
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	if s.sessions.HasActiveProject(slug) {
		httpx.Error(w, http.StatusConflict, "project_has_active_session")
		return
	}
	if err := workspace.DeleteProject(slug); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httpx.Error(w, http.StatusNotFound, "project not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	store, err := folderstore.NewFolderStore()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "project deleted but folder cleanup failed: "+err.Error())
		return
	}
	if _, err := store.RemoveProject(slug); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "project deleted but folder cleanup failed: "+err.Error())
		return
	}
	s.cache.Invalidate(slug)
	s.sessions.RemoveProject(slug)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true, "projectId": slug})
}

// validEditableFilename limits learner-owned zone writes to safe filenames.
var validEditableFilename = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)

// cache is the package-global project index. Populated on startup and on writes.

// createProjectRequest is the body of POST /api/projects.
// Why/Current/Target/Standard seed only system-learning project.md files.
// Discipline-map creation ignores them at the workspace boundary.
type createProjectRequest struct {
	Title       string                      `json:"title"`
	Slug        string                      `json:"slug,omitempty"`
	ProjectType workspace.ProjectType       `json:"projectType,omitempty"`
	Why         string                      `json:"why,omitempty"`
	Current     string                      `json:"current,omitempty"`
	Target      string                      `json:"target,omitempty"`
	Standard    string                      `json:"standard,omitempty"`
	ScopeSource *learningScopeSourceRequest `json:"scopeSource,omitempty"`
}

type learningScopeSourceRequest struct {
	Type    string `json:"type"`
	MapSlug string `json:"mapSlug"`
	TopicID string `json:"topicId"`
}

// handleListProjects returns the indexed project tree.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if err := s.cache.Rebuild(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "rebuild: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": s.cache.All()})
}

// handleCreateProject creates a flat, independent project.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		httpx.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = workspace.Slugify(title)
	}
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug: "+slug)
		return
	}
	projectType := workspace.NormalizeProjectType(req.ProjectType)
	if !workspace.ValidateProjectType(projectType) {
		httpx.Error(w, http.StatusBadRequest, "invalid_project_type")
		return
	}
	var scope *learningscope.Scope
	if req.ScopeSource != nil {
		if projectType != workspace.ProjectTypeSystemLearning {
			httpx.Error(w, http.StatusBadRequest, "scope_source_requires_system_learning")
			return
		}
		resolved, err := resolveMapLearningScope(*req.ScopeSource)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		scope = &resolved
	}
	if err := workspace.CreateProjectSkeletonWithInput(slug, title, "", workspace.ProjectInput{
		ProjectType:   projectType,
		Why:           req.Why,
		Current:       req.Current,
		Target:        req.Target,
		Standard:      req.Standard,
		LearningScope: scope,
	}); err != nil {
		if workspace.IsSlugConflict(err) {
			httpx.Error(w, http.StatusConflict, "project already exists: "+slug)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.cache.Invalidate(slug)
	state, _ := workspace.ReadProjectState(slug)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"project": state})
}

func resolveMapLearningScope(source learningScopeSourceRequest) (learningscope.Scope, error) {
	source.Type = strings.TrimSpace(source.Type)
	source.MapSlug = strings.TrimSpace(source.MapSlug)
	source.TopicID = strings.TrimSpace(source.TopicID)
	if source.Type != learningscope.SourceDisciplineMap ||
		!workspace.ValidateSlug(source.MapSlug) || source.TopicID == "" {
		return learningscope.Scope{}, errors.New("invalid_scope_source")
	}
	state, err := workspace.ReadProjectState(source.MapSlug)
	if err != nil || state.ProjectType != workspace.ProjectTypeDisciplineMap {
		return learningscope.Scope{}, errors.New("invalid_scope_source")
	}
	root, err := workspace.ProjectRootForSlug(source.MapSlug)
	if err != nil {
		return learningscope.Scope{}, errors.New("invalid_scope_source")
	}
	catalog, err := readDisciplineTopics(root)
	if err != nil {
		return learningscope.Scope{}, errors.New("discipline_topics_unavailable")
	}
	topic, ok := learningscope.FindTopic(catalog, source.TopicID)
	if !ok {
		return learningscope.Scope{}, errors.New("discipline_topic_not_found")
	}
	return learningscope.SnapshotFromTopic(source.MapSlug, topic, time.Now()), nil
}

// handleGetProject returns one project's full state.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		if errors.Is(err, errors.New("not found")) || strings.Contains(err.Error(), "not exist") {
			httpx.Error(w, http.StatusNotFound, "project not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"project": struct {
			*workspace.ProjectState
			GeneratedZones []workspace.ZoneName `json:"generatedZones"`
		}{
			ProjectState:   state,
			GeneratedZones: state.GeneratedZones,
		},
	})
}

// handleProjectTree preserves the legacy endpoint name but returns the same
// flat, peer-level project set used by the sidebar.
func (s *Server) handleProjectTree(w http.ResponseWriter, r *http.Request) {
	if err := s.cache.Rebuild(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	flat := s.cache.All()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": flat})
}

// handleGetZone returns a single zone resource.
func (s *Server) handleGetZone(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	zone := r.PathValue("zone")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	if !workspace.ValidateZoneName(zone) {
		httpx.Error(w, http.StatusBadRequest, "invalid zone: "+zone)
		return
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "project_not_found")
		return
	}
	if state.ProjectType != workspace.ProjectTypeSystemLearning {
		httpx.Error(w, http.StatusBadRequest, "project_has_no_zones")
		return
	}
	zoneName := workspace.ZoneName(zone)
	preds, err := workspace.ResolvePredecessorFiles(slug, zoneName)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"zone":         zoneName,
		"predecessors": preds,
	})
}

// handleReadFile returns the raw text of a project file by relative path.
// Restricted to summary/ and zone folders. Global learner preferences use the
// dedicated workspace-level preferences endpoint.
func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/files/projects/"+slug+"/")
	if rel == "" || strings.Contains(rel, "..") || strings.ContainsAny(rel, "\\") {
		httpx.Error(w, http.StatusBadRequest, "invalid path")
		return
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	// Path traversal check
	relCheck, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		httpx.Error(w, http.StatusBadRequest, "path escapes project")
		return
	}
	// Allow only specific dirs.
	parts := strings.Split(filepath.ToSlash(relCheck), "/")
	if len(parts) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no path")
		return
	}
	top := parts[0]
	allowedTop := map[string]bool{"summary": true, "intro": true, "explain": true, "practice": true, "extend": true}
	if !allowedTop[top] {
		httpx.Error(w, http.StatusForbidden, "directory not readable")
		return
	}
	if top == "practice" && len(parts) == 2 && parts[1] == "answer-key.json" {
		httpx.Error(w, http.StatusForbidden, "answer key is private")
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", contentTypeFor(filepath.Base(abs)))
	w.Write(data)
}

// handleWriteFile is the POST counterpart of handleReadFile.
// Write whitelist: summary/<file>, explain/notes.md, intro/survey.json,
// extend/flower.json. The retired project-memory path is intentionally absent.
func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/files/projects/"+slug+"/")
	if rel == "" || strings.Contains(rel, "..") || strings.ContainsAny(rel, "\\") {
		httpx.Error(w, http.StatusBadRequest, "invalid path")
		return
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	relCheck, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		httpx.Error(w, http.StatusBadRequest, "path escapes project")
		return
	}
	parts := strings.Split(filepath.ToSlash(relCheck), "/")
	if len(parts) != 2 {
		httpx.Error(w, http.StatusForbidden, "POST writes only allowed for <dir>/<filename>")
		return
	}
	dir, file := parts[0], parts[1]
	allowed := dir == "summary"
	// explain/ is read-only except for explain/notes.md (learner-owned notes).
	if dir == "explain" && file == "notes.md" {
		allowed = true
	}
	if dir == "intro" && file == "survey.json" {
		allowed = true
	}
	if dir == "extend" && file == "flower.json" {
		allowed = true
	}
	if !allowed {
		httpx.Error(w, http.StatusForbidden, "directory not writable")
		return
	}
	if !validEditableFilename.MatchString(file) {
		httpx.Error(w, http.StatusBadRequest, "invalid filename")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body) > 1<<20 {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "file too large (>1MB)")
		return
	}
	if err := workspace.AtomicWriteFile(abs, body, 0o644); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dir == "extend" && file == "flower.json" {
		digest := sha256.Sum256(body)
		_, _, err := awardLearningEvent(slug, progressstore.ProgressEvent{
			ID:            "extend-flower:" + hex.EncodeToString(digest[:12]),
			SourceType:    "extend-flower",
			SourceID:      rel,
			ActivityDelta: 1,
			Title:         "编辑知识花朵",
			Detail:        "补充或调整花瓣内容",
		})
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "record flower activity: "+err.Error())
			return
		}
	}
	s.cache.Invalidate(slug)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "bytes": len(body)})
}

// ensure workspace is imported even if no direct usage
var _ = workspace.ZoneIntro
