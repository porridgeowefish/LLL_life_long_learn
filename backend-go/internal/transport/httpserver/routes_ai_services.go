package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/askaiconfig"
	"github.com/xmz14/lll/backend-go/internal/askaiprovider"
	"github.com/xmz14/lll/backend-go/internal/httpx"
)

type aiServiceBinding struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
}

type aiServiceProvider struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name,omitempty"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
}

type aiServicesResource struct {
	Providers []aiServiceProvider         `json:"providers"`
	Bindings  map[string]aiServiceBinding `json:"bindings"`
}

func (s *Server) handleGetAIServices(w http.ResponseWriter, r *http.Request) {
	cfg, err := askaiconfig.Load()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "unable to load AI services")
		return
	}
	resource := aiServicesResource{Bindings: map[string]aiServiceBinding{}}
	if cfg != nil {
		for _, provider := range cfg.Providers {
			kind := provider.Kind
			if kind == "openai" {
				kind = "openai-compatible"
			}
			key := ""
			if provider.APIKey != "" {
				key = maskedKey
			}
			resource.Providers = append(resource.Providers, aiServiceProvider{ID: provider.ID, Kind: kind, Name: provider.Name, BaseURL: provider.BaseURL, APIKey: key})
		}
		for _, name := range []string{"teacher", "annotationAskAI", "conversationCompaction"} {
			if binding, ok := cfg.Bindings[name]; ok {
				resource.Bindings[name] = aiServiceBinding{ProviderID: binding.ProviderID, Model: binding.Model}
				continue
			}
			if provider := cfg.Find(""); provider != nil {
				resource.Bindings[name] = aiServiceBinding{ProviderID: provider.ID, Model: provider.Model}
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, resource)
}

func (s *Server) handlePutAIServices(w http.ResponseWriter, r *http.Request) {
	var in aiServicesResource
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid AI services body")
		return
	}
	old, _ := askaiconfig.Load()
	oldByID := map[string]askaiconfig.Provider{}
	if old != nil {
		for _, provider := range old.Providers {
			oldByID[provider.ID] = provider
		}
	}
	teacher := in.Bindings["teacher"]
	providers := make([]askaiconfig.Provider, 0, len(in.Providers))
	for _, provider := range in.Providers {
		kind := provider.Kind
		if kind == "openai-compatible" {
			kind = "openai"
		}
		model := oldByID[provider.ID].Model
		if model == "" {
			for _, binding := range in.Bindings {
				if binding.ProviderID == provider.ID && binding.Model != "" {
					model = binding.Model
					break
				}
			}
		}
		apiKey := provider.APIKey
		if apiKey == maskedKey {
			apiKey = oldByID[provider.ID].APIKey
		}
		providers = append(providers, askaiconfig.Provider{ID: strings.TrimSpace(provider.ID), Kind: kind, Name: provider.Name, BaseURL: strings.TrimSpace(provider.BaseURL), APIKey: strings.TrimSpace(apiKey), Model: strings.TrimSpace(model)})
	}
	bindings := map[string]askaiconfig.Binding{}
	for name, binding := range in.Bindings {
		bindings[name] = askaiconfig.Binding{ProviderID: binding.ProviderID, Model: binding.Model}
	}
	if err := askaiconfig.Save(askaiconfig.Config{Default: teacher.ProviderID, Providers: providers, Bindings: bindings}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "unable to save AI services")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProbeAIServices(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Binding    string             `json:"binding"`
		ProviderID string             `json:"providerId"`
		Inline     *aiServiceProvider `json:"inline"`
		Model      string             `json:"model"`
	}
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid probe request")
		return
	}
	var provider askaiconfig.Provider
	if in.Inline != nil {
		kind := in.Inline.Kind
		if kind == "openai-compatible" {
			kind = "openai"
		}
		provider = askaiconfig.Provider{ID: in.Inline.ID, Kind: kind, BaseURL: in.Inline.BaseURL, APIKey: in.Inline.APIKey, Model: in.Model}
	} else {
		cfg, err := askaiconfig.Load()
		if err != nil || cfg == nil || cfg.Find(in.ProviderID) == nil {
			httpx.Error(w, http.StatusBadRequest, "provider not found")
			return
		}
		provider = *cfg.Find(in.ProviderID)
		if in.Model != "" {
			provider.Model = in.Model
		}
	}
	started := time.Now()
	_, err := askaiprovider.Complete(r.Context(), askaiprovider.Provider{Kind: provider.Kind, BaseURL: provider.BaseURL, APIKey: provider.APIKey, Model: provider.Model}, "Reply with ok.", []askaiprovider.Message{{Role: "user", Content: "ping"}})
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "latencyMs": time.Since(started).Milliseconds(), "error": "provider probe failed"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "latencyMs": time.Since(started).Milliseconds(), "capabilities": map[string]bool{"streaming": true, "toolUse": true, "reasoningSummary": true}})
}
