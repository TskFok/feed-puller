package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"feed-puller/internal/auth"
	"feed-puller/internal/rss"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Migrate(ctx context.Context) error {
	for _, statement := range migrations {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行数据库迁移失败: %w", err)
		}
	}
	return nil
}

func (s *Store) BootstrapAdmin(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return fmt.Errorf("管理员邮箱和密码不能为空")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), updated_at = CURRENT_TIMESTAMP
	`, email, hash)
	if err != nil {
		return fmt.Errorf("初始化管理员失败: %w", err)
	}
	return nil
}

func (s *Store) Authenticate(ctx context.Context, email, password string) (User, error) {
	user, err := s.UserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	if !auth.VerifyPassword(user.PasswordHash, password) {
		return User{}, sql.ErrNoRows
	}
	return user, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, COALESCE(feishu_open_id, ''), COALESCE(feishu_name, ''), created_at, updated_at
		FROM users WHERE email = ?
	`, strings.TrimSpace(strings.ToLower(email)))
	return scanUser(row)
}

func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, COALESCE(feishu_open_id, ''), COALESCE(feishu_name, ''), created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	return scanUser(row)
}

func (s *Store) UserByFeishuOpenID(ctx context.Context, openID string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, COALESCE(feishu_open_id, ''), COALESCE(feishu_name, ''), created_at, updated_at
		FROM users WHERE feishu_open_id = ?
	`, strings.TrimSpace(openID))
	return scanUser(row)
}

func (s *Store) BindFeishu(ctx context.Context, userID int64, openID, name string) error {
	if strings.TrimSpace(openID) == "" {
		return fmt.Errorf("飞书 open_id 不能为空")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET feishu_open_id = ?, feishu_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, strings.TrimSpace(openID), strings.TrimSpace(name), userID)
	if err != nil {
		return fmt.Errorf("绑定飞书失败: %w", err)
	}
	return nil
}

func (s *Store) UnbindFeishu(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET feishu_open_id = NULL, feishu_name = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("解绑飞书失败: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, rawToken string, userID int64, expiresAt time.Time) error {
	tokenHash := HashToken(rawToken)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)
	`, tokenHash, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, rawToken string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, HashToken(rawToken))
	return err
}

func (s *Store) UserBySession(ctx context.Context, rawToken string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, COALESCE(u.feishu_open_id, ''), COALESCE(u.feishu_name, ''), u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > CURRENT_TIMESTAMP
	`, HashToken(rawToken))
	return scanUser(row)
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE name = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取设置失败: %w", err)
	}
	return value, nil
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (name, value) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP
	`, key, value)
	if err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	return nil
}

func (s *Store) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions ORDER BY sort_order ASC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询订阅失败: %w", err)
	}
	defer rows.Close()
	return scanSubscriptions(rows)
}

const subscriptionExcludeProwlarrSQL = ` AND feed_url NOT LIKE 'prowlarr://%'`

