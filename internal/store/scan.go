package store

import (
	"database/sql"
	"time"
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
	err := row.Scan(&sub.ID, &sub.Name, &sub.FeedURL, &sub.Enabled, &sub.PollIntervalMinutes, &sub.DownloadDir, &sub.UseProxy, &lastFetched, &sub.LastError, &sub.CreatedAt, &sub.UpdatedAt)
	if lastFetched.Valid {
		sub.LastFetchedAt = &lastFetched.Time
	}
	return sub, err
}

func scanSubscriptions(rows *sql.Rows) ([]Subscription, error) {
	var subscriptions []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, rows.Err()
}

func scanItems(rows *sql.Rows) ([]Item, error) {
	var items []Item
	for rows.Next() {
		var item Item
		var published sql.NullTime
		if err := rows.Scan(&item.ID, &item.SubscriptionID, &item.GUID, &item.Title, &item.Link, &item.DownloadURL, &item.DedupeKey, &published, &item.DownloadStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if published.Valid {
			item.PublishedAt = &published.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDownloadTasks(rows *sql.Rows) ([]DownloadTask, error) {
	var tasks []DownloadTask
	for rows.Next() {
		var task DownloadTask
		if err := rows.Scan(&task.ID, &task.ItemID, &task.SubscriptionID, &task.URL, &task.Dir, &task.Status, &task.Aria2GID, &task.Error, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
