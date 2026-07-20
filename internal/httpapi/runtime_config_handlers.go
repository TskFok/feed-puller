package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"feed-puller/internal/store"
)

func (s *Server) handleRuntimeServiceConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.currentRuntimeServiceConfig())
	case http.MethodPut:
		var input store.RuntimeServiceConfig
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		input = normalizeRuntimeServiceConfig(input)
		if err := validateAria2RPCURL(input.Aria2RPCURL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.SaveRuntimeServiceConfig(r.Context(), input); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.replaceRuntimeServiceConfig(input)
		writeJSON(w, http.StatusOK, input)
	default:
		methodNotAllowed(w)
	}
}

func normalizeRuntimeServiceConfig(input store.RuntimeServiceConfig) store.RuntimeServiceConfig {
	input.Aria2RPCURL = strings.TrimSpace(input.Aria2RPCURL)
	input.Aria2RPCSecret = strings.TrimSpace(input.Aria2RPCSecret)
	input.FeishuAppID = strings.TrimSpace(input.FeishuAppID)
	input.FeishuAppSecret = strings.TrimSpace(input.FeishuAppSecret)
	input.Aria2HookSecret = strings.TrimSpace(input.Aria2HookSecret)
	return input
}

func validateAria2RPCURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("Aria2 RPC 地址必须是 HTTP 或 HTTPS URL")
	}
	return nil
}