func (s *Store) CountSubscriptions(ctx context.Context) (int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscriptions WHERE 1=1`+subscriptionExcludeProwlarrSQL).Scan(&total); err != nil {
		return 0, fmt.Errorf("统计订阅数量失败: %w", err)
	}
	return total, nil
}

func (s *Store) ListSubscriptionsPage(ctx context.Context, page, pageSize int) ([]Subscription, int, error) {
	page, pageSize, offset := NormalizePage(page, pageSize)
	total, err := s.CountSubscriptions(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions
		WHERE feed_url NOT LIKE 'prowlarr://%'
		ORDER BY sort_order ASC, id DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询订阅失败: %w", err)
	}
	defer rows.Close()
	items, err := scanSubscriptions(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) ListSubscriptionIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM subscriptions
		WHERE feed_url NOT LIKE 'prowlarr://%'
		ORDER BY sort_order ASC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询订阅 ID 失败: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) DueSubscriptions(ctx context.Context, now time.Time) ([]Subscription, error) {
	nowUTC := now.UTC()
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions
		WHERE enabled = TRUE
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询待拉取订阅失败: %w", err)
	}
	defer rows.Close()
	raw, err := scanSubscriptions(rows)
	if err != nil {
		return nil, err
	}
	due := make([]Subscription, 0, len(raw))
	for i := range raw {
		if SubscriptionPollDue(&raw[i], nowUTC) {
			due = append(due, raw[i])
		}
	}
	return due, nil
}

func (s *Store) GetSubscription(ctx context.Context, id int64) (Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+subscriptionColumns+`
		FROM subscriptions WHERE id = ?
	`, id)
	return scanSubscription(row)
}

func (s *Store) CreateSubscription(ctx context.Context, sub Subscription) (Subscription, error) {
	if err := validateSubscription(sub); err != nil {
		return Subscription{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (name, feed_url, enabled, poll_interval_minutes, poll_cron, poll_cron_timezone, download_dir, include_keywords, exclude_keywords, use_proxy, rss_parser, ai_rename_enabled, ai_rename_season, ai_rename_episode_offset, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MIN(sort_order), 1) - 1 FROM subscriptions AS s))
	`, strings.TrimSpace(sub.Name), strings.TrimSpace(sub.FeedURL), sub.Enabled, sub.PollIntervalMinutes, NormalizePollCron(sub.PollCron), NormalizePollCronTimezone(sub.PollCronTimezone), strings.TrimSpace(sub.DownloadDir), sub.IncludeKeywords, sub.ExcludeKeywords, sub.UseProxy, rss.NormalizeParser(sub.RSSParser), sub.AIRenameEnabled, normalizeAIRenameSeason(sub.AIRenameSeason), sub.AIRenameEpOffset)
	if err != nil {
		return Subscription{}, fmt.Errorf("创建订阅失败: %w", err)
	}
	id, _ := result.LastInsertId()
	return s.GetSubscription(ctx, id)
}

func (s *Store) UpdateSubscription(ctx context.Context, id int64, sub Subscription) (Subscription, error) {
	if err := validateSubscription(sub); err != nil {
		return Subscription{}, err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET name = ?, feed_url = ?, enabled = ?, poll_interval_minutes = ?, poll_cron = ?, poll_cron_timezone = ?, download_dir = ?, include_keywords = ?, exclude_keywords = ?, use_proxy = ?, rss_parser = ?, ai_rename_enabled = ?, ai_rename_season = ?, ai_rename_episode_offset = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, strings.TrimSpace(sub.Name), strings.TrimSpace(sub.FeedURL), sub.Enabled, sub.PollIntervalMinutes, NormalizePollCron(sub.PollCron), NormalizePollCronTimezone(sub.PollCronTimezone), strings.TrimSpace(sub.DownloadDir), sub.IncludeKeywords, sub.ExcludeKeywords, sub.UseProxy, rss.NormalizeParser(sub.RSSParser), sub.AIRenameEnabled, normalizeAIRenameSeason(sub.AIRenameSeason), sub.AIRenameEpOffset, id)
	if err != nil {
		return Subscription{}, fmt.Errorf("更新订阅失败: %w", err)
	}
	return s.GetSubscription(ctx, id)
}

func (s *Store) ReorderSubscriptions(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("订阅顺序不能为空")
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("订阅 ID 无效")
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("订阅 ID 重复")
		}
		seen[id] = struct{}{}
	}

	all, err := s.ListSubscriptions(ctx)
	if err != nil {
		return err
	}
	if len(all) != len(ids) {
		return fmt.Errorf("请提供全部订阅的顺序")
	}
	for _, sub := range all {
		if _, ok := seen[sub.ID]; !ok {
			return fmt.Errorf("请提供全部订阅的顺序")
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("更新订阅顺序失败: %w", err)
	}
	defer tx.Rollback()

	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE subscriptions SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, i, id); err != nil {
			return fmt.Errorf("更新订阅顺序失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("更新订阅顺序失败: %w", err)
	}
	return nil
}

func (s *Store) DeleteSubscription(ctx context.Context, id int64) error {
	sub, err := s.GetSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("订阅不存在")
	}
	if IsProwlarrInternalSubscription(sub) {
		return fmt.Errorf("不能删除系统 Prowlarr 订阅")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM download_tasks WHERE subscription_id = ?`, id); err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM feed_items WHERE subscription_id = ?`, id); err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}
	return nil
}

