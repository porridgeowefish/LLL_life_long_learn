package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/confusionstore"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

const maskedKey = "••••"

func maskProviders(ps []askaiconfig.Provider) []askaiconfig.Provider {
	out := make([]askaiconfig.Provider, len(ps))
	for i, p := range ps {
		if p.APIKey != "" {
			p.APIKey = maskedKey
		}
		out[i] = p
	}
	return out
}

func (s *Server) handleGetAskAiSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := askaiconfig.Load()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg == nil {
		cfg = &askaiconfig.Config{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"default":      cfg.Default,
		"searchEngine": cfg.SearchEngine,
		"providers":    maskProviders(cfg.Providers),
	})
}

func (s *Server) handlePutAskAiSettings(w http.ResponseWriter, r *http.Request) {
	var in askaiconfig.Config
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	// Preserve real keys for providers the client echoed back masked.
	old, _ := askaiconfig.Load()
	oldByKey := map[string]askaiconfig.Provider{}
	if old != nil {
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
	if err := askaiconfig.Save(in); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProbeAskAi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProviderID string                `json:"providerId"`
		Inline     *askaiconfig.Provider `json:"inline"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	var pc askaiconfig.Provider
	if in.Inline != nil {
		pc = *in.Inline
	} else {
		cfg, err := askaiconfig.Load()
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
	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}
	_, err := askaiprovider.Complete(r.Context(), prov, "Reply with the single word: ok", []askaiprovider.Message{{Role: "user", Content: "ping"}})
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
	store, err := confusionstore.New(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conf, err := store.Get(cid)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "confusion not found")
		return
	}
	cfg, err := askaiconfig.Load()
	if err != nil || !cfg.Enabled() {
		httpx.Error(w, http.StatusBadRequest, "ask-ai not configured")
		return
	}
	pc := cfg.Find(in.ProviderID)
	if pc == nil {
		httpx.Error(w, http.StatusBadRequest, "provider not found")
		return
	}

	// Append the user turn immediately so it persists even if the stream aborts.
	if _, err := store.AppendAskMessage(cid, confusionstore.AskMessage{Role: "user", Content: in.Content}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build the message history (prior turns + this user turn).
	var msgs []askaiprovider.Message
	if conf.Ask != nil {
		for _, m := range conf.Ask.Messages {
			msgs = append(msgs, askaiprovider.Message{Role: m.Role, Content: m.Content})
		}
	}
	msgs = append(msgs, askaiprovider.Message{Role: "user", Content: in.Content})

	system := ""
	if in.PageArtifactID != "" {
		if ctx, err := loadPageContext(slug, in.PageArtifactID); err == nil {
			system = "You are helping a learner studying the following page. Answer in context.\n\n" + ctx
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}
	var sb strings.Builder
	writeFrame := func(f askaiprovider.Frame) {
		b, _ := json.Marshal(f)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		if f.Type == "text" {
			sb.WriteString(f.Content)
		}
	}
	if err := askaiprovider.Stream(r.Context(), prov, system, msgs, writeFrame); err != nil {
		writeFrame(askaiprovider.Frame{Type: "error", Content: err.Error()})
		return
	}
	// Persist the assistant reply (best-effort).
	_, _ = store.AppendAskMessage(cid, confusionstore.AskMessage{Role: "assistant", Content: sb.String()})
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
	store, err := confusionstore.New(slug)
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
	broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})

	go summarizeAskExchange(slug, cid, conf.QuoteSnapshot, conf.Ask)

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "pending"})
}

func summarizeAskExchange(slug, cid, quote string, ask *confusionstore.Ask) {
	cfg, err := askaiconfig.Load()
	if err != nil || !cfg.Enabled() {
		store, _ := confusionstore.New(slug)
		store.SetAskSummary(cid, "总结生成失败：未配置 Ask-AI 模型源。", "failed")
		broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
		return
	}
	pc := cfg.Find("")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	prov := askaiprovider.Provider{Kind: pc.Kind, BaseURL: pc.BaseURL, APIKey: pc.APIKey, Model: pc.Model, Reasoning: pc.Reasoning, Thinking: pc.Thinking}

	system := "结合用户的疑问点和下面的对话，生成一段不超过 250 个汉字的中文总结，帮助用户日后回忆这次答疑的结论。直接输出总结正文，不要寒暄或多余说明。"
	var msgs []askaiprovider.Message
	msgs = append(msgs, askaiprovider.Message{Role: "user", Content: "疑问原文：" + quote})
	if ask != nil {
		for _, m := range ask.Messages {
			msgs = append(msgs, askaiprovider.Message{Role: m.Role, Content: m.Content})
		}
	}
	summary, err := askaiprovider.Complete(ctx, prov, system, msgs)
	store, _ := confusionstore.New(slug)
	if err != nil || strings.TrimSpace(summary) == "" {
		store.SetAskSummary(cid, "总结生成失败。", "failed")
	} else {
		store.SetAskSummary(cid, strings.TrimSpace(summary), "done")
	}
	broadcaster.Emit("confusion-updated", map[string]any{"projectSlug": slug, "action": "summarize", "id": cid})
}
