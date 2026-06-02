package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"feed-puller/internal/downloader"
	"feed-puller/internal/paths"
	"feed-puller/internal/rss"
	"feed-puller/internal/store"
)

const proxySettingKey = "proxy_url"

type Service struct {
	store               *store.Store
	aria2               *downloader.Aria2Client
	log                 *slog.Logger
	pathMap             paths.Mapper
	feishuBot           feishuNotifySender
	feishuBatchMu       sync.Mutex
	feishuBatchComplete []feishuNotifyPayload
	feishuBatchFail     []feishuNotifyPayload
	feishuBatchTimer    *time.Timer
}

func NewService(store *store.Store, aria2 *downloader.Aria2Client, log *slog.Logger, pathMap ...paths.Mapper) *Service {
	s := &Service{store: store, aria2: aria2, log: log}
	if len(pathMap) > 0 {
		s.pathMap = pathMap[0]
	}
	return s
}

func (s *Service) mapDownloadPath(path string) string {
	return s.pathMap.Map(path)
}

func (s *Service) PollSubscription(ctx context.Context, sub store.Subscription) ([]store.Item, error) {
	proxyURL, err := s.store.GetSetting(ctx, proxySettingKey)
	if err != nil {
		return nil, err
	}
	fetcher, err := rss.NewFetcher(proxyURL)
	if err != nil {
		return nil, err
	}
	feed, err := fetcher.Fetch(ctx, sub.FeedURL, sub.UseProxy, sub.RSSParser)
	if err != nil {
		_ = s.store.MarkSubscriptionFetched(ctx, sub.ID, err.Error())
		return nil, err
	}
	items, err := rss.FilterFeedItems(feed.Items, sub.IncludeKeywords, sub.ExcludeKeywords)
	if err != nil {
		_ = s.store.MarkSubscriptionFetched(ctx, sub.ID, err.Error())
		return nil, err
	}
	saved, err := s.store.SaveFeedItems(ctx, sub.ID, items)
	if err != nil {
		_ = s.store.MarkSubscriptionFetched(ctx, sub.ID, err.Error())
		return nil, err
	}
	if err := s.store.MarkSubscriptionFetched(ctx, sub.ID, ""); err != nil {
		return nil, err
	}
	s.log.Info("订阅拉取完成", "subscription_id", sub.ID, "items", len(feed.Items), "after_filter", len(items), "persisted", len(saved))
	return saved, nil
}

// SubmitItemDownload 将单条条目提交给 aria2（原为批量队列中的一条）。
func (s *Service) SubmitItemDownload(ctx context.Context, itemID int64) error {
	item, err := s.store.GetItem(ctx, itemID)
	if err != nil {
		return err
	}
	sub, err := s.store.GetSubscription(ctx, item.SubscriptionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(item.DownloadURL) == "" {
		return fmt.Errorf("条目没有下载地址")
	}
	downloadURL := strings.TrimSpace(item.DownloadURL)
	if rss.NormalizeParser(sub.RSSParser) == rss.ParserMikan {
		proxyURL, err := s.store.GetSetting(ctx, proxySettingKey)
		if err != nil {
			return err
		}
		resolver, err := rss.NewFetcher(proxyURL)
		if err != nil {
			return err
		}
		resolved, err := rss.ResolveMikanDownloadURL(ctx, resolver, downloadURL, sub.UseProxy)
		if err != nil {
			return fmt.Errorf("解析 Mikan 下载地址失败: %w", err)
		}
		downloadURL = resolved
	}
	if !CanSubmitItemDownload(item.DownloadStatus) {
		if item.DownloadStatus == "submitting" {
			return fmt.Errorf("条目正在提交下载，请稍候")
		}
		return fmt.Errorf("条目当前不可下载")
	}
	pending := store.PendingDownload{
		ItemID:         item.ID,
		SubscriptionID: item.SubscriptionID,
		URL:            downloadURL,
		Dir:            sub.DownloadDir,
	}
	if strings.TrimSpace(pending.Dir) == "" {
		_ = s.store.RecordDownloadResult(ctx, pending, "failed", "", "下载目录不能为空")
		return fmt.Errorf("下载目录不能为空")
	}
	if err := s.store.MarkDownloadSubmitting(ctx, item.ID); err != nil {
		return err
	}
	gid, err := s.aria2.AddURI(ctx, pending.URL, pending.Dir)
	if err != nil {
		_ = s.store.RecordDownloadResult(ctx, pending, "failed", "", err.Error())
		return err
	}
	if err := s.store.RecordDownloadResult(ctx, pending, "submitted", gid, ""); err != nil {
		return fmt.Errorf("记录下载任务失败: %w", err)
	}
	s.log.Info("下载任务已提交", "item_id", item.ID, "gid", gid)
	return nil
}

// CanSubmitItemDownload 判断条目是否可（再次）提交 aria2 下载。
func CanSubmitItemDownload(status string) bool {
	return status != "submitting"
}

// ItemDownloadFailure 单条批量下载失败原因。
type ItemDownloadFailure struct {
	ItemID int64
	Error  string
}

const maxBatchItemDownloads = 50

// SubmitItemDownloads 批量提交条目下载；单条失败不影响其余条目。
func (s *Service) SubmitItemDownloads(ctx context.Context, itemIDs []int64) ([]store.Item, []ItemDownloadFailure) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	if len(itemIDs) > maxBatchItemDownloads {
		return nil, []ItemDownloadFailure{{ItemID: 0, Error: fmt.Sprintf("单次最多提交 %d 条", maxBatchItemDownloads)}}
	}
	seen := make(map[int64]struct{}, len(itemIDs))
	items := make([]store.Item, 0, len(itemIDs))
	var failures []ItemDownloadFailure
	for _, id := range itemIDs {
		if id <= 0 {
			failures = append(failures, ItemDownloadFailure{ItemID: id, Error: "无效的条目 ID"})
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if err := s.SubmitItemDownload(ctx, id); err != nil {
			failures = append(failures, ItemDownloadFailure{ItemID: id, Error: err.Error()})
			continue
		}
		item, err := s.store.GetItem(ctx, id)
		if err != nil {
			failures = append(failures, ItemDownloadFailure{ItemID: id, Error: err.Error()})
			continue
		}
		items = append(items, item)
	}
	return items, failures
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
		downloadURL := item.URL
		sub, subErr := s.store.GetSubscription(ctx, item.SubscriptionID)
		if subErr == nil && rss.NormalizeParser(sub.RSSParser) == rss.ParserMikan {
			proxyURL, err := s.store.GetSetting(ctx, proxySettingKey)
			if err == nil {
				if resolver, err := rss.NewFetcher(proxyURL); err == nil {
					if resolved, err := rss.ResolveMikanDownloadURL(ctx, resolver, downloadURL, sub.UseProxy); err == nil && resolved != "" {
						downloadURL = resolved
					}
				}
			}
		}
		item.URL = downloadURL
		gid, err := s.aria2.AddURI(ctx, downloadURL, item.Dir)
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
