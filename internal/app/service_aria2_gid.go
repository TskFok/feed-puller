package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

// tellStatusForDownloadTask 查询任务 aria2 状态，沿 followedBy 链切换到实体下载 GID 并写回数据库。
func (s *Service) tellStatusForDownloadTask(ctx context.Context, task store.DownloadTask) (effectiveGID string, status map[string]any, err error) {
	stored := strings.TrimSpace(task.Aria2GID)
	if stored == "" {
		return "", nil, downloader.ErrEmptyGID
	}
	effectiveGID, status, err = s.aria2.TellStatusEffective(ctx, stored)
	if err != nil {
		return stored, status, err
	}
	if effectiveGID != stored {
		if updateErr := s.store.UpdateDownloadTaskAria2GID(ctx, task.ID, effectiveGID); updateErr != nil {
			return effectiveGID, status, updateErr
		}
		s.log.Info("aria2 元数据下载已切换为实体下载 GID",
			"task_id", task.ID, "item_id", task.ItemID, "from_gid", stored, "to_gid", effectiveGID)
	}
	return effectiveGID, status, nil
}

// syncDownloadTaskGID 仅解析并更新 GID，供进度展示等场景使用。
func (s *Service) syncDownloadTaskGID(ctx context.Context, task store.DownloadTask) (string, error) {
	effective, _, err := s.tellStatusForDownloadTask(ctx, task)
	return effective, err
}

// findDownloadTaskForAria2Hook 按钩子 GID 查找任务；若 GID 为 followedBy 子任务则通过 following 反查并更新 GID。
func (s *Service) findDownloadTaskForAria2Hook(ctx context.Context, gid string) (store.DownloadTask, error) {
	gid = strings.TrimSpace(gid)
	task, err := s.store.DownloadTaskByAria2GID(ctx, gid)
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.DownloadTask{}, err
	}
	status, err := s.aria2.TellStatus(ctx, gid)
	if err != nil {
		return store.DownloadTask{}, ErrAria2HookTaskNotFound
	}
	parent := downloader.FollowingGID(status)
	if parent == "" {
		return store.DownloadTask{}, ErrAria2HookTaskNotFound
	}
	task, err = s.store.DownloadTaskByAria2GID(ctx, parent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.DownloadTask{}, ErrAria2HookTaskNotFound
		}
		return store.DownloadTask{}, err
	}
	if err := s.store.UpdateDownloadTaskAria2GID(ctx, task.ID, gid); err != nil {
		return store.DownloadTask{}, err
	}
	s.log.Info("aria2 hook: 通过 following 关联到下载任务",
		"task_id", task.ID, "parent_gid", parent, "effective_gid", gid)
	return task, nil
}
