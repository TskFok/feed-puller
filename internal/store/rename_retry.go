package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	RenameRetryStatusPending    = "pending"
	RenameRetryStatusSucceeded  = "succeeded"
	RenameRetryStatusAbandoned  = "abandoned"
	maxRenameRetryListLimit     = 100
)

// RenameRetry 表示一条待重试的重命名任务。
type RenameRetry struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	FilePath    string    `json:"file_path"`
	RetryCount  int        `json:"retry_count"`
	FailedAt    time.Time  `json:"failed_at"`
	NextRetryAt time.Time  `json:"next_retry_at"`
	LastError   string     `json:"last_error,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EnqueueRenameRetry 在首次重命名失败时写入重试队列；若已有记录则忽略。
func (s *Store) EnqueueRenameRetry(ctx context.Context, retry RenameRetry) error {
	if retry.TaskID <= 0 {
		return fmt.Errorf("无效的任务 ID")
	}
	filePath := strings.TrimSpace(retry.FilePath)
	if filePath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if retry.FailedAt.IsZero() {
		retry.FailedAt = time.Now().UTC()
	}
	if retry.NextRetryAt.IsZero() {
		return fmt.Errorf("下次重试时间不能为空")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT IGNORE INTO rename_retries (task_id, file_path, retry_count, failed_at, next_retry_at, last_error, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, retry.TaskID, filePath, retry.RetryCount, retry.FailedAt, retry.NextRetryAt, strings.TrimSpace(retry.LastError), RenameRetryStatusPending)
	if err != nil {
		return fmt.Errorf("写入重命名重试队列失败: %w", err)
	}
	return nil
}

// ListDueRenameRetries 返回已到重试时间的 pending 记录。
func (s *Store) ListDueRenameRetries(ctx context.Context, now time.Time, limit int) ([]RenameRetry, error) {
	if limit <= 0 || limit > maxRenameRetryListLimit {
		limit = maxRenameRetryListLimit
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, file_path, retry_count, failed_at, next_retry_at, COALESCE(last_error, ''), status, created_at, updated_at
		FROM rename_retries
		WHERE status = ? AND next_retry_at <= ?
		ORDER BY next_retry_at ASC
		LIMIT ?
	`, RenameRetryStatusPending, now, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待重试重命名任务失败: %w", err)
	}
	defer rows.Close()
	out := make([]RenameRetry, 0)
	for rows.Next() {
		var row RenameRetry
		if err := rows.Scan(&row.ID, &row.TaskID, &row.FilePath, &row.RetryCount, &row.FailedAt, &row.NextRetryAt, &row.LastError, &row.Status, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// UpdateRenameRetryAfterAttempt 在一次重试尝试后更新计数、错误与下次重试时间或终态。
func (s *Store) UpdateRenameRetryAfterAttempt(ctx context.Context, id int64, retryCount int, lastError string, nextRetryAt *time.Time, status string) error {
	if id <= 0 {
		return fmt.Errorf("无效的重试记录 ID")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = RenameRetryStatusPending
	}
	if nextRetryAt != nil {
		_, err := s.db.ExecContext(ctx, `
			UPDATE rename_retries
			SET retry_count = ?, last_error = ?, next_retry_at = ?, status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, retryCount, strings.TrimSpace(lastError), *nextRetryAt, status, id)
		if err != nil {
			return fmt.Errorf("更新重命名重试记录失败: %w", err)
		}
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE rename_retries
		SET retry_count = ?, last_error = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, retryCount, strings.TrimSpace(lastError), status, id)
	if err != nil {
		return fmt.Errorf("更新重命名重试记录失败: %w", err)
	}
	return nil
}

// MarkRenameRetrySucceeded 将重试记录标记为成功（手动或自动重命名成功后调用）。
func (s *Store) MarkRenameRetrySucceeded(ctx context.Context, taskID int64) error {
	if taskID <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE rename_retries
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = ? AND status = ?
	`, RenameRetryStatusSucceeded, taskID, RenameRetryStatusPending)
	if err != nil {
		return fmt.Errorf("标记重命名重试成功失败: %w", err)
	}
	return nil
}

// GetRenameRetryByTaskID 按下载任务 ID 查询重试记录。
func (s *Store) GetRenameRetryByTaskID(ctx context.Context, taskID int64) (RenameRetry, error) {
	if taskID <= 0 {
		return RenameRetry{}, fmt.Errorf("无效的任务 ID")
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, task_id, file_path, retry_count, failed_at, next_retry_at, COALESCE(last_error, ''), status, created_at, updated_at
		FROM rename_retries WHERE task_id = ?
	`, taskID)
	var retry RenameRetry
	if err := row.Scan(&retry.ID, &retry.TaskID, &retry.FilePath, &retry.RetryCount, &retry.FailedAt, &retry.NextRetryAt, &retry.LastError, &retry.Status, &retry.CreatedAt, &retry.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return RenameRetry{}, err
		}
		return RenameRetry{}, fmt.Errorf("查询重命名重试记录失败: %w", err)
	}
	return retry, nil
}
