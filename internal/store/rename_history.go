package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	RenameHistoryStatusSuccess = "success"
	RenameHistoryStatusSkipped = "skipped"
	RenameHistoryStatusFailed  = "failed"
)

// RenameHistory AI 重命名操作记录。
type RenameHistory struct {
	ID               int64     `json:"id"`
	SubscriptionID   int64     `json:"subscription_id,omitempty"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	RenamedPath      string    `json:"renamed_path,omitempty"`
	AIPrompt         string    `json:"ai_prompt"`
	AIResponse       string    `json:"ai_response,omitempty"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateRenameHistory 写入一条重命名历史。
func (s *Store) CreateRenameHistory(ctx context.Context, row RenameHistory) error {
	status := strings.TrimSpace(row.Status)
	originalFilename := strings.TrimSpace(row.OriginalFilename)
	originalPath := strings.TrimSpace(row.OriginalPath)
	if status == "" || originalFilename == "" || originalPath == "" {
		return fmt.Errorf("重命名历史 status、original_filename 与 original_path 不能为空")
	}
	var subscriptionID any
	if row.SubscriptionID > 0 {
		subscriptionID = row.SubscriptionID
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rename_history (subscription_id, original_filename, original_path, renamed_path, ai_prompt, ai_response, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, subscriptionID, originalFilename, originalPath, nullIfEmpty(row.RenamedPath), row.AIPrompt, nullIfEmpty(row.AIResponse), status, nullIfEmpty(row.Error))
	if err != nil {
		return fmt.Errorf("写入重命名历史失败: %w", err)
	}
	return nil
}

// ListRenameHistoryPage 分页列出重命名历史（按时间倒序）。
func (s *Store) ListRenameHistoryPage(ctx context.Context, page, pageSize int) ([]RenameHistory, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rename_history`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计重命名历史失败: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(subscription_id, 0), original_filename, original_path, COALESCE(renamed_path, ''), ai_prompt, COALESCE(ai_response, ''), status, COALESCE(error, ''), created_at
		FROM rename_history
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询重命名历史失败: %w", err)
	}
	defer rows.Close()
	var out []RenameHistory
	for rows.Next() {
		var row RenameHistory
		if err := rows.Scan(
			&row.ID, &row.SubscriptionID, &row.OriginalFilename, &row.OriginalPath, &row.RenamedPath,
			&row.AIPrompt, &row.AIResponse, &row.Status, &row.Error, &row.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描重命名历史失败: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
