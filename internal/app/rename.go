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
	_, _, _, err = s.renameDownloadFileAt(ctx, sub, itemTitle, filePath)
	if err != nil {
		s.log.Warn("重命名下载文件失败", "subscription_id", sub.ID, "file", filePath, "error", err)
	}
}

// maybeRenameDownloadFileAt 在已知文件路径时执行 AI 重命名，供 aria2 hook 直接复用。
func (s *Service) maybeRenameDownloadFileAt(ctx context.Context, sub store.Subscription, itemTitle, filePath string) {
	if !sub.AIRenameEnabled {
		return
	}
	from, to, skipped, err := s.renameDownloadFileAt(ctx, sub, itemTitle, filePath)
	if err != nil {
		s.log.Warn("重命名下载文件失败", "subscription_id", sub.ID, "file", filePath, "error", err)
		return
	}
	if skipped {
		return
	}
	s.log.Info("下载文件已重命名", "subscription_id", sub.ID, "from", from, "to", to)
}

// renameDownloadFileAt 执行刮削重命名，返回原路径、目标路径及是否因已是目标格式而跳过。
func (s *Service) renameDownloadFileAt(ctx context.Context, sub store.Subscription, itemTitle, filePath string) (from, to string, skipped bool, err error) {
	from = strings.TrimSpace(filePath)
	if from == "" {
		return "", "", false, errEmptyRenamePath
	}
	configs, err := s.store.ListAIConfigs(ctx)
	if err != nil {
		return from, "", false, err
	}
	if len(configs) == 0 {
		return from, "", false, errNoAIConfig
	}
	cfg := configs[0]

	filename := filepath.Base(from)
	detected, err := aiclient.ExtractEpisode(ctx, cfg.BaseURL, cfg.APIKey, cfg.Model, filename, itemTitle)
	if err != nil {
		if local, ok := rename.DetectEpisodeLocally(filename, itemTitle); ok {
			detected = local
			s.log.Info("AI 识别集数失败，使用本地规则", "subscription_id", sub.ID, "episode", detected, "error", err)
		} else {
			return from, "", false, fmt.Errorf("识别集数失败: %w", err)
		}
	}
	finalEpisode, err := rename.FinalEpisode(detected, sub.AIRenameEpOffset)
	if err != nil {
		return from, "", false, err
	}
	season := sub.AIRenameSeason
	if season < 1 {
		season = 1
	}
	targetPath := rename.BuildScrapeFilename(from, season, finalEpisode)
	if strings.TrimSpace(targetPath) == strings.TrimSpace(from) {
		return from, from, true, nil
	}
	if err := rename.RenameFile(from, targetPath); err != nil {
		return from, "", false, err
	}
	return from, targetPath, false, nil
}
