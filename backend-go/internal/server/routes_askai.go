package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/httpx"
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
