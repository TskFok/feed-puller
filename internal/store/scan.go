package store

import (
	"database/sql"
	"time"

	"feed-puller/internal/rss"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FeishuOpenID, &user.FeishuName, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func scanSubscription(row rowScanner) (Subscription, error) {
	var sub Subscription
	var lastFetched sql.NullTime
	err := row.Scan(&sub.ID, &sub.Name, &sub.FeedURL, &sub.Enabled, &sub.PollIntervalMinutes, &sub.PollCron, &sub.PollCronTimezone, &sub.DownloadDir, &sub.IncludeKeywords, &sub.ExcludeKeywords, &sub.UseProxy, &sub.RSSParser, &sub.AIRenameEnabled, &sub.AIRenameSeason, &sub.AIRenameEpOffset, &lastFetched, &sub.LastError, &sub.SortOrder, &sub.CreatedAt, &sub.UpdatedAt)
	if lastFetched.Valid {
		sub.LastFetchedAt = &lastFetched.Time
	}
	sub.RSSParser = rss.NormalizeParser(sub.RSSParser)
	return sub, err
}

func scanSubscriptions(rows *sql.Rows) ([]Subscription, error) {
	// 使用非 nil 空切片，JSON 编码为 []；nil 切片会变成 null，前端 .map 会崩溃。
	subscriptions := make([]Subscription, 0)
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, rows.Err()
}

func scanItemRow(row rowScanner) (Item, error) {
	var item Item
	var published sql.NullTime
	if err := row.Scan(&item.ID, &item.SubscriptionID, &item.GUID, &item.Title, &item.Link, &item.DownloadURL, &item.DedupeKey, &published, &item.DownloadStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Item{}, err
	}
	if published.Valid {
		item.PublishedAt = &published.Time
	}
	return item, nil
}

func scanItems(rows *sql.Rows) ([]Item, error) {
	items := make([]Item, 0)
	for rows.Next() {
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDownloadTasks(rows *sql.Rows) ([]DownloadTask, error) {
	tasks := make([]DownloadTask, 0)
	for rows.Next() {
		var task DownloadTask
		if err := rows.Scan(&task.ID, &task.ItemID, &task.SubscriptionID, &task.URL, &task.Dir, &task.Status, &task.Aria2GID, &task.Error, &task.FinalPath, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
