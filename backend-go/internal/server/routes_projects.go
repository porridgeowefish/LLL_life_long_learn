package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/projectindex"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// validMemoryFilename limits POST writes to memory/ to safe filenames only.
var validMemoryFilename = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)

// cache is the package-global project index. Populated on startup and on writes.
var cache = projectindex.New()

// createProjectRequest is the body of POST /api/projects.
// Why/Current/Target/Standard are optional background fields that seed project.md
// (background collection per LEARNING_PROJECT_STRUCTURE.md).
type createProjectRequest struct {
	Title           string `json:"title"`
	Slug            string `json:"slug,omitempty"`
	ParentProjectID string `json:"parentProjectId,omitempty"`
	Why             string `json:"why,omitempty"`
	Current         string `json:"current,omitempty"`
	Target          string `json:"target,omitempty"`
	Standard        string `json:"standard,omitempty"`
}

// handleListProjects returns the indexed project tree.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if err := cache.Rebuild(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "rebuild: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": cache.All()})
}

// handleCreateProject creates a top-level project.
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
	if err := workspace.CreateProjectSkeletonWithInput(slug, title, "", workspace.ProjectInput{
		Why:      req.Why,
		Current:  req.Current,
		Target:   req.Target,
		Standard: req.Standard,
	}); err != nil {
		if workspace.IsSlugConflict(err) {
			httpx.Error(w, http.StatusConflict, "project already exists: "+slug)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	cache.Invalidate(slug)
	state, _ := workspace.ReadProjectState(slug)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"project": state})
}

// handleCreateSubproject creates a child project under an existing parent.
func (s *Server) handleCreateSubproject(w http.ResponseWriter, r *http.Request) {
	parentSlug := r.PathValue("id")
	if !workspace.ValidateSlug(parentSlug) {
		httpx.Error(w, http.StatusBadRequest, "invalid parent slug")
		return
	}
	if exists, err := workspace.ProjectExists(parentSlug); err != nil || !exists {
		httpx.Error(w, http.StatusNotFound, "parent project not found")
		return
	}
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
	childSlug := strings.TrimSpace(req.Slug)
	if childSlug == "" {
		childSlug = workspace.Slugify(title)
	}
	if !workspace.ValidateSlug(childSlug) {
		httpx.Error(w, http.StatusBadRequest, "invalid child slug: "+childSlug)
		return
	}
	if err := workspace.CreateSubprojectWithInput(parentSlug, childSlug, title, workspace.ProjectInput{
		Why:      req.Why,
		Current:  req.Current,
		Target:   req.Target,
		Standard: req.Standard,
	}); err != nil {
		if workspace.IsSlugConflict(err) {
			httpx.Error(w, http.StatusConflict, "child project already exists: "+childSlug)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	cache.Invalidate(childSlug)
	cache.Invalidate(parentSlug)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"project": map[string]any{
			"slug":            childSlug,
			"title":           title,
			"parentProjectId": parentSlug,
		},
	})
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

// handleProjectTree returns a tree-shaped sidebar payload.
// This slice returns the same shape as GET /api/projects but grouped by parent.
func (s *Server) handleProjectTree(w http.ResponseWriter, r *http.Request) {
	if err := cache.Rebuild(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	flat := cache.All()
	type treeNode struct {
		workspace.ProjectMeta
		Subprojects []workspace.ProjectMeta `json:"subprojects,omitempty"`
	}
	byID := map[string]*treeNode{}
	var roots []*treeNode
	for _, p := range flat {
		node := &treeNode{ProjectMeta: p}
		byID[p.Slug] = node
	}
	for _, p := range flat {
		node := byID[p.Slug]
		if p.ParentProjectID == "" {
			roots = append(roots, node)
		} else if parent, ok := byID[p.ParentProjectID]; ok {
			parent.Subprojects = append(parent.Subprojects, p)
		}
	}
	out := make([]workspace.ProjectMeta, 0, len(roots))
	for _, r := range roots {
		out = append(out, r.ProjectMeta)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": out})
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
// Restricted to memory/, summary/, and zone folders. Used by frontend for
// reading project-memory.md and zone output.md as raw text.
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
	allowedTop := map[string]bool{"memory": true, "summary": true, "intro": true, "explain": true, "practice": true, "extend": true}
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// handleWriteFile is the POST counterpart of handleReadFile.
// Write whitelist: memory/<file>, summary/<file>, explain/notes.md.
// Learner-owned files — iter-03 expanded from memory-only to support the
// MarkdownEditor writing notes and summaries from the UI.
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
	allowed := dir == "memory" || dir == "summary"
	// explain/ is read-only except for explain/notes.md (learner-owned notes).
	if dir == "explain" && file == "notes.md" {
		allowed = true
	}
	if !allowed {
		httpx.Error(w, http.StatusForbidden, "directory not writable")
		return
	}
	if !validMemoryFilename.MatchString(file) {
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
	cache.Invalidate(slug)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "bytes": len(body)})
}

// ensure workspace is imported even if no direct usage
var _ = workspace.ZoneIntro