// NewFeedItemDownloadStatus 决定 RSS 新条目的初始下载状态。
func NewFeedItemDownloadStatus(downloadURL string, previewOnly bool) string {
	if strings.TrimSpace(downloadURL) == "" {
		return "skipped"
	}
	if previewOnly {
		return "preview"
	}
	return "pending"
}

func (s *Store) SaveFeedItems(ctx context.Context, subscriptionID int64, items []rss.FeedItem, previewOnly bool) ([]Item, error) {
	var out []Item
	for _, item := range items {
		key := rss.DedupeKey(item)
		if key == "" {
			continue
		}
		var existingID int64
		err := s.db.QueryRowContext(ctx, `
			SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?
		`, subscriptionID, key).Scan(&existingID)
		if err == nil {
			if _, err := s.db.ExecContext(ctx, `
				UPDATE feed_items SET title = ?, link = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
			`, strings.TrimSpace(item.Title), strings.TrimSpace(item.Link), existingID); err != nil {
				return out, fmt.Errorf("更新条目失败: %w", err)
			}
			row := s.db.QueryRowContext(ctx, `
				SELECT id, subscription_id, COALESCE(guid, ''), title, COALESCE(link, ''), COALESCE(download_url, ''), dedupe_key, published_at, download_status, created_at, updated_at
				FROM feed_items WHERE id = ?
			`, existingID)
			stored, err := scanItemRow(row)
			if err != nil {
				return out, fmt.Errorf("读取条目失败: %w", err)
			}
			out = append(out, stored)
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return out, fmt.Errorf("查询条目失败: %w", err)
		}
		status := NewFeedItemDownloadStatus(item.DownloadURL, previewOnly)
		res, err := s.db.ExecContext(ctx, `
			INSERT INTO feed_items (subscription_id, guid, title, link, download_url, dedupe_key, published_at, download_status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, subscriptionID, strings.TrimSpace(item.GUID), strings.TrimSpace(item.Title), strings.TrimSpace(item.Link), strings.TrimSpace(item.DownloadURL), key, nullableTime(item.PublishedAt), status)
		if err != nil {
			return out, fmt.Errorf("保存条目失败: %w", err)
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return out, fmt.Errorf("读取新条目 id 失败: %w", err)
		}
		row := s.db.QueryRowContext(ctx, `
			SELECT id, subscription_id, COALESCE(guid, ''), title, COALESCE(link, ''), COALESCE(download_url, ''), dedupe_key, published_at, download_status, created_at, updated_at
			FROM feed_items WHERE id = ?
		`, newID)
		stored, err := scanItemRow(row)
		if err != nil {
			return out, fmt.Errorf("读取条目失败: %w", err)
		}
		out = append(out, stored)
	}
	return out, nil
}

func (s *Store) GetItem(ctx context.Context, id int64) (Item, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, subscription_id, COALESCE(guid, ''), title, COALESCE(link, ''), COALESCE(download_url, ''), dedupe_key, published_at, download_status, created_at, updated_at
		FROM feed_items WHERE id = ?
	`, id)
	item, err := scanItemRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, fmt.Errorf("条目不存在")
	}
	return item, err
}

func (s *Store) MarkSubscriptionFetched(ctx context.Context, id int64, errText string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE subscriptions SET last_fetched_at = CURRENT_TIMESTAMP, last_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, nullableString(errText), id)
	return err
}

func (s *Store) ListItems(ctx context.Context, subscriptionID int64, limit int) ([]Item, error) {
	items, _, err := s.ListItemsPage(ctx, subscriptionID, 1, limit)
	return items, err
}

func (s *Store) countItems(ctx context.Context, subscriptionID int64) (int, error) {
	var total int
	var err error
	if subscriptionID > 0 {
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feed_items WHERE subscription_id = ?`, subscriptionID).Scan(&total)
	} else {
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feed_items`).Scan(&total)
	}
	if err != nil {
		return 0, fmt.Errorf("统计条目数量失败: %w", err)
	}
	return total, nil
}

