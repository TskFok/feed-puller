package app

import (
	"context"
	"strings"

	"feed-puller/internal/downloader"
)

// SyncAria2DownloadStatus 轮询 aria2 中已提交任务的状态，完成或失败时写回数据库。
func (s *Service) SyncAria2DownloadStatus(ctx context.Context) error {
	tasks, err := s.store.ListSubmittedDownloadTasks(ctx, 100)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		gid := strings.TrimSpace(task.Aria2GID)
		if gid == "" {
			continue
		}
		status, err := s.aria2.TellStatus(ctx, gid)
		if err != nil {
			s.log.Warn("查询 aria2 任务状态失败", "task_id", task.ID, "gid", gid, "error", err)
			continue
		}
		state, errMsg := downloader.ParseAria2TaskStatus(status)
		switch state {
		case downloader.Aria2TaskComplete:
			sub, subErr := s.store.GetSubscription(ctx, task.SubscriptionID)
			itemTitle := ""
			if item, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
				itemTitle = item.Title
			}
			if subErr == nil {
				s.maybeRenameDownloadFile(ctx, sub, itemTitle, status)
			} else {
				s.log.Warn("读取订阅失败，跳过重命名", "subscription_id", task.SubscriptionID, "error", subErr)
			}
			if err := s.store.CompleteDownloadTask(ctx, task.ID, task.ItemID); err != nil {
				s.log.Warn("记录下载完成失败", "task_id", task.ID, "error", err)
				continue
			}
			s.log.Info("下载已完成", "task_id", task.ID, "item_id", task.ItemID, "gid", gid)
		case downloader.Aria2TaskError:
			if err := s.store.FailDownloadTaskFromAria2(ctx, task.ID, task.ItemID, errMsg); err != nil {
				s.log.Warn("记录下载失败状态失败", "task_id", task.ID, "error", err)
				continue
			}
			s.log.Info("下载失败", "task_id", task.ID, "item_id", task.ItemID, "gid", gid, "error", errMsg)
		case downloader.Aria2TaskRemoved:
			// 任务被外部移除，不再轮询。
			continue
		default:
			// active / waiting / paused：继续等待。
		}
	}
	return nil
}
