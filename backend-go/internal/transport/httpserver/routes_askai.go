package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	annotationstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"github.com/xmz14/lll/backend-go/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const maskedKey = "••••"

func maskProviders(ps []teacher.ProviderConfig) []teacher.ProviderConfig {
	out := make([]teacher.ProviderConfig, len(ps))
	for i, p := range ps {
		if p.APIKey != "" {
			p.APIKey = maskedKey
		}
		out[i] = p
	}
	return out
}

func (s *Server) handleGetAskAiSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := teacher.LoadConfig()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg == nil {
		cfg = &teacher.Config{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"default":      cfg.Default,
		"searchEngine": cfg.SearchEngine,
		"providers":    maskProviders(cfg.Providers),
		"bindings":     cfg.Bindings,
	})
}

func (s *Server) handlePutAskAiSettings(w http.ResponseWriter, r *http.Request) {
	var in teacher.Config
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	// Preserve real keys for providers the client echoed back masked.
	old, _ := teacher.LoadConfig()
	oldByKey := map[string]teacher.ProviderConfig{}
	if old != nil {
		if in.Bindings == nil {
			in.Bindings = old.Bindings
		}
		for _, p := range old.Providers {
			oldByKey[p.ID] = p
		}
	}
	for i, p := range in.Providers {
		if p.APIKey == maskedKey {
			if prev, ok := oldByKey[p.ID]; ok {
				in.Providers[i].APIKey = prev.APIKey
			}
		}
	}
	if err := teacher.SaveConfig(in); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProbeAskAi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProviderID string                  `json:"providerId"`
		Inline     *teacher.ProviderConfig `json:"inline"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	var pc teacher.ProviderConfig
	if in.Inline != nil {
		pc = *in.Inline
	} else {
		cfg, err := teacher.LoadConfig()
		if err != nil || cfg == nil {
			httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
			return
		}
		p := cfg.Find(in.ProviderID)
		if p == nil {
			httpx.Error(w, http.StatusBadRequest, "provider not found")
			return
		}
		pc = *p
	}
	prov := teacher.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}
	_, err := teacher.Complete(r.Context(), prov, "Reply with the single word: ok", []teacher.AIMessage{{Role: "user", Content: "ping"}})
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAskAiStream(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	cid := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	var in struct {
		ProviderID     string `json:"providerId"`
		Content        string `json:"content"`
		PageArtifactID string `json:"pageArtifactId"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conf, err := store.Get(cid)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	cfg, err := teacher.LoadConfig()
	if err != nil || !cfg.Enabled() {
		httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
		return
	}
	pc := cfg.Find(in.ProviderID)
	if in.ProviderID == "" {
		pc = cfg.Resolve("annotationAskAI")
	}
	if pc == nil {
		httpx.Error(w, http.StatusBadRequest, "provider not found")
		return
	}

	// Append the user turn immediately so it persists even if the stream aborts.
	updatedAnnotation, err := store.AppendMessage(cid, annotationstore.Message{Role: "learner", Content: in.Content})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	messageID := cid
	if updatedAnnotation.Ask != nil && len(updatedAnnotation.Ask.Messages) > 0 {
		messageID = updatedAnnotation.Ask.Messages[len(updatedAnnotation.Ask.Messages)-1].ID
	}
	_, _, _ = awardLearningEvent(slug, progressstore.Event{
		ID:         "ask-ai:" + messageID,
		SourceType: "ask-ai", SourceID: cid, ActivityDelta: 1,
		Title: "使用 Ask AI", Detail: strings.TrimSpace(conf.QuoteSnapshot),
	})

	// Build the message history (prior turns + this user turn).
	var msgs []teacher.AIMessage
	if conf.Ask != nil {
		for _, m := range conf.Ask.Messages {
			role := m.Role
			if role == "learner" {
				role = "user"
			}
			msgs = append(msgs, teacher.AIMessage{Role: role, Content: m.Content})
		}
	}
	msgs = append(msgs, teacher.AIMessage{Role: "user", Content: in.Content})

	system := "你是正文批注旁的轻量答疑助手。只回答当前问题，不使用教师的五角色流程，也不调用任何工具。以下内容是学习者明确选中并主动提问的引用；版本 ID 只用于说明引用来源。\n\n引用：" + conf.QuoteSnapshot + "\n正文版本：" + conf.AssetVersionID

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	prov := teacher.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: false, Thinking: false}
	var sb strings.Builder
	writeFrame := func(f teacher.AIFrame) {
		if f.Type == "thinking" {
			return
		}
		b, _ := json.Marshal(f)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		if f.Type == "text" {
			sb.WriteString(f.Content)
		}
	}
	if err := teacher.Stream(r.Context(), prov, system, msgs, writeFrame); err != nil {
		status := "failed"
		if errors.Is(err, context.Canceled) {
			status = "interrupted"
		}
		if sb.Len() > 0 {
			_, _ = store.AppendMessage(cid, annotationstore.Message{Role: "assistant", Content: sb.String(), Status: status})
			s.emitAnnotationUpdated(slug, "ask", cid)
		}
		writeFrame(teacher.AIFrame{Type: "error", Content: "回答暂时中断，已保留收到的部分内容。"})
		return
	}
	// Persist the assistant reply (best-effort).
	if sb.Len() > 0 {
		_, _ = store.AppendMessage(cid, annotationstore.Message{Role: "assistant", Content: sb.String(), Status: "completed"})
	}
	s.emitAnnotationUpdated(slug, "ask", cid)
}

