package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"feed-puller/internal/store"
)

func (s *Server) handleFeishuNotifySetting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.store.GetFeishuNotifyConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var input store.FeishuNotifyConfig
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		cfg := input
		cfg.NotifyType = strings.TrimSpace(cfg.NotifyType)
		cfg.Webhook = strings.TrimSpace(cfg.Webhook)
		cfg.ReceiveOpenID = strings.TrimSpace(cfg.ReceiveOpenID)
		cfg.ReceiveTargets = strings.TrimSpace(cfg.ReceiveTargets)
		cfg.CompleteTitleTemplate = strings.TrimSpace(cfg.CompleteTitleTemplate)
		cfg.FailTitleTemplate = strings.TrimSpace(cfg.FailTitleTemplate)
		if cfg.NotifyType == "" && cfg.Webhook != "" {
			cfg.NotifyType = "webhook"
		}
		if cfg.NotifyType == "api" && cfg.ReceiveOpenID == "" && cfg.ReceiveTargets == "" {
			if user, ok := s.currentUser(r); ok && strings.TrimSpace(user.FeishuOpenID) != "" {
				cfg.ReceiveOpenID = strings.TrimSpace(user.FeishuOpenID)
			}
		}
		if err := s.store.SaveFeishuNotifyConfig(r.Context(), cfg); err != nil {
			if store.IsInvalidFeishuNotifyType(err) || store.IsInvalidFeishuBatchWindow(err) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		saved, err := s.store.GetFeishuNotifyConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleFeishuNotifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	cfg, err := s.store.GetFeishuNotifyConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg.NotifyType == "api" && strings.TrimSpace(cfg.ReceiveOpenID) == "" && strings.TrimSpace(cfg.ReceiveTargets) == "" {
		if user, ok := s.currentUser(r); ok && strings.TrimSpace(user.FeishuOpenID) != "" {
			cfg.ReceiveOpenID = strings.TrimSpace(user.FeishuOpenID)
		}
	}
	if err := s.service.SendFeishuTestNotification(r.Context(), cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "测试消息已发送"})
}

func (s *Server) handleFeishuNotifyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	params := parsePageParams(r)
	rows, total, err := s.store.ListFeishuNotifyHistoryPage(r.Context(), params.Page, params.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []store.FeishuNotifyHistory{}
	}
	writePaginatedJSON(w, http.StatusOK, rows, total, params.Page, params.PageSize)
}
