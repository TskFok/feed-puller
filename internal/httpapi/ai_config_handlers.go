package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"feed-puller/internal/aiclient"
	"feed-puller/internal/store"
)

func (s *Server) handleAIConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		params := parsePageParams(r)
		configs, total, err := s.store.ListAIConfigsPage(r.Context(), params.Page, params.PageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writePaginatedJSON(w, http.StatusOK, configs, total, params.Page, params.PageSize)
	case http.MethodPost:
		var input aiConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		cfg, err := s.store.CreateAIConfig(r.Context(), input.toAIConfig())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, cfg)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleAIConfigByID(w http.ResponseWriter, r *http.Request) {
	id, tail, ok := parseIDTail(r.URL.Path, "/api/ai-configs/")
	if !ok {
		writeError(w, http.StatusNotFound, "AI 配置不存在")
		return
	}
	if tail == "test" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		cfg, err := s.store.GetAIConfig(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err := aiclient.TestConnection(r.Context(), cfg.BaseURL, cfg.APIKey, cfg.Model); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "API 连通正常"})
		return
	}
	if tail != "" {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg, err := s.store.GetAIConfig(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var input aiConfigInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		cfg, err := s.store.UpdateAIConfig(r.Context(), id, input.toAIConfig())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodDelete:
		if err := s.store.DeleteAIConfig(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

type aiConfigInput struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Model  string `json:"model"`
	APIKey string `json:"api_key"`
}

func (input aiConfigInput) toAIConfig() store.AIConfig {
	return store.AIConfig{
		Name:    strings.TrimSpace(input.Name),
		BaseURL: strings.TrimSpace(input.URL),
		Model:   strings.TrimSpace(input.Model),
		APIKey:  strings.TrimSpace(input.APIKey),
	}
}
