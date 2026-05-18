package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"feed-puller/internal/downloader"
	"feed-puller/internal/rss"
	"feed-puller/internal/store"
)

const proxySettingKey = "proxy_url"

type Service struct {
	store *store.Store
	aria2 *downloader.Aria2Client
	log   *slog.Logger
}

func NewService(store *store.Store, aria2 *downloader.Aria2Client, log *slog.Logger) *Service {
	return &Service{store: store, aria2: aria2, log: log}
}

func (s *Service) PollSubscription(ctx context.Context, sub store.Subscription) error {
	proxyURL, err := s.store.GetSetting(ctx, proxySettingKey)
	if err != nil {
		return err
	}
	fetcher, err := rss.NewFetcher(proxyURL)
	if err != nil {
		return err
	}
	feed, err := fetcher.Fetch(ctx, sub.FeedURL, sub.UseProxy)
	if err != nil {
		_ = s.store.MarkSubscriptionFetched(ctx, sub.ID, err.Error())
		return err
	}
	inserted, err := s.store.SaveFeedItems(ctx, sub.ID, feed.Items)
	if err != nil {
		_ = s.store.MarkSubscriptionFetched(ctx, sub.ID, err.Error())
		return err
	}
	if err := s.store.MarkSubscriptionFetched(ctx, sub.ID, ""); err != nil {
		return err
	}
	s.log.Info("订阅拉取完成", "subscription_id", sub.ID, "items", len(feed.Items), "inserted", inserted)
	return nil
}

func (s *Service) SubmitPendingDownloads(ctx context.Context) error {
	pending, err := s.store.PendingDownloads(ctx, 50)
	if err != nil {
		return err
	}
	for _, item := range pending {
		if strings.TrimSpace(item.Dir) == "" {
			_ = s.store.RecordDownloadResult(ctx, item, "failed", "", "下载目录不能为空")
			continue
		}
		if err := s.store.MarkDownloadSubmitting(ctx, item.ItemID); err != nil {
			return err
		}
		gid, err := s.aria2.AddURI(ctx, item.URL, item.Dir)
		if err != nil {
			_ = s.store.RecordDownloadResult(ctx, item, "failed", "", err.Error())
			s.log.Warn("提交 aria2 失败", "item_id", item.ItemID, "error", err)
			continue
		}
		if err := s.store.RecordDownloadResult(ctx, item, "submitted", gid, ""); err != nil {
			return fmt.Errorf("记录下载任务失败: %w", err)
		}
		s.log.Info("下载任务已提交", "item_id", item.ItemID, "gid", gid)
	}
	return nil
}

func ProxySettingKey() string {
	return proxySettingKey
}
