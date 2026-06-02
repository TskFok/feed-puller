package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FeishuNotifyHistory 飞书通知发送历史。
type FeishuNotifyHistory struct {
	ID         int64     `json:"id"`
	EventType  string    `json:"event_type"`
	Source     string    `json:"source"`
	NotifyType string    `json:"notify_type"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	ItemCount  int       `json:"item_count"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateFeishuNotifyHistory 写入一条通知历史。
func (s *Store) CreateFeishuNotifyHistory(ctx context.Context, row FeishuNotifyHistory) error {
	eventType := strings.TrimSpace(row.EventType)
	source := strings.TrimSpace(row.Source)
	status := strings.TrimSpace(row.Status)
	if eventType == "" || status == "" {
		return fmt.Errorf("通知历史 event_type 与 status 不能为空")
	}
	if source == "" {
		source = "rss"
	}
	itemCount := row.ItemCount
	if itemCount <= 0 {
		itemCount = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO feishu_notify_history (event_type, source, notify_type, title, content, item_count, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, eventType, source, strings.TrimSpace(row.NotifyType), strings.TrimSpace(row.Title), row.Content, itemCount, status, nullIfEmpty(row.Error))
	if err != nil {
		return fmt.Errorf("写入通知历史失败: %w", err)
	}
	return nil
}

// ListFeishuNotifyHistoryPage 分页列出通知历史（按时间倒序）。
func (s *Store) ListFeishuNotifyHistoryPage(ctx context.Context, page, pageSize int) ([]FeishuNotifyHistory, int, error) {
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
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feishu_notify_history`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计通知历史失败: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, source, notify_type, title, content, item_count, status, COALESCE(error, ''), created_at
		FROM feishu_notify_history
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知历史失败: %w", err)
	}
	defer rows.Close()
	var out []FeishuNotifyHistory
	for rows.Next() {
		var row FeishuNotifyHistory
		if err := rows.Scan(
			&row.ID, &row.EventType, &row.Source, &row.NotifyType, &row.Title, &row.Content,
			&row.ItemCount, &row.Status, &row.Error, &row.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描通知历史失败: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
