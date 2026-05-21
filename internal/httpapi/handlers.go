package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"feed-puller/internal/app"
	"feed-puller/internal/rss"
	"feed-puller/internal/store"
)

func (s *Server) handleSubscriptionNextPollPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input nextPollPreviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	sub := input.toPreviewSubscription()
	next, err := store.PreviewSubscriptionNextPoll(sub)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"next_poll_at": next.UTC().Format(time.RFC3339)})
}

func (s *Server) handleSubscriptionIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ids, err := s.store.ListSubscriptionIDs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
}

func (s *Server) handleSubscriptionReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w)
		return
	}
	var input struct {
		SubscriptionIDs []int64 `json:"subscription_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if err := s.store.ReorderSubscriptions(r.Context(), input.SubscriptionIDs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		params := parsePageParams(r)
		subscriptions, total, err := s.store.ListSubscriptionsPage(r.Context(), params.Page, params.PageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enrichSubscriptionsNextPoll(subscriptions)
		writePaginatedJSON(w, http.StatusOK, subscriptions, total, params.Page, params.PageSize)
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
		enrichSubscriptionNextPoll(&sub)
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
		items, err := s.service.PollSubscription(r.Context(), sub)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		proxyURL, _ := s.store.GetSetting(r.Context(), app.ProxySettingKey())
		fetcher, ferr := rss.NewFetcher(proxyURL)
		if ferr != nil {
			s.log.Warn("创建 HTTP 客户端失败，跳过文件大小探测", "error", ferr)
		}
		var payload []polledItemJSON
		if fetcher != nil {
			payload = buildPolledItemsJSON(r.Context(), fetcher, sub, items)
		} else {
			payload = itemsToPolledJSON(items)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": payload})
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
		enrichSubscriptionNextPoll(&sub)
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
		enrichSubscriptionNextPoll(&sub)
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

func (s *Server) handleItemsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	subID, _ := strconv.ParseInt(r.URL.Query().Get("subscription_id"), 10, 64)
	params := parsePageParams(r)
	items, total, err := s.store.ListItemsPage(r.Context(), subID, params.Page, params.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writePaginatedJSON(w, http.StatusOK, items, total, params.Page, params.PageSize)
}

func (s *Server) handleItemsBatchStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		ItemIDs        []int64 `json:"item_ids"`
		DownloadStatus string  `json:"download_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	items, err := s.store.BatchUpdateItemDownloadStatus(r.Context(), input.ItemIDs, input.DownloadStatus)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleItemsBatchDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		ItemIDs []int64 `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if len(input.ItemIDs) == 0 {
		writeError(w, http.StatusBadRequest, "请至少选择一条条目")
		return
	}
	items, failures := s.service.SubmitItemDownloads(r.Context(), input.ItemIDs)
	payload := map[string]any{"items": items}
	if len(failures) > 0 {
		out := make([]map[string]any, len(failures))
		for i, f := range failures {
			out[i] = map[string]any{"item_id": f.ItemID, "error": f.Error}
		}
		payload["failures"] = out
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleItemSubroutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/items/")
	if rest == "" {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "条目不存在")
		return
	}
	if len(parts) >= 2 && parts[1] == "download" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		if err := s.service.SubmitItemDownload(r.Context(), id); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := s.store.GetItem(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	writeError(w, http.StatusNotFound, "接口不存在")
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

func (s *Server) handleActiveDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	params := parsePageParams(r)
	rows, total, err := s.service.ListActiveDownloadsWithProgress(r.Context(), params.Page, params.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writePaginatedJSON(w, http.StatusOK, rows, total, params.Page, params.PageSize)
}

func (s *Server) handleCompletedDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	_ = s.service.SyncAria2DownloadStatus(r.Context())
	params := parsePageParams(r)
	rows, total, err := s.store.ListCompletedDownloadsPage(r.Context(), params.Page, params.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writePaginatedJSON(w, http.StatusOK, rows, total, params.Page, params.PageSize)
}

func (s *Server) handleDownloadByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/downloads/")
	if rest == "retry" {
		if err := s.service.SubmitPendingDownloads(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	id, tail, ok := parseIDTail(r.URL.Path, "/api/downloads/")
	if !ok {
		writeError(w, http.StatusNotFound, "接口不存在")
		return
	}
	switch tail {
	case "rename":
		if id <= 0 {
			writeError(w, http.StatusNotFound, "接口不存在")
			return
		}
		result, err := s.service.RetryCompletedDownloadRename(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusNotFound, "接口不存在")
	}
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
	PollCron            string `json:"poll_cron"`
	PollCronTimezone    string `json:"poll_cron_timezone"`
	DownloadDir         string `json:"download_dir"`
	IncludeKeywords     string `json:"include_keywords"`
	ExcludeKeywords     string `json:"exclude_keywords"`
	UseProxy            bool   `json:"use_proxy"`
	RSSParser           string `json:"rss_parser"`
	AIRenameEnabled     bool   `json:"ai_rename_enabled"`
	AIRenameSeason      int    `json:"ai_rename_season"`
	AIRenameEpOffset    int    `json:"ai_rename_episode_offset"`
}

type nextPollPreviewInput struct {
	Enabled             bool   `json:"enabled"`
	PollIntervalMinutes int    `json:"poll_interval_minutes"`
	PollCron            string `json:"poll_cron"`
	PollCronTimezone    string `json:"poll_cron_timezone"`
	LastFetchedAt       string `json:"last_fetched_at,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
}

func (input nextPollPreviewInput) toPreviewSubscription() store.Subscription {
	interval := input.PollIntervalMinutes
	if interval <= 0 {
		interval = 30
	}
	sub := store.Subscription{
		Enabled:             input.Enabled,
		PollIntervalMinutes: interval,
		PollCron:            strings.TrimSpace(input.PollCron),
		PollCronTimezone:    strings.TrimSpace(input.PollCronTimezone),
	}
	if t, ok := parseOptionalRFC3339(input.LastFetchedAt); ok {
		sub.LastFetchedAt = &t
	}
	if t, ok := parseOptionalRFC3339(input.CreatedAt); ok {
		sub.CreatedAt = t
	} else if sub.LastFetchedAt == nil {
		sub.CreatedAt = time.Now().UTC()
	}
	return sub
}

func parseOptionalRFC3339(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func (input subscriptionInput) toSubscription() store.Subscription {
	interval := input.PollIntervalMinutes
	if interval <= 0 {
		interval = 30
	}
	cronExpr := strings.TrimSpace(input.PollCron)
	return store.Subscription{
		Name:                input.Name,
		FeedURL:             input.FeedURL,
		Enabled:             input.Enabled,
		PollIntervalMinutes: interval,
		PollCron:            cronExpr,
		PollCronTimezone:    strings.TrimSpace(input.PollCronTimezone),
		DownloadDir:         input.DownloadDir,
		IncludeKeywords:     strings.TrimSpace(input.IncludeKeywords),
		ExcludeKeywords:     strings.TrimSpace(input.ExcludeKeywords),
		UseProxy:            input.UseProxy,
		RSSParser:           rss.NormalizeParser(input.RSSParser),
		AIRenameEnabled:     input.AIRenameEnabled,
		AIRenameSeason:      input.AIRenameSeason,
		AIRenameEpOffset:    input.AIRenameEpOffset,
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

type polledItemJSON struct {
	ID             int64      `json:"id"`
	SubscriptionID int64      `json:"subscription_id"`
	Title          string     `json:"title"`
	Link           string     `json:"link,omitempty"`
	DownloadURL    string     `json:"download_url,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	DownloadStatus string     `json:"download_status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ContentLength  *int64     `json:"content_length,omitempty"`
}

func itemToPolledJSON(it store.Item) polledItemJSON {
	return polledItemJSON{
		ID:             it.ID,
		SubscriptionID: it.SubscriptionID,
		Title:          it.Title,
		Link:           it.Link,
		DownloadURL:    it.DownloadURL,
		PublishedAt:    it.PublishedAt,
		DownloadStatus: it.DownloadStatus,
		CreatedAt:      it.CreatedAt,
		UpdatedAt:      it.UpdatedAt,
	}
}

func itemsToPolledJSON(items []store.Item) []polledItemJSON {
	out := make([]polledItemJSON, len(items))
	for i, it := range items {
		out[i] = itemToPolledJSON(it)
	}
	return out
}

func buildPolledItemsJSON(ctx context.Context, fetcher *rss.Fetcher, sub store.Subscription, items []store.Item) []polledItemJSON {
	out := make([]polledItemJSON, len(items))
	if len(items) == 0 {
		return out
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	var mu sync.Mutex
	for i := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, it store.Item) {
			defer wg.Done()
			defer func() { <-sem }()
			pj := itemToPolledJSON(it)
			if strings.TrimSpace(it.DownloadURL) != "" {
				if n, ok := fetcher.ProbeContentLength(ctx, it.DownloadURL, sub.UseProxy); ok {
					v := n
					pj.ContentLength = &v
				}
			}
			mu.Lock()
			out[i] = pj
			mu.Unlock()
		}(i, items[i])
	}
	wg.Wait()
	return out
}

func enrichSubscriptionNextPoll(sub *store.Subscription) {
	store.ApplySubscriptionNextPoll(sub, time.Now().UTC())
}

func enrichSubscriptionsNextPoll(subs []store.Subscription) {
	now := time.Now().UTC()
	for i := range subs {
		store.ApplySubscriptionNextPoll(&subs[i], now)
	}
}
