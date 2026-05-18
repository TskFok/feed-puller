package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"feed-puller/internal/app"
	"feed-puller/internal/rss"
	"feed-puller/internal/store"
)

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		subscriptions, err := s.store.ListSubscriptions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, subscriptions)
	case http.MethodPost:
		var input subscriptionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		sub, err := s.store.CreateSubscription(r.Context(), input.toSubscription())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		go func() {
			if err := s.service.PollSubscription(contextWithoutCancel(r), sub); err != nil {
				s.log.Warn("新增订阅首次拉取失败", "subscription_id", sub.ID, "error", err)
			}
			if err := s.service.SubmitPendingDownloads(contextWithoutCancel(r)); err != nil {
				s.log.Warn("新增订阅提交下载失败", "subscription_id", sub.ID, "error", err)
			}
		}()
		writeJSON(w, http.StatusCreated, sub)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id, tail, ok := parseIDTail(r.URL.Path, "/api/subscriptions/")
	if !ok {
		writeError(w, http.StatusNotFound, "订阅不存在")
		return
	}
	if tail == "refresh" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		sub, err := s.store.GetSubscription(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "订阅不存在")
			return
		}
		if err := s.service.PollSubscription(r.Context(), sub); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if err := s.service.SubmitPendingDownloads(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if tail != "" {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}

	switch r.Method {
	case http.MethodGet:
		sub, err := s.store.GetSubscription(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "订阅不存在")
			return
		}
		writeJSON(w, http.StatusOK, sub)
	case http.MethodPut:
		var input subscriptionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		sub, err := s.store.UpdateSubscription(r.Context(), id, input.toSubscription())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sub)
	case http.MethodDelete:
		if err := s.store.DeleteSubscription(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	subID, _ := strconv.ParseInt(r.URL.Query().Get("subscription_id"), 10, 64)
	items, err := s.store.ListItems(r.Context(), subID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	tasks, err := s.store.ListDownloads(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleDownloadByID(w http.ResponseWriter, r *http.Request) {
	_, tail, ok := parseIDTail(r.URL.Path, "/api/downloads/")
	if !ok || tail != "retry" {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := s.service.SubmitPendingDownloads(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProxySetting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		value, err := s.store.GetSetting(r.Context(), app.ProxySettingKey())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"proxy_url": value})
	case http.MethodPut:
		var input struct {
			ProxyURL string `json:"proxy_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		if strings.TrimSpace(input.ProxyURL) != "" {
			if _, err := rssProxyURL(input.ProxyURL); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		if err := s.store.SetSetting(r.Context(), app.ProxySettingKey(), strings.TrimSpace(input.ProxyURL)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"proxy_url": strings.TrimSpace(input.ProxyURL)})
	default:
		methodNotAllowed(w)
	}
}

type subscriptionInput struct {
	Name                string `json:"name"`
	FeedURL             string `json:"feed_url"`
	Enabled             bool   `json:"enabled"`
	PollIntervalMinutes int    `json:"poll_interval_minutes"`
	DownloadDir         string `json:"download_dir"`
	UseProxy            bool   `json:"use_proxy"`
}

func (input subscriptionInput) toSubscription() store.Subscription {
	interval := input.PollIntervalMinutes
	if interval <= 0 {
		interval = 30
	}
	return store.Subscription{
		Name:                input.Name,
		FeedURL:             input.FeedURL,
		Enabled:             input.Enabled,
		PollIntervalMinutes: interval,
		DownloadDir:         input.DownloadDir,
		UseProxy:            input.UseProxy,
	}
}

func parseIDTail(path, prefix string) (int64, string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	if rest == path || rest == "" {
		return 0, "", false
	}
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	if len(parts) == 1 {
		return id, "", true
	}
	return id, strings.Join(parts[1:], "/"), true
}

func rssProxyURL(raw string) (string, error) {
	fetcher, err := rss.NewFetcher(raw)
	if err != nil {
		return "", err
	}
	_ = fetcher
	return strings.TrimSpace(raw), nil
}

func contextWithoutCancel(r *http.Request) context.Context {
	return context.WithoutCancel(r.Context())
}
