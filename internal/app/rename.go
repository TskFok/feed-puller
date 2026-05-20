package app

import (
	"context"
	"path/filepath"
	"strings"

	"feed-puller/internal/aiclient"
	"feed-puller/internal/downloader"
	"feed-puller/internal/rename"
	"feed-puller/internal/store"
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
	s.maybeRenameDownloadFileAt(ctx, sub, itemTitle, filePath)
}

// maybeRenameDownloadFileAt 在已知文件路径时执行 AI 重命名，供 aria2 hook 直接复用。
func (s *Service) maybeRenameDownloadFileAt(ctx context.Context, sub store.Subscription, itemTitle, filePath string) {
	if !sub.AIRenameEnabled {
		return
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		s.log.Warn("下载文件路径为空，跳过重命名", "subscription_id", sub.ID)
		return
	}
	configs, err := s.store.ListAIConfigs(ctx)
	if err != nil {
		s.log.Warn("读取 AI 配置失败，跳过重命名", "subscription_id", sub.ID, "error", err)
		return
	}
	if len(configs) == 0 {
		s.log.Warn("未配置 AI 服务，跳过重命名", "subscription_id", sub.ID)
		return
	}
	cfg := configs[0]

	filename := filepath.Base(filePath)
	detected, err := aiclient.ExtractEpisode(ctx, cfg.BaseURL, cfg.APIKey, cfg.Model, filename, itemTitle)
	if err != nil {
		if local, ok := rename.DetectEpisodeLocally(filename, itemTitle); ok {
			detected = local
			s.log.Info("AI 识别集数失败，使用本地规则", "subscription_id", sub.ID, "episode", detected, "error", err)
		} else {
			s.log.Warn("识别集数失败，跳过重命名", "subscription_id", sub.ID, "file", filePath, "error", err)
			return
		}
	}
	finalEpisode, err := rename.FinalEpisode(detected, sub.AIRenameEpOffset)
	if err != nil {
		s.log.Warn("计算最终集数失败，跳过重命名", "subscription_id", sub.ID, "detected", detected, "offset", sub.AIRenameEpOffset, "error", err)
		return
	}
	season := sub.AIRenameSeason
	if season < 1 {
		season = 1
	}
	targetPath := rename.BuildScrapeFilename(filePath, season, finalEpisode)
	if strings.TrimSpace(targetPath) == strings.TrimSpace(filePath) {
		return
	}
	if err := rename.RenameFile(filePath, targetPath); err != nil {
		s.log.Warn("重命名下载文件失败", "subscription_id", sub.ID, "from", filePath, "to", targetPath, "error", err)
		return
	}
	s.log.Info("下载文件已重命名", "subscription_id", sub.ID, "from", filePath, "to", targetPath)
}
