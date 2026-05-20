package store

import (
	"context"
	"fmt"
	"strings"
)

// ListSubmittedDownloadTasks 返回已提交 aria2、等待完成确认的任务。
func (s *Store) ListSubmittedDownloadTasks(ctx context.Context, limit int) ([]DownloadTask, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, item_id, subscription_id, url, dir, status, COALESCE(aria2_gid, ''), COALESCE(error, ''), created_at, updated_at
		FROM download_tasks
		WHERE status = 'submitted' AND aria2_gid IS NOT NULL AND aria2_gid <> ''
		ORDER BY id ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询进行中的下载任务失败: %w", err)
	}
	defer rows.Close()
	return scanDownloadTasks(rows)
}

// ListActiveDownloads 返回已提交 aria2、尚未标记完成的任务（含条目标题与订阅名称）。
func (s *Store) ListActiveDownloads(ctx context.Context, limit int) ([]ActiveDownload, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT dt.id, dt.item_id, dt.subscription_id, sub.name, COALESCE(i.title, ''), dt.url, dt.dir,
			COALESCE(dt.aria2_gid, ''), dt.updated_at
		FROM download_tasks dt
		JOIN feed_items i ON i.id = dt.item_id
		JOIN subscriptions sub ON sub.id = dt.subscription_id
		WHERE dt.status = 'submitted' AND dt.aria2_gid IS NOT NULL AND dt.aria2_gid <> ''
		ORDER BY dt.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询进行中的下载失败: %w", err)
	}
	defer rows.Close()
	out := make([]ActiveDownload, 0)
	for rows.Next() {
		var row ActiveDownload
		if err := rows.Scan(&row.ID, &row.ItemID, &row.SubscriptionID, &row.SubscriptionName, &row.Title, &row.URL, &row.Dir, &row.Aria2GID, &row.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// DownloadTaskByAria2GID 按 aria2 gid 查找下载任务，供 aria2 hook 回调使用。
// 找不到时返回 sql.ErrNoRows 包裹的错误，调用方应据此返回 404，避免误报。
func (s *Store) DownloadTaskByAria2GID(ctx context.Context, gid string) (DownloadTask, error) {
	gid = strings.TrimSpace(gid)
	if gid == "" {
		return DownloadTask{}, fmt.Errorf("aria2 gid 不能为空")
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, item_id, subscription_id, url, dir, status, COALESCE(aria2_gid, ''), COALESCE(error, ''), created_at, updated_at
		FROM download_tasks WHERE aria2_gid = ? ORDER BY id DESC LIMIT 1
	`, gid)
	var task DownloadTask
	if err := row.Scan(&task.ID, &task.ItemID, &task.SubscriptionID, &task.URL, &task.Dir, &task.Status, &task.Aria2GID, &task.Error, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return DownloadTask{}, err
	}
	return task, nil
}

// UpdateDownloadTaskAria2GID 更新下载任务关联的 aria2 GID（磁力元数据完成后 followedBy 切换时使用）。
func (s *Store) UpdateDownloadTaskAria2GID(ctx context.Context, taskID int64, gid string) error {
	gid = strings.TrimSpace(gid)
	if gid == "" {
		return fmt.Errorf("aria2 gid 不能为空")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE download_tasks SET aria2_gid = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, gid, taskID)
	if err != nil {
		return fmt.Errorf("更新下载任务 GID 失败: %w", err)
	}
	return nil
}

// CompleteDownloadTask 将下载任务与对应 feed 条目标记为已完成。
func (s *Store) CompleteDownloadTask(ctx context.Context, taskID, itemID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		UPDATE download_tasks SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, taskID); err != nil {
		return fmt.Errorf("更新下载任务状态失败: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE feed_items SET download_status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, itemID); err != nil {
		return fmt.Errorf("更新条目下载状态失败: %w", err)
	}
	return tx.Commit()
}

// FailDownloadTaskFromAria2 将 aria2 侧失败同步到下载任务与 feed 条目。
func (s *Store) FailDownloadTaskFromAria2(ctx context.Context, taskID, itemID int64, errText string) error {
	errText = strings.TrimSpace(errText)
	if errText == "" {
		errText = "aria2 下载失败"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		UPDATE download_tasks SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, errText, taskID); err != nil {
		return fmt.Errorf("更新下载任务状态失败: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE feed_items SET download_status = 'failed', updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, itemID); err != nil {
		return fmt.Errorf("更新条目下载状态失败: %w", err)
	}
	return tx.Commit()
}

// ListCompletedDownloads 返回已完成的下载记录（含条目标题与订阅名称）。
func (s *Store) ListCompletedDownloads(ctx context.Context, limit int) ([]CompletedDownload, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT dt.id, dt.item_id, dt.subscription_id, sub.name, COALESCE(i.title, ''), dt.url, dt.dir, dt.updated_at
		FROM download_tasks dt
		JOIN feed_items i ON i.id = dt.item_id
		JOIN subscriptions sub ON sub.id = dt.subscription_id
		WHERE dt.status = 'completed'
		ORDER BY dt.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询已完成下载失败: %w", err)
	}
	defer rows.Close()
	out := make([]CompletedDownload, 0)
	for rows.Next() {
		var row CompletedDownload
		if err := rows.Scan(&row.ID, &row.ItemID, &row.SubscriptionID, &row.SubscriptionName, &row.Title, &row.URL, &row.Dir, &row.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
