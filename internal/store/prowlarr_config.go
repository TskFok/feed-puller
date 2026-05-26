package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	ProwlarrInternalFeedURLMovie = "prowlarr://movie"
	ProwlarrInternalFeedURLTV     = "prowlarr://tv"
	ProwlarrInternalFeedURLLegacy = "prowlarr://internal"
	ProwlarrInternalMovieName      = "Prowlarr 电影"
	ProwlarrInternalTVName         = "Prowlarr 剧集"
	settingProwlarrURL             = "prowlarr_url"
	settingProwlarrAPIKey          = "prowlarr_api_key"
	settingProwlarrDownloadDir     = "prowlarr_download_dir"
	settingProwlarrTVDownloadDir   = "prowlarr_tv_download_dir"
	settingProwlarrMovieSub        = "prowlarr_subscription_id"
	settingProwlarrTVSub           = "prowlarr_tv_subscription_id"
	settingProwlarrMovieRename     = "prowlarr_movie_rename_enabled"
	settingProwlarrTMDBAPIKey      = "prowlarr_tmdb_api_key"
	settingProwlarrIndexerIDs      = "prowlarr_indexer_ids"
)

// ProwlarrConfig 表示 Prowlarr 集成配置。
type ProwlarrConfig struct {
	URL                string  `json:"url"`
	APIKey             string  `json:"api_key"`
	DownloadDir        string  `json:"download_dir"`
	TVDownloadDir      string  `json:"tv_download_dir"`
	MovieRenameEnabled bool    `json:"movie_rename_enabled"`
	TMDBAPIKey         string  `json:"tmdb_api_key"`
	IndexerIDs         []int64 `json:"indexer_ids"`
	SubscriptionID     int64   `json:"subscription_id,omitempty"`
	TVSubscriptionID   int64   `json:"tv_subscription_id,omitempty"`
	Configured         bool    `json:"configured"`
}

func IsProwlarrInternalSubscription(sub Subscription) bool {
	feedURL := strings.TrimSpace(sub.FeedURL)
	return strings.HasPrefix(feedURL, "prowlarr://")
}

func IsProwlarrMovieSubscription(sub Subscription) bool {
	feedURL := strings.TrimSpace(sub.FeedURL)
	return feedURL == ProwlarrInternalFeedURLMovie || feedURL == ProwlarrInternalFeedURLLegacy
}

func IsProwlarrTVSubscription(sub Subscription) bool {
	return strings.TrimSpace(sub.FeedURL) == ProwlarrInternalFeedURLTV
}

func (s *Store) GetProwlarrConfig(ctx context.Context) (ProwlarrConfig, error) {
	urlVal, err := s.GetSetting(ctx, settingProwlarrURL)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	apiKey, err := s.GetSetting(ctx, settingProwlarrAPIKey)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	dir, err := s.GetSetting(ctx, settingProwlarrDownloadDir)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	tvDir, err := s.GetSetting(ctx, settingProwlarrTVDownloadDir)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	movieRenameRaw, err := s.GetSetting(ctx, settingProwlarrMovieRename)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	tmdbKey, err := s.GetSetting(ctx, settingProwlarrTMDBAPIKey)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	indexerRaw, err := s.GetSetting(ctx, settingProwlarrIndexerIDs)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	subIDRaw, err := s.GetSetting(ctx, settingProwlarrMovieSub)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	tvSubIDRaw, err := s.GetSetting(ctx, settingProwlarrTVSub)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	indexerIDs, err := ParseProwlarrIndexerIDs(indexerRaw)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	subID, _ := strconv.ParseInt(strings.TrimSpace(subIDRaw), 10, 64)
	tvSubID, _ := strconv.ParseInt(strings.TrimSpace(tvSubIDRaw), 10, 64)
	cfg := ProwlarrConfig{
		URL:                strings.TrimSpace(urlVal),
		APIKey:             strings.TrimSpace(apiKey),
		DownloadDir:        strings.TrimSpace(dir),
		TVDownloadDir:      strings.TrimSpace(tvDir),
		MovieRenameEnabled: strings.EqualFold(strings.TrimSpace(movieRenameRaw), "true") || movieRenameRaw == "1",
		TMDBAPIKey:         strings.TrimSpace(tmdbKey),
		IndexerIDs:         indexerIDs,
		SubscriptionID:     subID,
		TVSubscriptionID:   tvSubID,
	}
	cfg.Configured = cfg.URL != "" && cfg.APIKey != "" && cfg.DownloadDir != ""
	return cfg, nil
}

