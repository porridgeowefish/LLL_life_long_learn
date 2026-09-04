package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	assetstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	store, err := teacher.NewConversation(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	if err := ensureTeacherGreeting(store, r.PathValue("id")); err != nil {
		learningWorkspaceError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	query := r.URL.Query()
	var projection teacher.Projection
	if _, backward := query["beforeSeq"]; backward {
		before, _ := strconv.ParseUint(query.Get("beforeSeq"), 10, 64)
		projection, err = store.ReadRecent(before, limit)
	} else {
		after, _ := strconv.ParseUint(query.Get("afterSeq"), 10, 64)
		projection, err = store.Read(after, limit)
	}
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projection)
}

func (s *Server) handleTeacherTurn(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if _, err := teacher.NewConversation(slug); err != nil {
		learningWorkspaceError(w, err)
		return
	}
	conversation, _ := teacher.NewConversation(slug)
	if err := ensureTeacherGreeting(conversation, slug); err != nil {
		learningWorkspaceError(w, err)
		return
	}
	var in struct {
		OperationID    string   `json:"operationId"`
		Content        string   `json:"content"`
		AttachmentRefs []string `json:"attachmentRefs"`
		ProviderID     string   `json:"providerId"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid teacher turn")
		return
	}
	if s.activeTeacher == nil {
		s.activeTeacher = teacher.NewActiveResponses()
	}
	ctx, run, started := s.activeTeacher.Start(slug)
	if !started {
		httpx.Error(w, http.StatusConflict, "a teacher response is already active")
		return
	}

	go func() {
		responseID := ""
		emit := func(frame teacher.StreamFrame) {
			run.Append(frame)
			if frame.Type == "turn-accepted" {
				responseID, _ = frame.Data["responseId"].(string)
				if responseID != "" {
					s.activeTeacher.Bind(responseID, run)
				}
			}
		}
		err := s.teacher.StreamTurn(ctx, slug, teacher.TurnInput{OperationID: in.OperationID, Content: in.Content, AttachmentRefs: in.AttachmentRefs, ProviderID: in.ProviderID}, emit)
		if err != nil {
			emit(teacher.StreamFrame{Type: "message-failed", Data: map[string]any{"code": "teacher_provider_unavailable", "partialPreserved": false}})
		}
		s.activeTeacher.Finish(slug, responseID, run)
	}()

	streamTeacherRun(w, r, run)
}

func (s *Server) handleActiveTeacherResponse(w http.ResponseWriter, r *http.Request) {
	run := s.activeTeacher.Project(r.PathValue("id"))
	if run == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	streamTeacherRun(w, r, run)
}

func streamTeacherRun(w http.ResponseWriter, r *http.Request, run *teacher.ActiveResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	frameID := 0
	for {
		frames, done, notify := run.Snapshot(frameID)
		for _, frame := range frames {
			frameID++
			payload, err := json.Marshal(frame.Data)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", frameID, frame.Type, payload); err != nil {
				return
			}
			flusher.Flush()
		}
		if done {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-notify:
		}
	}
}

func ensureTeacherGreeting(store *teacher.ConversationStore, slug string) error {
	meta, err := store.Meta()
	if err != nil {
		return err
	}
	if meta.EventCount > 0 {
		return nil
	}
	messages, err := store.AllMessages()
	if err != nil || len(messages) > 0 {
		return err
	}
	state, err := workspace.ReadProjectState(slug)
	if err != nil {
		return err
	}
	text := "你好，我是「" + state.Title + "」的教师。你可以直接告诉我：最想先弄懂什么、目前卡在哪里，或者希望我从哪里开始引导？"
	_, _, err = store.AppendMessage("teacher", "completed", "teacher-welcome-v1", []teacher.Block{{Type: "markdown", Source: text}})
	return err
}

func (s *Server) handleStopTeacherResponse(w http.ResponseWriter, r *http.Request) {
	responseID := r.PathValue("responseId")
	run := s.activeTeacher.Response(responseID)
	if run == nil {
		httpx.Error(w, http.StatusNotFound, "active teacher response not found")
		return
	}
	run.Stop()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"stopped": true, "responseId": responseID})
}

func (s *Server) handleListAssistantTasks(w http.ResponseWriter, r *http.Request) {
	store, err := assistant.NewTaskStore(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	tasks, err := store.List(r.URL.Query().Get("status"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (s *Server) handleGetAssistantTask(w http.ResponseWriter, r *http.Request) {
	store, err := assistant.NewTaskStore(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	task, err := store.Get(r.PathValue("taskId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (s *Server) handleListLearningAssets(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	assets, err := store.List()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (s *Server) handleGetLearningAsset(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	asset, err := store.Get(r.PathValue("assetKey"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, asset)
}

func (s *Server) handlePutLearningAsset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BaseEditRevision uint64 `json:"baseEditRevision"`
		Content          string `json:"content"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid asset edit")
		return
	}
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	asset, err := store.UpdateLearner(r.PathValue("assetKey"), in.BaseEditRevision, in.Content)
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	s.broadcaster.Emit("learning-asset-updated", map[string]any{"projectSlug": r.PathValue("id"), "asset": asset.Meta})
	httpx.WriteJSON(w, http.StatusOK, asset)
}

