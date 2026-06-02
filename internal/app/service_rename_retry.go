package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"feed-puller/internal/downloader"
	"feed-puller/internal/downloads"
	"feed-puller/internal/rename"
	"feed-puller/internal/store"
)

const maxRenameAutoRetries = 3

var renameRetryDelays = []time.Duration{
	time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

// RenameDownloadResult 表示一次重命名操作的结果。
type RenameDownloadResult struct {
	FromPath string `json:"from_path,omitempty"`
	ToPath   string `json:"to_path,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Message  string `json:"message,omitempty"`
}

func nextRenameRetryAt(failedAt time.Time, completedRetries int) (time.Time, bool) {
	if completedRetries < 0 || completedRetries >= len(renameRetryDelays) {
		return time.Time{}, false
	}
	return failedAt.Add(renameRetryDelays[completedRetries]), true
}

func (s *Service) scheduleRenameRetry(ctx context.Context, taskID int64, filePath, errMsg string) {
	if taskID <= 0 {
		return
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return
	}
	now := time.Now().UTC()
	nextAt, ok := nextRenameRetryAt(now, 0)
	if !ok {
		return
	}
	if err := s.store.EnqueueRenameRetry(ctx, store.RenameRetry{
		TaskID:      taskID,
		FilePath:    filePath,
		RetryCount:  0,
		FailedAt:    now,
		NextRetryAt: nextAt,
		LastError:   errMsg,
	}); err != nil {
		s.log.Warn("写入重命名重试队列失败", "task_id", taskID, "error", err)
		return
	}
	s.log.Info("已加入重命名重试队列", "task_id", taskID, "next_retry_at", nextAt)
}

// ProcessDueRenameRetries 处理已到期的重命名重试任务。
func (s *Service) ProcessDueRenameRetries(ctx context.Context) error {
	retries, err := s.store.ListDueRenameRetries(ctx, time.Now().UTC(), 50)
	if err != nil {
		return err
	}
	for _, retry := range retries {
		s.processRenameRetry(ctx, retry)
	}
	return nil
}

func (s *Service) processRenameRetry(ctx context.Context, retry store.RenameRetry) {
	task, err := s.store.GetDownloadTask(ctx, retry.TaskID)
	if err != nil {
		s.log.Warn("重命名重试: 读取下载任务失败", "task_id", retry.TaskID, "error", err)
		return
	}
	if downloads.DownloadStatus(task.Status) != downloads.StatusCompleted {
		if markErr := s.store.UpdateRenameRetryAfterAttempt(ctx, retry.ID, retry.RetryCount, "下载任务未完成", nil, store.RenameRetryStatusAbandoned); markErr != nil {
			s.log.Warn("重命名重试: 更新状态失败", "retry_id", retry.ID, "error", markErr)
		}
		return
	}

	sub, err := s.store.GetSubscription(ctx, task.SubscriptionID)
	if err != nil {
		s.log.Warn("重命名重试: 读取订阅失败", "task_id", retry.TaskID, "error", err)
		return
	}
	item := store.Item{Title: ""}
	if fetched, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
		item = fetched
	}

	filePath, err := s.resolveRenameRetryFilePath(ctx, task, retry.FilePath)
	if err != nil {
		s.handleRenameRetryFailure(ctx, retry, sub, item, err.Error())
		return
	}

	result, renameErr := s.attemptDownloadRename(ctx, sub, item, filePath)
	if renameErr == nil {
		if path := strings.TrimSpace(result.ToPath); path != "" {
			if updateErr := s.store.UpdateDownloadTaskFinalPath(ctx, retry.TaskID, path); updateErr != nil {
				s.log.Warn("重命名重试: 更新文件路径失败", "task_id", retry.TaskID, "path", path, "error", updateErr)
			}
		} else if path := strings.TrimSpace(result.FromPath); path != "" {
			if updateErr := s.store.UpdateDownloadTaskFinalPath(ctx, retry.TaskID, path); updateErr != nil {
				s.log.Warn("重命名重试: 更新文件路径失败", "task_id", retry.TaskID, "path", path, "error", updateErr)
			}
		}
		if markErr := s.store.UpdateRenameRetryAfterAttempt(ctx, retry.ID, retry.RetryCount, "", nil, store.RenameRetryStatusSucceeded); markErr != nil {
			s.log.Warn("重命名重试: 标记成功失败", "retry_id", retry.ID, "error", markErr)
		}
		s.log.Info("重命名重试成功", "task_id", retry.TaskID, "retry_count", retry.RetryCount+1)
		return
	}

	s.handleRenameRetryFailure(ctx, retry, sub, item, renameErr.Error())
}

func (s *Service) handleRenameRetryFailure(ctx context.Context, retry store.RenameRetry, sub store.Subscription, item store.Item, errMsg string) {
	nextCount := retry.RetryCount + 1
	if nextCount >= maxRenameAutoRetries {
		if markErr := s.store.UpdateRenameRetryAfterAttempt(ctx, retry.ID, nextCount, errMsg, nil, store.RenameRetryStatusAbandoned); markErr != nil {
			s.log.Warn("重命名重试: 标记放弃失败", "retry_id", retry.ID, "error", markErr)
		}
		s.log.Warn("重命名重试全部失败", "task_id", retry.TaskID, "error", errMsg)
		s.notifyRenameRetryExhausted(ctx, sub, item, retry.FilePath, errMsg)
		return
	}
	nextAt, ok := nextRenameRetryAt(retry.FailedAt, nextCount)
	if !ok {
		if markErr := s.store.UpdateRenameRetryAfterAttempt(ctx, retry.ID, nextCount, errMsg, nil, store.RenameRetryStatusAbandoned); markErr != nil {
			s.log.Warn("重命名重试: 标记放弃失败", "retry_id", retry.ID, "error", markErr)
		}
		s.notifyRenameRetryExhausted(ctx, sub, item, retry.FilePath, errMsg)
		return
	}
	if markErr := s.store.UpdateRenameRetryAfterAttempt(ctx, retry.ID, nextCount, errMsg, &nextAt, store.RenameRetryStatusPending); markErr != nil {
		s.log.Warn("重命名重试: 更新下次重试时间失败", "retry_id", retry.ID, "error", markErr)
		return
	}
	s.log.Info("重命名重试失败，等待下次重试",
		"task_id", retry.TaskID, "attempt", nextCount, "next_retry_at", nextAt, "error", errMsg)
}

func (s *Service) notifyRenameRetryExhausted(ctx context.Context, sub store.Subscription, item store.Item, filePath, errMsg string) {
	msg := fmt.Sprintf("重命名失败（已自动重试 %d 次）: %s", maxRenameAutoRetries, strings.TrimSpace(errMsg))
	payload := feishuPayloadFromSubscription(sub, item, strings.TrimSpace(filePath), msg)
	s.queueFeishuNotify(ctx, feishuNotifyFail, payload)
}

func (s *Service) attemptDownloadRename(ctx context.Context, sub store.Subscription, item store.Item, filePath string) (RenameDownloadResult, error) {
	if store.IsProwlarrInternalSubscription(sub) {
		finalPath, renameErr := s.renameProwlarrAt(ctx, sub, item, filePath)
		if renameErr != nil {
			return RenameDownloadResult{FromPath: filePath}, renameErr
		}
		if strings.TrimSpace(finalPath) == strings.TrimSpace(filePath) {
			return RenameDownloadResult{FromPath: filePath, ToPath: finalPath, Skipped: true}, nil
		}
		return RenameDownloadResult{FromPath: filePath, ToPath: finalPath}, nil
	}
	if !sub.AIRenameEnabled {
		return RenameDownloadResult{}, fmt.Errorf("订阅未启用 AI 重命名")
	}
	from, to, skipped, err := s.renameDownloadFileAt(ctx, sub, item.Title, filePath)
	if err != nil {
		return RenameDownloadResult{FromPath: from}, err
	}
	if skipped {
		return RenameDownloadResult{FromPath: from, ToPath: to, Skipped: true}, nil
	}
	return RenameDownloadResult{FromPath: from, ToPath: to}, nil
}

func (s *Service) resolveRenameRetryFilePath(ctx context.Context, task store.DownloadTask, storedPath string) (string, error) {
	if path := strings.TrimSpace(s.mapDownloadPath(storedPath)); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return s.resolveCompletedDownloadFilePath(ctx, task)
}

// RetryCompletedDownloadRename 对已完成的下载任务重新执行 AI 刮削重命名。
func (s *Service) RetryCompletedDownloadRename(ctx context.Context, taskID int64) (RenameDownloadResult, error) {
	task, err := s.store.GetDownloadTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RenameDownloadResult{}, fmt.Errorf("下载任务不存在")
		}
		return RenameDownloadResult{}, err
	}
	if downloads.DownloadStatus(task.Status) != downloads.StatusCompleted {
		return RenameDownloadResult{}, fmt.Errorf("仅支持对已完成的任务重命名")
	}

	sub, err := s.store.GetSubscription(ctx, task.SubscriptionID)
	if err != nil {
		return RenameDownloadResult{}, fmt.Errorf("读取订阅失败: %w", err)
	}
	if !sub.AIRenameEnabled {
		return RenameDownloadResult{}, fmt.Errorf("请先在订阅设置中启用 AI 重命名")
	}

	itemTitle := ""
	if item, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
		itemTitle = item.Title
	}

	filePath, err := s.resolveCompletedDownloadFilePath(ctx, task)
	if err != nil {
		return RenameDownloadResult{}, err
	}

	from, to, skipped, err := s.renameDownloadFileAt(ctx, sub, itemTitle, filePath)
	if err != nil {
		return RenameDownloadResult{}, err
	}
	if path := strings.TrimSpace(to); path != "" {
		if updateErr := s.store.UpdateDownloadTaskFinalPath(ctx, taskID, path); updateErr != nil {
			s.log.Warn("更新下载文件路径失败", "task_id", taskID, "path", path, "error", updateErr)
		}
	} else if path := strings.TrimSpace(from); path != "" {
		if updateErr := s.store.UpdateDownloadTaskFinalPath(ctx, taskID, path); updateErr != nil {
			s.log.Warn("更新下载文件路径失败", "task_id", taskID, "path", path, "error", updateErr)
		}
	}
	out := RenameDownloadResult{FromPath: from, ToPath: to, Skipped: skipped}
	if skipped {
		out.Message = "文件名已符合刮削格式，无需重命名"
	} else {
		out.Message = "重命名成功"
	}
	if markErr := s.store.MarkRenameRetrySucceeded(ctx, taskID); markErr != nil {
		s.log.Warn("标记重命名重试成功失败", "task_id", taskID, "error", markErr)
	}
	return out, nil
}

func (s *Service) resolveCompletedDownloadFilePath(ctx context.Context, task store.DownloadTask) (string, error) {
	if path := strings.TrimSpace(s.mapDownloadPath(task.FinalPath)); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	gid := strings.TrimSpace(task.Aria2GID)
	if gid != "" {
		_, status, err := s.aria2.TellStatusEffective(ctx, gid)
		if err == nil {
			if path, pathErr := downloader.Aria2DownloadPath(status); pathErr == nil {
				path = s.mapDownloadPath(path)
				if _, statErr := os.Stat(path); statErr == nil {
					return path, nil
				}
			}
		}
	}
	dir := strings.TrimSpace(s.mapDownloadPath(task.Dir))
	if dir == "" {
		return "", fmt.Errorf("下载目录为空")
	}
	return rename.FindLargestMediaFileInDir(dir)
}