func (s *Store) SaveProwlarrConfig(ctx context.Context, cfg ProwlarrConfig) (ProwlarrConfig, error) {
	urlVal := strings.TrimSpace(cfg.URL)
	apiKey := strings.TrimSpace(cfg.APIKey)
	dir := strings.TrimSpace(cfg.DownloadDir)
	tvDir := strings.TrimSpace(cfg.TVDownloadDir)
	if urlVal == "" {
		return ProwlarrConfig{}, fmt.Errorf("Prowlarr 地址不能为空")
	}
	if apiKey == "" {
		return ProwlarrConfig{}, fmt.Errorf("Prowlarr API Key 不能为空")
	}
	if dir == "" {
		return ProwlarrConfig{}, fmt.Errorf("电影保存目录不能为空")
	}
	if tvDir == "" {
		tvDir = dir
	}
	movieSubID, err := s.ensureProwlarrSubscriptionByFeedURL(ctx, ProwlarrInternalFeedURLMovie, ProwlarrInternalMovieName, dir)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	tvSubID, err := s.ensureProwlarrSubscriptionByFeedURL(ctx, ProwlarrInternalFeedURLTV, ProwlarrInternalTVName, tvDir)
	if err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrURL, urlVal); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrAPIKey, apiKey); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrDownloadDir, dir); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrTVDownloadDir, tvDir); err != nil {
		return ProwlarrConfig{}, err
	}
	movieRename := "false"
	if cfg.MovieRenameEnabled {
		movieRename = "true"
	}
	if err := s.SetSetting(ctx, settingProwlarrMovieRename, movieRename); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrTMDBAPIKey, strings.TrimSpace(cfg.TMDBAPIKey)); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrIndexerIDs, EncodeProwlarrIndexerIDs(cfg.IndexerIDs)); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrMovieSub, strconv.FormatInt(movieSubID, 10)); err != nil {
		return ProwlarrConfig{}, err
	}
	if err := s.SetSetting(ctx, settingProwlarrTVSub, strconv.FormatInt(tvSubID, 10)); err != nil {
		return ProwlarrConfig{}, err
	}
	return s.GetProwlarrConfig(ctx)
}

// EnsureProwlarrSubscription 创建或更新电影 Prowlarr 订阅。
func (s *Store) EnsureProwlarrSubscription(ctx context.Context, downloadDir string) (int64, error) {
	return s.ensureProwlarrSubscriptionByFeedURL(ctx, ProwlarrInternalFeedURLMovie, ProwlarrInternalMovieName, downloadDir)
}

func (s *Store) EnsureProwlarrTVSubscription(ctx context.Context, downloadDir string) (int64, error) {
	return s.ensureProwlarrSubscriptionByFeedURL(ctx, ProwlarrInternalFeedURLTV, ProwlarrInternalTVName, downloadDir)
}

func (s *Store) ensureProwlarrSubscriptionByFeedURL(ctx context.Context, feedURL, name, downloadDir string) (int64, error) {
	downloadDir = strings.TrimSpace(downloadDir)
	if downloadDir == "" {
		return 0, fmt.Errorf("保存目录不能为空")
	}
	feedCandidates := []string{feedURL}
	if feedURL == ProwlarrInternalFeedURLMovie {
		feedCandidates = append(feedCandidates, ProwlarrInternalFeedURLLegacy)
	}

	for _, candidate := range feedCandidates {
		rows, err := s.db.QueryContext(ctx, `
			SELECT `+subscriptionColumns+`
			FROM subscriptions
			WHERE feed_url = ?
			ORDER BY id ASC
			LIMIT 1
		`, candidate)
		if err != nil {
			return 0, fmt.Errorf("查询 Prowlarr 订阅失败: %w", err)
		}
		subs, err := scanSubscriptions(rows)
		rows.Close()
		if err != nil {
			return 0, err
		}
		if len(subs) > 0 {
			sub := subs[0]
			if sub.DownloadDir != downloadDir || sub.FeedURL != feedURL || sub.Name != name {
				if _, err := s.updateProwlarrSubscription(ctx, sub.ID, name, feedURL, downloadDir); err != nil {
					return 0, err
				}
			}
			return sub.ID, nil
		}
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (
			name, feed_url, enabled, poll_interval_minutes, poll_cron, poll_cron_timezone,
			download_dir, include_keywords, exclude_keywords, use_proxy, rss_parser,
			ai_rename_enabled, ai_rename_season, ai_rename_episode_offset, sort_order
		) VALUES (?, ?, FALSE, 1440, '', 'UTC', ?, '', '', FALSE, 'generic', FALSE, 1, 0, 999999)
	`, name, feedURL, downloadDir)
	if err != nil {
		return 0, fmt.Errorf("创建 Prowlarr 订阅失败: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

func (s *Store) updateProwlarrSubscription(ctx context.Context, id int64, name, feedURL, downloadDir string) (Subscription, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET name = ?, feed_url = ?, enabled = FALSE, download_dir = ?, ai_rename_enabled = FALSE,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, feedURL, strings.TrimSpace(downloadDir), id)
	if err != nil {
		return Subscription{}, fmt.Errorf("更新 Prowlarr 订阅失败: %w", err)
	}
	return s.GetSubscription(ctx, id)
}

func (s *Store) ProwlarrSubscriptionID(ctx context.Context, mediaType string) (int64, error) {
	cfg, err := s.GetProwlarrConfig(ctx)
	if err != nil {
		return 0, err
	}
	if mediaType == ProwlarrMediaTV {
		if cfg.TVSubscriptionID > 0 {
			return cfg.TVSubscriptionID, nil
		}
		tvDir := cfg.TVDownloadDir
		if tvDir == "" {
			tvDir = cfg.DownloadDir
		}
		return s.EnsureProwlarrTVSubscription(ctx, tvDir)
	}
	if cfg.SubscriptionID > 0 {
		return cfg.SubscriptionID, nil
	}
	return s.EnsureProwlarrSubscription(ctx, cfg.DownloadDir)
}
