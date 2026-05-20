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
			if downloader.IsGIDNotFound(err) {
				// aria2 仅会在任务进入终态后清理记录，因此 GID 已不存在视为已完成；
				// 跳过重命名（无 files 列表），仅写库以便出现在「下载完成」列表中。
				if completeErr := s.store.CompleteDownloadTask(ctx, task.ID, task.ItemID); completeErr != nil {
					s.log.Warn("记录下载完成失败", "task_id", task.ID, "error", completeErr)
					continue
				}
				s.log.Info("aria2 已无任务记录，按完成处理", "task_id", task.ID, "item_id", task.ItemID, "gid", gid)
				continue
			}
			s.log.Warn("查询 aria2 任务状态失败", "task_id", task.ID, "gid", gid, "error", err)
			continue
		}
		state, errMsg := downloader.ParseAria2TaskStatus(status)
		switch state {
		case downloader.Aria2TaskComplete:
			if !downloader.IsAria2DownloadReady(status) {
				s.log.Info("aria2 报告 complete 但尚无已完成的实体文件，继续等待",
					"task_id", task.ID, "item_id", task.ItemID, "gid", gid)
				continue
			}
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
