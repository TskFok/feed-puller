package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"feed-puller/internal/downloader"
	"feed-puller/internal/downloads"
	"feed-puller/internal/rename"
	"feed-puller/internal/store"
)

// RenameDownloadResult 表示一次重命名操作的结果。
type RenameDownloadResult struct {
	FromPath string `json:"from_path,omitempty"`
	ToPath   string `json:"to_path,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Message  string `json:"message,omitempty"`
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
	out := RenameDownloadResult{FromPath: from, ToPath: to, Skipped: skipped}
	if skipped {
		out.Message = "文件名已符合刮削格式，无需重命名"
	} else {
		out.Message = "重命名成功"
	}
	return out, nil
}

func (s *Service) resolveCompletedDownloadFilePath(ctx context.Context, task store.DownloadTask) (string, error) {
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
