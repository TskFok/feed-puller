package store

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/rss"
)

func TestSaveFeedItems_InsertReturnsStoredRow(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const subID int64 = 5
	rssItems := []rss.FeedItem{
		{GUID: "g1", Title: "Hello", Link: "http://l", DownloadURL: "http://d/file.mp4"},
	}
	key := rss.DedupeKey(rssItems[0])

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(subID, key).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO feed_items").
		WithArgs(subID, "g1", "Hello", "http://l", "http://d/file.mp4", key, nil, "pending").
		WillReturnResult(sqlmock.NewResult(88, 1))

	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	sqlRows := sqlmock.NewRows([]string{
		"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
	}).AddRow(int64(88), subID, "g1", "Hello", "http://l", "http://d/file.mp4", key, nil, "pending", now, now)

	mock.ExpectQuery("SELECT id, subscription_id, COALESCE\\(guid, ''\\), title, COALESCE\\(link, ''\\), COALESCE\\(download_url, ''\\), dedupe_key, published_at, download_status, created_at, updated_at").
		WithArgs(int64(88)).
		WillReturnRows(sqlRows)

	s := New(db)
	got, err := s.SaveFeedItems(context.Background(), subID, rssItems)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != 88 || got[0].Title != "Hello" || got[0].DownloadStatus != "pending" {
		t.Fatalf("unexpected row: %+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