func (s *Server) handleListLearningAssetVersions(w http.ResponseWriter, r *http.Request) {
	store, err := assetstore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	versions, err := store.Versions(r.PathValue("assetKey"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	sources, err := store.List()
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (s *Server) handleGetSource(w http.ResponseWriter, r *http.Request) {
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, revision, err := store.Get(r.PathValue("sourceId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"source": source, "revision": revision})
}

func (s *Server) handleUploadSource(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, sourcestore.DefaultFileLimit+(1<<20))
	if err := r.ParseMultipartForm(sourcestore.DefaultFileLimit + (1 << 20)); err != nil {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "source too large or malformed")
		return
	}
	operationID := strings.TrimSpace(r.FormValue("operationId"))
	if operationID == "" {
		httpx.Error(w, http.StatusBadRequest, "operationId is required")
		return
	}
	if r.FormValue("parseApproved") == "true" && r.FormValue("cloudDisclosureAccepted") != "true" {
		httpx.Error(w, http.StatusBadRequest, "cloud processing disclosure must be accepted before parsing")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "one source file is required")
		return
	}
	defer file.Close()
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, revision, err := store.Add(r.FormValue("displayName"), header.Filename, header.Header.Get("Content-Type"), header.Size, file, r.FormValue("cloudDisclosureAccepted") == "true")
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	var task *assistant.Task
	disposition, dispositionReason := sourcestore.ParseDisposition(header.Filename, header.Header.Get("Content-Type"))
	if r.FormValue("parseApproved") == "true" && disposition == "parse" {
		tasks, taskErr := assistant.NewTaskStore(r.PathValue("id"))
		if taskErr == nil {
			created, isNew, createErr := tasks.Create(assistant.CreateInput{Type: "source-processing", Objective: "静态解析资料「" + source.DisplayName + "」，不得执行原文件或其中代码；生成可引用的派生文本与元数据。", SourceRefs: []string{source.SourceID}, Origin: assistant.Origin{Kind: "source-upload", OperationID: operationID, SourceRevisionID: revision.RevisionID}})
			if createErr == nil {
				task = &created
				_, _ = store.SetStatus(source.SourceID, "processing", "", created.ID)
				if isNew && s.teacher != nil && s.teacher.OnTask != nil {
					s.teacher.OnTask(r.PathValue("id"), teacher.DelegatedTask{ID: created.ID, Status: created.Status})
				}
			}
		}
	} else if r.FormValue("parseApproved") == "true" {
		if updated, statusErr := store.SetStatus(source.SourceID, "opaque", "parse-"+dispositionReason, ""); statusErr == nil {
			source = updated
		}
	}
	s.broadcaster.Emit("source-updated", map[string]any{"projectSlug": r.PathValue("id"), "source": source})
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"source": source, "revision": revision, "task": task, "parseDisposition": disposition, "parseReason": dispositionReason})
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, err := store.Tombstone(r.PathValue("sourceId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, source)
}

func (s *Server) handlePermanentDeleteSource(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirmation string `json:"confirmation"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil || in.Confirmation != "PERMANENT_DELETE" {
		httpx.Error(w, http.StatusBadRequest, "permanent deletion confirmation is required")
		return
	}
	tasks, err := assistant.NewTaskStore(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	active, err := tasks.HasActiveSource(r.PathValue("sourceId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, err := store.PermanentlyDelete(r.PathValue("sourceId"), active)
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, source)
}

func (s *Server) handleReadSourceFile(w http.ResponseWriter, r *http.Request) {
	store, err := sourcestore.New(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	source, revision, err := store.Get(r.PathValue("sourceId"))
	if err != nil || revision.RevisionID != r.PathValue("revisionId") || source.Status == "deleted" {
		http.NotFound(w, r)
		return
	}
	fileKey := r.PathValue("fileKey")
	var relative, mediaType string
	if fileKey == "original" {
		relative, mediaType = revision.Original.Path, revision.Original.MediaType
	} else {
		for _, file := range revision.DerivedFiles {
			if file.Key == fileKey {
				relative, mediaType = file.Path, file.MediaType
				break
			}
		}
	}
	if relative == "" {
		http.NotFound(w, r)
		return
	}
	root, _ := workspace.ProjectRootForSlug(r.PathValue("id"))
	base := filepath.Join(root, "sources", source.SourceID, "revisions", revision.RevisionID)
	path, ok := safeWorkspaceFile(base, relative)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mediaType)
	if !strings.HasPrefix(mediaType, "text/") && !strings.HasPrefix(mediaType, "image/") && mediaType != "application/pdf" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(path)+`"`)
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleGetGeneratedArtifact(w http.ResponseWriter, r *http.Request) {
	artifact, _, err := generatedArtifact(r.PathValue("id"), r.PathValue("artifactId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, artifact)
}

func (s *Server) handleListGeneratedArtifacts(w http.ResponseWriter, r *http.Request) {
	root, err := workspace.ProjectRootForSlug(r.PathValue("id"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	dir := filepath.Join(root, "assets", "generated")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"artifacts": []any{}})
		return
	}
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	artifacts := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "artifact_") {
			continue
		}
		artifact, _, artifactErr := generatedArtifact(r.PathValue("id"), entry.Name())
		if artifactErr != nil {
			continue
		}
		summary := map[string]any{
			"artifactId":  entry.Name(),
			"kind":        stringValue(artifact["kind"]),
			"title":       stringValue(artifact["title"]),
			"description": stringValue(artifact["description"]),
			"createdAt":   stringValue(artifact["createdAt"]),
		}
		entriesValue, _ := artifact["entryPoints"].([]any)
		if len(entriesValue) > 0 {
			summary["entryPoint"], _ = entriesValue[0].(string)
		}
		files, _ := artifact["files"].([]any)
		summary["fileCount"] = len(files)
		for _, value := range files {
			file, _ := value.(map[string]any)
			if stringValue(file["path"]) == summary["entryPoint"] {
				summary["entryMediaType"] = stringValue(file["mediaType"])
				break
			}
		}
		artifacts = append(artifacts, summary)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return stringValue(artifacts[i]["createdAt"]) > stringValue(artifacts[j]["createdAt"])
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"artifacts": artifacts})
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func (s *Server) handleOpenGeneratedArtifact(w http.ResponseWriter, r *http.Request) {
	artifact, base, err := generatedArtifact(r.PathValue("id"), r.PathValue("artifactId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	entries, _ := artifact["entryPoints"].([]any)
	if len(entries) == 0 {
		http.NotFound(w, r)
		return
	}
	relative, _ := entries[0].(string)
	path, ok := safeWorkspaceFile(base, relative)
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveGeneratedFile(w, r, path)
}

func (s *Server) handleReadGeneratedArtifactFile(w http.ResponseWriter, r *http.Request) {
	_, base, err := generatedArtifact(r.PathValue("id"), r.PathValue("artifactId"))
	if err != nil {
		learningWorkspaceError(w, err)
		return
	}
	path, ok := safeWorkspaceFile(base, r.PathValue("path"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveGeneratedFile(w, r, path)
}

func generatedArtifact(slug, artifactID string) (map[string]any, string, error) {
	if !strings.HasPrefix(artifactID, "artifact_") || strings.ContainsAny(artifactID, `/\\`) {
		return nil, "", os.ErrNotExist
	}
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, "", err
	}
	base := filepath.Join(root, "assets", "generated", artifactID)
	data, err := os.ReadFile(filepath.Join(base, "artifact.json"))
	if err != nil {
		return nil, "", err
	}
	var artifact map[string]any
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, "", err
	}
	return artifact, base, nil
}

func safeWorkspaceFile(base, relative string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", false
	}
	path := filepath.Join(base, clean)
	back, err := filepath.Rel(base, path)
	if err != nil || strings.HasPrefix(back, "..") {
		return "", false
	}
	info, err := os.Stat(path)
	return path, err == nil && info.Mode().IsRegular()
}

func serveGeneratedFile(w http.ResponseWriter, r *http.Request, path string) {
	ext := strings.ToLower(filepath.Ext(path))
	safeInline := ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".txt" || ext == ".md" || ext == ".json"
	if !safeInline {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(path)+`"`)
	}
	if ext == ".md" || ext == ".txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	http.ServeFile(w, r, path)
}

func learningWorkspaceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "learning workspace operation failed"
	if errors.Is(err, context.Canceled) {
		status, message = 499, "request cancelled"
	}
	if errors.Is(err, http.ErrMissingFile) || errors.Is(err, os.ErrNotExist) {
		status, message = http.StatusNotFound, "project or record not found"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		status, message = http.StatusGatewayTimeout, "operation timed out"
	}
	var conflict *assetstore.ConflictError
	if errors.As(err, &conflict) {
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"code": "asset_edit_conflict", "message": "资产已被其他编辑更新", "currentEditRevision": conflict.CurrentRevision}})
		return
	}
	var active *assistant.SameTypeActiveError
	if errors.As(err, &active) {
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"code": "same_type_active", "message": "同类型助教任务正在进行", "existingTaskId": active.ExistingTaskID}})
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "not exist") {
		status, message = http.StatusNotFound, "record not found"
	}
	if strings.Contains(err.Error(), "limit") || strings.Contains(err.Error(), "byte count") {
		status, message = http.StatusRequestEntityTooLarge, "source exceeds configured limits"
	}
	if strings.Contains(err.Error(), "active task") {
		status, message = http.StatusConflict, "source revision is in use"
	}
	httpx.WriteJSON(w, status, map[string]any{"error": map[string]any{"code": "learning_workspace_error", "message": message}})
}