func (s *Store) ListItemsPage(ctx context.Context, subscriptionID int64, page, pageSize int) ([]Item, int, error) {
	page, pageSize, offset := NormalizePage(page, pageSize)
	total, err := s.countItems(ctx, subscriptionID)
	if err != nil {
		return nil, 0, err
	}
	query := `
		SELECT id, subscription_id, COALESCE(guid, ''), title, COALESCE(link, ''), COALESCE(download_url, ''), dedupe_key, published_at, download_status, created_at, updated_at
		FROM feed_items
	`
	var rows *sql.Rows
	if subscriptionID > 0 {
		rows, err = s.db.QueryContext(ctx, query+` WHERE subscription_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`, subscriptionID, pageSize, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, query+` ORDER BY id DESC LIMIT ? OFFSET ?`, pageSize, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("查询条目失败: %w", err)
	}
	defer rows.Close()
	items, err := scanItems(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetDownloadTask 按任务 ID 查询下载任务。
func (s *Store) GetDownloadTask(ctx context.Context, id int64) (DownloadTask, error) {
	if id <= 0 {
		return DownloadTask{}, fmt.Errorf("无效的任务 ID")
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT `+downloadTaskColumns+`
		FROM download_tasks WHERE id = ?
	`, id)
	var task DownloadTask
	if err := row.Scan(&task.ID, &task.ItemID, &task.SubscriptionID, &task.URL, &task.Dir, &task.Status, &task.Aria2GID, &task.Error, &task.FinalPath, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return DownloadTask{}, err
	}
	return task, nil
}

func (s *Store) ListDownloads(ctx context.Context, limit int) ([]DownloadTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+downloadTaskColumns+`
		FROM download_tasks ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询下载任务失败: %w", err)
	}
	defer rows.Close()
	return scanDownloadTasks(rows)
}

func (s *Store) PendingDownloads(ctx context.Context, limit int) ([]PendingDownload, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT i.id, i.subscription_id, i.download_url, sub.download_dir
		FROM feed_items i
		JOIN subscriptions sub ON sub.id = i.subscription_id
		WHERE i.download_status IN ('pending', 'preview', 'failed') AND i.download_url IS NOT NULL AND i.download_url <> ''
		ORDER BY i.id ASC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待下载条目失败: %w", err)
	}
	defer rows.Close()
	var pending []PendingDownload
	for rows.Next() {
		var item PendingDownload
		if err := rows.Scan(&item.ItemID, &item.SubscriptionID, &item.URL, &item.Dir); err != nil {
			return nil, err
		}
		pending = append(pending, item)
	}
	return pending, rows.Err()
}

func (s *Store) MarkDownloadSubmitting(ctx context.Context, itemID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE feed_items SET download_status = 'submitting', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, itemID)
	return err
}

func (s *Store) RecordDownloadResult(ctx context.Context, pending PendingDownload, status, gid, errText string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO download_tasks (item_id, subscription_id, url, dir, status, aria2_gid, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, pending.ItemID, pending.SubscriptionID, pending.URL, pending.Dir, status, nullableString(gid), nullableString(errText))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE feed_items SET download_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, status, pending.ItemID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func validateSubscription(sub Subscription) error {
	if strings.TrimSpace(sub.Name) == "" {
		return fmt.Errorf("订阅名称不能为空")
	}
	if strings.TrimSpace(sub.FeedURL) == "" {
		return fmt.Errorf("订阅地址不能为空")
	}
	if strings.TrimSpace(sub.DownloadDir) == "" {
		return fmt.Errorf("保存路径不能为空")
	}
	if err := validatePollSchedule(sub); err != nil {
		return err
	}
	if err := rss.ValidateKeywordPatterns(sub.IncludeKeywords, sub.ExcludeKeywords); err != nil {
		return err
	}
	if err := rss.ValidateParser(sub.RSSParser); err != nil {
		return err
	}
	return nil
}

func normalizeAIRenameSeason(season int) int {
	if season < 1 {
		return 1
	}
	return season
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