// loadPageContext reads a project-relative artifact (e.g. "explain/pages/01.md")
// capped to keep prompts small.
func loadPageContext(slug, pageArtifactID string) (string, error) {
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pageArtifactID)))
	if err != nil {
		return "", err
	}
	s := string(data)
	const cap = 6000
	if len(s) > cap {
		s = s[:cap] + "\n…(truncated)"
	}
	return s, nil
}

func (s *Server) handleAskAiSummarize(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	cid := r.PathValue("confusionId")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}
	store, err := openAnnotationStore(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conf, err := store.Get(cid)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}

	// Mark pending immediately + notify (sidebar shows "生成总结中...").
	store.SetAskSummary(cid, "", "pending")
	s.broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})

	go s.summarizeAskExchange(slug, cid, conf.QuoteSnapshot, conf.Ask)

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "pending"})
}

func (s *Server) summarizeAskExchange(slug, cid, quote string, ask *annotationstore.Ask) {
	cfg, err := teacher.LoadConfig()
	if err != nil || !cfg.Enabled() {
		store, _ := openAnnotationStore(slug)
		store.SetAskSummary(cid, "总结生成失败：未配置 Ask-AI 模型源。", "failed")
		s.broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
		return
	}
	pc := cfg.Resolve("annotationAskAI")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	prov := teacher.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}

	system := "结合用户的疑问点和下面的对话，生成一段不超过 250 个汉字的中文总结，帮助用户日后回忆这次答疑的结论。直接输出总结正文，不要寒暄或多余说明。"
	var msgs []teacher.AIMessage
	msgs = append(msgs, teacher.AIMessage{Role: "user", Content: "疑问原文：" + quote})
	if ask != nil {
		for _, m := range ask.Messages {
			msgs = append(msgs, teacher.AIMessage{Role: m.Role, Content: m.Content})
		}
	}
	summary, err := teacher.Complete(ctx, prov, system, msgs)
	store, _ := openAnnotationStore(slug)
	if err != nil || strings.TrimSpace(summary) == "" {
		store.SetAskSummary(cid, "总结生成失败。", "failed")
	} else {
		store.SetAskSummary(cid, strings.TrimSpace(summary), "done")
	}
	s.broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
}
