package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"feed-puller/internal/prowlarr"
	"feed-puller/internal/rename"
	"feed-puller/internal/rss"
	"feed-puller/internal/store"
	"feed-puller/internal/tmdb"
)

var (
	ErrProwlarrNotConfigured     = errors.New("Prowlarr 未配置")
	ErrProwlarrReleaseInProgress = errors.New("该资源正在下载中")
	ErrProwlarrReleaseCompleted  = errors.New("该资源已下载完成")
)

// ProwlarrSearchRequest 表示 Prowlarr 搜索请求。
type ProwlarrSearchRequest struct {
	Query               string
	Type                prowlarr.SearchType
	Sort                prowlarr.SortBy
	IndexerIDs          []int64
	IndexerIDsSpecified bool
	Limit               int
	Offset              int
}

// ProwlarrReleaseInput 表示从前端提交的 Prowlarr release 下载请求。
type ProwlarrReleaseInput struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	MediaType   string `json:"media_type"`
	DownloadURL string `json:"download_url"`
	InfoHash    string `json:"info_hash"`
	IndexerID   int64  `json:"indexer_id"`
	ImdbID      int64  `json:"imdb_id"`
	TmdbID      int64  `json:"tmdb_id"`
	TvdbID      int64  `json:"tvdb_id"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
}

func (s *Service) GetProwlarrConfig(ctx context.Context) (store.ProwlarrConfig, error) {
	return s.store.GetProwlarrConfig(ctx)
}

func (s *Service) SaveProwlarrConfig(ctx context.Context, cfg store.ProwlarrConfig) (store.ProwlarrConfig, error) {
	return s.store.SaveProwlarrConfig(ctx, cfg)
}

func (s *Service) TestProwlarrConnection(ctx context.Context, cfg store.ProwlarrConfig) error {
	urlVal := strings.TrimSpace(cfg.URL)
	apiKey := strings.TrimSpace(cfg.APIKey)
	client := prowlarr.NewClient(urlVal, apiKey)
	return client.TestConnection(ctx)
}

func (s *Service) ListProwlarrIndexers(ctx context.Context) ([]prowlarr.Indexer, error) {
	cfg, err := s.requireProwlarrConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := prowlarr.NewClient(cfg.URL, cfg.APIKey)
	indexers, err := client.ListIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Prowlarr 索引器失败: %w", err)
	}
	return prowlarr.FilterEnabledTorrentIndexers(indexers), nil
}

// ProwlarrReleaseFailure 单条 Prowlarr 批量下载失败原因。
type ProwlarrReleaseFailure struct {
	GUID  string
	Error string
}

const maxBatchProwlarrDownloads = 50
const maxProwlarrTorrentBytes = 4 << 20

type prowlarrTorrentFetchResult struct {
	Body        []byte
	FinalURL    string
	ContentType string
	MagnetURL   string
}

type prowlarrTorrentFetcher func(ctx context.Context, rawURL string, headers map[string]string) (prowlarrTorrentFetchResult, error)

var prowlarrTorrentHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req != nil && req.URL != nil && isMagnetURL(req.URL.String()) {
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	},
}

// ProwlarrSearchHistoryDetail 包含持久化的搜索结果。
type ProwlarrSearchHistoryDetail struct {
	store.ProwlarrSearchHistory
	Results []prowlarr.Release `json:"results"`
}

func (s *Service) SearchProwlarr(ctx context.Context, req ProwlarrSearchRequest) ([]prowlarr.Release, error) {
	cfg, err := s.requireProwlarrConfig(ctx)
	if err != nil {
		return nil, err
	}
	searchType := req.Type
	if searchType == "" {
		searchType = prowlarr.SearchTypeMovie
	}
	displayQuery := strings.TrimSpace(req.Query)
	normalized := prowlarr.NormalizeSearchQuery(displayQuery, searchType)
	if normalized == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	indexerIDs := req.IndexerIDs
	if len(indexerIDs) == 0 && !req.IndexerIDsSpecified {
		indexerIDs = cfg.IndexerIDs
	}
	sortBy := req.Sort
	if sortBy == "" {
		sortBy = prowlarr.SortBySeeders
	}
	client := prowlarr.NewClient(cfg.URL, cfg.APIKey)
	releases, err := client.Search(ctx, prowlarr.SearchInput{
		Query:      normalized,
		Type:       searchType,
		IndexerIDs: indexerIDs,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("Prowlarr 搜索失败: %w", err)
	}
	releases = prowlarr.FilterTorrentReleases(releases)
	prowlarr.SortReleases(releases, sortBy)
	resultsJSON := "[]"
	if raw, err := json.Marshal(releases); err == nil {
		resultsJSON = string(raw)
	}
	if recordErr := s.store.RecordProwlarrSearchHistory(ctx, store.ProwlarrSearchHistory{
		DisplayQuery: displayQuery,
		Query:        normalized,
		MediaType:    string(searchType),
		SortBy:       string(sortBy),
		IndexerIDs:   indexerIDs,
		ResultCount:  len(releases),
		ResultsJSON:  resultsJSON,
	}); recordErr != nil {
		s.log.Warn("保存 Prowlarr 搜索历史失败", "error", recordErr)
	}
	return releases, nil
}

// SearchProwlarrMovies 兼容旧调用。
func (s *Service) SearchProwlarrMovies(ctx context.Context, query string, limit, offset int) ([]prowlarr.Release, error) {
	return s.SearchProwlarr(ctx, ProwlarrSearchRequest{
		Query:  query,
		Type:   prowlarr.SearchTypeMovie,
		Sort:   prowlarr.SortBySeeders,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *Service) SubmitProwlarrRelease(ctx context.Context, input ProwlarrReleaseInput) (store.Item, error) {
	cfg, err := s.requireProwlarrConfig(ctx)
	if err != nil {
		return store.Item{}, err
	}
	guid := strings.TrimSpace(input.GUID)
	if guid == "" {
		return store.Item{}, fmt.Errorf("guid 不能为空")
	}
	mediaType := strings.TrimSpace(input.MediaType)
	if mediaType == "" {
		mediaType = store.ProwlarrMediaMovie
	}
	release := prowlarr.Release{
		GUID:        guid,
		Title:       strings.TrimSpace(input.Title),
		DownloadURL: strings.TrimSpace(input.DownloadURL),
		InfoHash:    strings.TrimSpace(input.InfoHash),
		IndexerID:   input.IndexerID,
		Protocol:    "torrent",
		ImdbID:      input.ImdbID,
		TmdbID:      input.TmdbID,
		TvdbID:      input.TvdbID,
		Season:      input.Season,
		Episode:     input.Episode,
	}
	downloadURL, err := s.resolveProwlarrDownloadURL(ctx, cfg, release)
	if err != nil {
		return store.Item{}, err
	}
	subID := cfg.SubscriptionID
	if mediaType == store.ProwlarrMediaTV {
		subID = cfg.TVSubscriptionID
	}
	if subID <= 0 {
		if mediaType == store.ProwlarrMediaTV {
			tvDir := cfg.TVDownloadDir
			if tvDir == "" {
				tvDir = cfg.DownloadDir
			}
			subID, err = s.store.EnsureProwlarrTVSubscription(ctx, tvDir)
		} else {
			subID, err = s.store.EnsureProwlarrSubscription(ctx, cfg.DownloadDir)
		}
		if err != nil {
			return store.Item{}, err
		}
	}
	meta := store.ProwlarrItemMeta{
		MediaType: mediaType,
		ImdbID:    input.ImdbID,
		TmdbID:    input.TmdbID,
		TvdbID:    input.TvdbID,
		Season:    input.Season,
		Episode:   input.Episode,
	}
	dedupeKey := "prowlarr:" + guid
	item, err := s.store.UpsertProwlarrItem(ctx, subID, release.Title, downloadURL, dedupeKey, guid, store.EncodeProwlarrItemMeta(meta))
	if err != nil {
		return store.Item{}, err
	}
	switch item.DownloadStatus {
	case "submitting", "submitted":
		return store.Item{}, ErrProwlarrReleaseInProgress
	case "completed":
		return store.Item{}, ErrProwlarrReleaseCompleted
	case "failed", "skipped":
		if err := s.store.ResetProwlarrItemForRetry(ctx, item.ID); err != nil {
			return store.Item{}, err
		}
	}
	if err := s.SubmitItemDownload(ctx, item.ID); err != nil {
		return store.Item{}, err
	}
	return s.store.GetItem(ctx, item.ID)
}

func (s *Service) ListProwlarrSearchHistory(ctx context.Context, limit int) ([]store.ProwlarrSearchHistory, error) {
	if _, err := s.requireProwlarrConfig(ctx); err != nil {
		return nil, err
	}
	return s.store.ListProwlarrSearchHistory(ctx, limit)
}

func (s *Service) GetProwlarrSearchHistory(ctx context.Context, id int64) (ProwlarrSearchHistoryDetail, error) {
	if _, err := s.requireProwlarrConfig(ctx); err != nil {
		return ProwlarrSearchHistoryDetail{}, err
	}
	entry, err := s.store.GetProwlarrSearchHistoryByID(ctx, id)
	if err != nil {
		return ProwlarrSearchHistoryDetail{}, err
	}
	results := make([]prowlarr.Release, 0)
	if raw := strings.TrimSpace(entry.ResultsJSON); raw != "" && raw != "null" {
		if err := json.Unmarshal([]byte(raw), &results); err != nil {
			return ProwlarrSearchHistoryDetail{}, fmt.Errorf("解析搜索历史结果失败: %w", err)
		}
	}
	if results == nil {
		results = []prowlarr.Release{}
	}
	return ProwlarrSearchHistoryDetail{
		ProwlarrSearchHistory: entry,
		Results:               results,
	}, nil
}

func (s *Service) DeleteProwlarrSearchHistory(ctx context.Context, id int64) error {
	if _, err := s.requireProwlarrConfig(ctx); err != nil {
		return err
	}
	return s.store.DeleteProwlarrSearchHistory(ctx, id)
}

func (s *Service) ClearProwlarrSearchHistory(ctx context.Context) error {
	if _, err := s.requireProwlarrConfig(ctx); err != nil {
		return err
	}
	return s.store.ClearProwlarrSearchHistory(ctx)
}

func (s *Service) ListProwlarrSubmittedGuids(ctx context.Context, guids []string) ([]string, error) {
	if _, err := s.requireProwlarrConfig(ctx); err != nil {
		return nil, err
	}
	return s.store.ListProwlarrSubmittedGuids(ctx, guids)
}

func (s *Service) SubmitProwlarrReleases(ctx context.Context, inputs []ProwlarrReleaseInput) ([]store.Item, []ProwlarrReleaseFailure) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if len(inputs) > maxBatchProwlarrDownloads {
		return nil, []ProwlarrReleaseFailure{{GUID: "", Error: fmt.Sprintf("单次最多提交 %d 条", maxBatchProwlarrDownloads)}}
	}
	items := make([]store.Item, 0, len(inputs))
	var failures []ProwlarrReleaseFailure
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		guid := strings.TrimSpace(input.GUID)
		if guid == "" {
			failures = append(failures, ProwlarrReleaseFailure{GUID: guid, Error: "guid 不能为空"})
			continue
		}
		if _, dup := seen[guid]; dup {
			continue
		}
		seen[guid] = struct{}{}
		item, err := s.SubmitProwlarrRelease(ctx, input)
		if err != nil {
			failures = append(failures, ProwlarrReleaseFailure{GUID: guid, Error: err.Error()})
			continue
		}
		items = append(items, item)
	}
	return items, failures
}

func (s *Service) resolveProwlarrDownloadURL(ctx context.Context, cfg store.ProwlarrConfig, release prowlarr.Release) (string, error) {
	downloadURL, err := prowlarr.ResolveTorrentURL(release)
	if err != nil {
		return "", err
	}
	if isMagnetURL(downloadURL) || !isHTTPURL(downloadURL) {
		return downloadURL, nil
	}
	fetcher := s.prowlarrTorrentFetcher
	if fetcher == nil {
		fetcher = fetchProwlarrTorrent
	}
	result, err := fetcher(ctx, downloadURL, prowlarrDownloadHeaders(downloadURL, cfg))
	if err != nil {
		return "", fmt.Errorf("解析 Prowlarr 下载地址失败: %w", err)
	}
	if isMagnetURL(result.MagnetURL) {
		return result.MagnetURL, nil
	}
	if isMagnetURL(result.FinalURL) {
		return result.FinalURL, nil
	}
	magnet, err := rss.MagnetFromTorrent(result.Body)
	if err != nil {
		if looksLikeProwlarrLoginPage(result) {
			return "", fmt.Errorf("Prowlarr 下载地址未返回有效 torrent，可能跳转到了登录页或反爬页面: %w", err)
		}
		return "", fmt.Errorf("Prowlarr 下载地址未返回有效 torrent: %w", err)
	}
	return magnet, nil
}

func fetchProwlarrTorrent(ctx context.Context, rawURL string, headers map[string]string) (prowlarrTorrentFetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return prowlarrTorrentFetchResult{}, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "feed-puller/1.0")
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := prowlarrTorrentHTTPClient.Do(req)
	if err != nil {
		return prowlarrTorrentFetchResult{}, fmt.Errorf("下载资源失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			if location := strings.TrimSpace(resp.Header.Get("Location")); isMagnetURL(location) {
				return prowlarrTorrentFetchResult{
					FinalURL:    location,
					ContentType: resp.Header.Get("Content-Type"),
					MagnetURL:   location,
				}, nil
			}
		}
		return prowlarrTorrentFetchResult{}, fmt.Errorf("下载资源失败: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProwlarrTorrentBytes))
	if err != nil {
		return prowlarrTorrentFetchResult{}, fmt.Errorf("读取资源失败: %w", err)
	}
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return prowlarrTorrentFetchResult{
		Body:        body,
		FinalURL:    finalURL,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func prowlarrDownloadHeaders(rawURL string, cfg store.ProwlarrConfig) map[string]string {
	headers := map[string]string{
		"Accept": "application/x-bittorrent, application/octet-stream, */*",
	}
	if referer := originReferer(rawURL); referer != "" {
		headers["Referer"] = referer
	}
	if sameURLHost(rawURL, cfg.URL) && strings.TrimSpace(cfg.APIKey) != "" {
		headers["X-Api-Key"] = strings.TrimSpace(cfg.APIKey)
	}
	return headers
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func isMagnetURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "magnet:")
}

func originReferer(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + "/"
}

func sameURLHost(left, right string) bool {
	leftURL, leftErr := url.Parse(strings.TrimSpace(left))
	rightURL, rightErr := url.Parse(strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return false
	}
	return strings.EqualFold(leftURL.Host, rightURL.Host) && leftURL.Host != ""
}

func looksLikeProwlarrLoginPage(result prowlarrTorrentFetchResult) bool {
	contentType := strings.ToLower(result.ContentType)
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		return true
	}
	if parsed, err := url.Parse(strings.TrimSpace(result.FinalURL)); err == nil {
		path := strings.ToLower(parsed.Path)
		if strings.Contains(path, "login") || strings.Contains(path, "signin") {
			return true
		}
	}
	body := strings.ToLower(strings.TrimSpace(string(result.Body[:min(len(result.Body), 512)])))
	return strings.HasPrefix(body, "<!doctype html") ||
		strings.HasPrefix(body, "<html") ||
		strings.Contains(body, "<title>login") ||
		strings.Contains(body, "please login")
}

func (s *Service) resolveProwlarrFinalPath(ctx context.Context, sub store.Subscription, item store.Item, filePath string, taskID int64) string {
	filePath = strings.TrimSpace(s.mapDownloadPath(filePath))
	if filePath == "" {
		return ""
	}
	finalPath, renameErr := s.renameProwlarrAt(ctx, sub, item, filePath)
	if renameErr != nil {
		s.log.Warn("Prowlarr 重命名失败", "subscription_id", sub.ID, "file", filePath, "error", renameErr)
		s.scheduleRenameRetry(ctx, taskID, filePath, renameErr.Error())
	}
	return finalPath
}

func (s *Service) renameProwlarrAt(ctx context.Context, sub store.Subscription, item store.Item, filePath string) (string, error) {
	filePath = strings.TrimSpace(s.mapDownloadPath(filePath))
	if filePath == "" {
		return "", fmt.Errorf("下载文件路径为空")
	}
	meta, ok := store.ParseProwlarrItemMeta(item.Link)
	if !ok {
		return filePath, nil
	}
	cfg, err := s.store.GetProwlarrConfig(ctx)
	if err != nil {
		return filePath, fmt.Errorf("读取 Prowlarr 配置失败: %w", err)
	}
	tmdbClient := tmdb.NewClient(cfg.TMDBAPIKey)
	switch meta.MediaType {
	case store.ProwlarrMediaTV:
		return s.renameProwlarrTV(ctx, tmdbClient, meta, item.Title, filePath)
	default:
		if !cfg.MovieRenameEnabled {
			return filePath, nil
		}
		return s.renameProwlarrMovie(ctx, tmdbClient, meta, item.Title, filePath)
	}
}

func (s *Service) renameProwlarrMovie(ctx context.Context, client *tmdb.Client, meta store.ProwlarrItemMeta, fallbackTitle, filePath string) (string, error) {
	title := strings.TrimSpace(fallbackTitle)
	year := 0
	if client.Enabled() {
		details, err := client.GetMovieDetails(ctx, meta.TmdbID, meta.ImdbID)
		if err == nil {
			if details.Title != "" {
				title = details.Title
			}
			year = details.Year
		} else {
			s.log.Warn("TMDB 查询电影信息失败，使用 release 标题", "error", err)
		}
	}
	target, err := rename.BuildMovieTargetPath(filePath, title, year)
	if err != nil {
		return filePath, fmt.Errorf("生成电影重命名路径失败: %w", err)
	}
	if err := rename.RenameFile(filePath, target); err != nil {
		return filePath, fmt.Errorf("电影重命名失败: %w", err)
	}
	s.log.Info("Prowlarr 电影已重命名", "from", filePath, "to", target)
	return target, nil
}

func (s *Service) renameProwlarrTV(ctx context.Context, client *tmdb.Client, meta store.ProwlarrItemMeta, fallbackTitle, filePath string) (string, error) {
	showTitle := strings.TrimSpace(fallbackTitle)
	if client.Enabled() {
		details, err := client.GetTVDetails(ctx, meta.TmdbID, meta.TvdbID)
		if err == nil && details.Name != "" {
			showTitle = details.Name
		} else if err != nil {
			s.log.Warn("TMDB 查询剧集信息失败，使用 release 标题", "error", err)
		}
	}
	season, episode := meta.Season, meta.Episode
	if season < 1 || episode < 1 {
		if parsedSeason, parsedEpisode, ok := parseSeasonEpisodeFromFilename(filepath.Base(filePath)); ok {
			season, episode = parsedSeason, parsedEpisode
		}
	}
	if season < 1 || episode < 1 {
		return filePath, nil
	}
	target, err := rename.BuildTVTargetPath(filePath, showTitle, season, episode)
	if err != nil {
		return filePath, fmt.Errorf("生成剧集重命名路径失败: %w", err)
	}
	if err := rename.RenameFile(filePath, target); err != nil {
		return filePath, fmt.Errorf("剧集重命名失败: %w", err)
	}
	s.log.Info("Prowlarr 剧集已重命名", "from", filePath, "to", target)
	return target, nil
}

func parseSeasonEpisodeFromFilename(name string) (season, episode int, ok bool) {
	name = strings.ToLower(name)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`s(\d{1,2})e(\d{1,3})`),
		regexp.MustCompile(`(\d{1,2})x(\d{1,3})`),
	}
	for _, re := range patterns {
		match := re.FindStringSubmatch(name)
		if len(match) == 3 {
			s, err1 := strconv.Atoi(match[1])
			e, err2 := strconv.Atoi(match[2])
			if err1 == nil && err2 == nil && s > 0 && e > 0 {
				return s, e, true
			}
		}
	}
	return 0, 0, false
}

func (s *Service) requireProwlarrConfig(ctx context.Context) (store.ProwlarrConfig, error) {
	cfg, err := s.store.GetProwlarrConfig(ctx)
	if err != nil {
		return store.ProwlarrConfig{}, err
	}
	if !cfg.Configured {
		return store.ProwlarrConfig{}, ErrProwlarrNotConfigured
	}
	return cfg, nil
}
