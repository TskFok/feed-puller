package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// UpsertProwlarrItem 写入或更新 Prowlarr 搜索产生的 feed_item。
func (s *Store) UpsertProwlarrItem(ctx context.Context, subscriptionID int64, title, downloadURL, dedupeKey, guid, link string) (Item, error) {
	title = strings.TrimSpace(title)
	downloadURL = strings.TrimSpace(downloadURL)
	dedupeKey = strings.TrimSpace(dedupeKey)
	guid = strings.TrimSpace(guid)
	link = strings.TrimSpace(link)
	if subscriptionID <= 0 {
		return Item{}, fmt.Errorf("订阅 ID 无效")
	}
	if dedupeKey == "" {
		return Item{}, fmt.Errorf("dedupe_key 不能为空")
	}
	if downloadURL == "" {
		return Item{}, fmt.Errorf("下载地址不能为空")
	}

	var existingID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?
	`, subscriptionID, dedupeKey).Scan(&existingID)
	if err == nil {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE feed_items
			SET title = ?, guid = ?, link = ?, download_url = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, title, nullableString(guid), nullableString(link), downloadURL, existingID); err != nil {
			return Item{}, fmt.Errorf("更新 Prowlarr 条目失败: %w", err)
		}
		return s.GetItem(ctx, existingID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Item{}, fmt.Errorf("查询 Prowlarr 条目失败: %w", err)
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO feed_items (subscription_id, guid, title, link, download_url, dedupe_key, published_at, download_status)
		VALUES (?, ?, ?, ?, ?, ?, NULL, 'pending')
	`, subscriptionID, nullableString(guid), title, nullableString(link), downloadURL, dedupeKey)
	if err != nil {
		return Item{}, fmt.Errorf("创建 Prowlarr 条目失败: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetItem(ctx, id)
}

// ResetProwlarrItemForRetry 将失败/跳过的 Prowlarr 条目重置为 pending 以便重新下载。
func (s *Store) ResetProwlarrItemForRetry(ctx context.Context, itemID int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE feed_items
		SET download_status = 'pending', updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND download_status IN ('failed', 'skipped', 'pending')
	`, itemID)
	if err != nil {
		return fmt.Errorf("重置条目状态失败: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("条目当前不可重试")
	}
	return nil
}

const maxProwlarrSubmittedGuidsLookup = 200

// ListProwlarrSubmittedGuids 返回已提交、下载中或已完成的 Prowlarr release guid。
func (s *Store) ListProwlarrSubmittedGuids(ctx context.Context, guids []string) ([]string, error) {
	if len(guids) == 0 {
		return nil, nil
	}
	if len(guids) > maxProwlarrSubmittedGuidsLookup {
		return nil, fmt.Errorf("单次最多查询 %d 个 guid", maxProwlarrSubmittedGuidsLookup)
	}
	seen := make(map[string]struct{}, len(guids))
	dedupeKeys := make([]string, 0, len(guids))
	for _, raw := range guids {
		guid := strings.TrimSpace(raw)
		if guid == "" {
			continue
		}
		if _, ok := seen[guid]; ok {
			continue
		}
		seen[guid] = struct{}{}
		dedupeKeys = append(dedupeKeys, "prowlarr:"+guid)
	}
	if len(dedupeKeys) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(dedupeKeys))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(dedupeKeys)+3)
	for i, key := range dedupeKeys {
		args[i] = key
	}
	args[len(dedupeKeys)] = "submitting"
	args[len(dedupeKeys)+1] = "submitted"
	args[len(dedupeKeys)+2] = "completed"

	query := fmt.Sprintf(`
		SELECT COALESCE(guid, ''), dedupe_key
		FROM feed_items
		WHERE dedupe_key IN (%s)
		  AND download_status IN (?, ?, ?)
	`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询 Prowlarr 已提交 guid 失败: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0, len(dedupeKeys))
	outSeen := make(map[string]struct{}, len(dedupeKeys))
	for rows.Next() {
		var guid, dedupeKey string
		if err := rows.Scan(&guid, &dedupeKey); err != nil {
			return nil, fmt.Errorf("读取 Prowlarr guid 失败: %w", err)
		}
		guid = strings.TrimSpace(guid)
		if guid == "" {
			guid = strings.TrimPrefix(strings.TrimSpace(dedupeKey), "prowlarr:")
		}
		if guid == "" {
			continue
		}
		if _, ok := outSeen[guid]; ok {
			continue
		}
		outSeen[guid] = struct{}{}
		out = append(out, guid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("查询 Prowlarr 已提交 guid 失败: %w", err)
	}
	return out, nil
}
