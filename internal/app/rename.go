package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"feed-puller/internal/aiclient"
	"feed-puller/internal/downloader"
	"feed-puller/internal/rename"
	"feed-puller/internal/store"
)

var (
	errEmptyRenamePath = errors.New("下载文件路径为空")
	errNoAIConfig      = errors.New("未配置 AI 服务")
)

// resolveDownloadFinalPath 在下载完成时执行可选重命名，返回应持久化的最终文件路径。
func (s *Service) resolveDownloadFinalPath(ctx context.Context, sub store.Subscription, item store.Item, filePath string, taskID int64) string {
	filePath = strings.TrimSpace(s.mapDownloadPath(filePath))
	if filePath == "" {
		return ""
	}
	if store.IsProwlarrInternalSubscription(sub) {
		return s.resolveProwlarrFinalPath(ctx, sub, item, filePath, taskID)
	}
	if !sub.AIRenameEnabled {
		return filePath
	}
	from, to, skipped, err := s.renameDownloadFileAt(ctx, sub, item.Title, filePath)
	if err != nil {
		s.log.Warn("重命名下载文件失败", "subscription_id", sub.ID, "file", filePath, "error", err)
		s.scheduleRenameRetry(ctx, taskID, from, err.Error())
		if from != "" {
			return from
		}
		return filePath
	}
	if skipped {
		return from
	}
	s.log.Info("下载文件已重命名", "subscription_id", sub.ID, "from", from, "to", to)
	return to
}

// maybeRenameDownloadFile 在下载完成后按订阅配置进行 AI 刮削重命名（aria2 tellStatus 路径）。
func (s *Service) maybeRenameDownloadFile(ctx context.Context, sub store.Subscription, itemTitle string, aria2Status map[string]any) {
	if !sub.AIRenameEnabled {
		return
	}
	filePath, err := downloader.Aria2DownloadPath(aria2Status)
	if err != nil {
		s.log.Warn("获取下载文件路径失败，跳过重命名", "subscription_id", sub.ID, "error", err)
		return
	}
	_ = s.resolveDownloadFinalPath(ctx, sub, store.Item{Title: itemTitle}, filePath, 0)
}

// renameDownloadFileAt 执行刮削重命名，返回原路径、目标路径及是否因已是目标格式而跳过。
func (s *Service) renameDownloadFileAt(ctx context.Context, sub store.Subscription, itemTitle, filePath string) (from, to string, skipped bool, err error) {
	from = strings.TrimSpace(s.mapDownloadPath(filePath))
	if from == "" {
		return "", "", false, errEmptyRenamePath
	}
	filename := filepath.Base(from)
	history := store.RenameHistory{
		SubscriptionID:   sub.ID,
		OriginalFilename: filename,
		OriginalPath:     from,
	}
	defer func() {
		s.recordRenameHistory(ctx, history, from, to, skipped, err)
	}()

	configs, err := s.store.ListAIConfigs(ctx)
	if err != nil {
		return from, "", false, err
	}
	if len(configs) == 0 {
		return from, "", false, errNoAIConfig
	}
	cfg := configs[0]

	var aiExtract *rename.AnimeExtract
	details, aiErr := aiclient.ExtractAnimeInfoDetailed(ctx, cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.RequestOptions, filename, itemTitle)
	if details != nil {
		history.AIPrompt = details.Prompt
		history.AIResponse = details.RawResponse
	}
	if aiErr == nil && details != nil && details.Info != nil {
		aiExtract = &rename.AnimeExtract{
			AnimeName: details.Info.AnimeName,
			Episode:   details.Info.Episode,
		}
	}
	localEpisode, localOK := rename.DetectEpisodeLocally(filename, itemTitle)
	if aiExtract == nil {
		if !localOK {
			return from, "", false, fmt.Errorf("识别番剧信息失败: %w", aiErr)
		}
		s.log.Info("AI 识别失败，使用本地规则", "subscription_id", sub.ID, "episode", localEpisode, "error", aiErr)
	}

	season := sub.AIRenameSeason
	if season < 1 {
		season = 1
	}
	scrape, err := rename.ResolveScrapeTarget(rename.ScrapeInput{
		FilePath:           from,
		Filename:           filename,
		Title:              itemTitle,
		SubscriptionSeason: season,
		EpisodeOffset:      sub.AIRenameEpOffset,
		AI:                 aiExtract,
		LocalEpisode:       localEpisode,
		LocalEpisodeOK:     localOK,
	})
	if err != nil {
		return from, "", false, err
	}
	targetPath := scrape.Path
	if strings.TrimSpace(targetPath) == strings.TrimSpace(from) {
		return from, from, true, nil
	}
	if err := rename.RenameFile(from, targetPath); err != nil {
		return from, "", false, err
	}
	return from, targetPath, false, nil
}

func (s *Service) recordRenameHistory(ctx context.Context, row store.RenameHistory, from, to string, skipped bool, renameErr error) {
	if strings.TrimSpace(row.OriginalFilename) == "" {
		row.OriginalFilename = filepath.Base(strings.TrimSpace(from))
	}
	if strings.TrimSpace(row.OriginalPath) == "" {
		row.OriginalPath = strings.TrimSpace(from)
	}
	switch {
	case renameErr != nil:
		row.Status = store.RenameHistoryStatusFailed
		row.Error = renameErr.Error()
	case skipped:
		row.Status = store.RenameHistoryStatusSkipped
		row.RenamedPath = strings.TrimSpace(from)
	default:
		row.Status = store.RenameHistoryStatusSuccess
		row.RenamedPath = strings.TrimSpace(to)
	}
	if err := s.store.CreateRenameHistory(ctx, row); err != nil {
		s.log.Warn("写入重命名历史失败", "subscription_id", row.SubscriptionID, "error", err)
	}
}
