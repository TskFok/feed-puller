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

func TestNewFeedItemDownloadStatus(t *testing.T) {
	t.Parallel()
	if got := NewFeedItemDownloadStatus("", false); got != "skipped" {
		t.Fatalf("empty url: got %q", got)
	}
	if got := NewFeedItemDownloadStatus("http://d/a.mp4", false); got != "pending" {
		t.Fatalf("scheduled poll: got %q", got)
	}
	if got := NewFeedItemDownloadStatus("http://d/a.mp4", true); got != "preview" {
		t.Fatalf("manual preview: got %q", got)
	}
}

func TestSaveFeedItems_PreviewOnly(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const subID int64 = 7
	key := "guid:g1"
	rssItems := []rss.FeedItem{{GUID: "g1", Title: "Hello", Link: "http://l", DownloadURL: "http://d/file.mp4"}}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(subID, key).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO feed_items").
		WithArgs(subID, "g1", "Hello", "http://l", "http://d/file.mp4", key, nil, "preview").
		WillReturnResult(sqlmock.NewResult(88, 1))

	now := time.Now()
	mock.ExpectQuery("SELECT id, subscription_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(int64(88), subID, "g1", "Hello", "http://l", "http://d/file.mp4", key, nil, "preview", now, now))

	s := New(db)
	got, err := s.SaveFeedItems(context.Background(), subID, rssItems, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DownloadStatus != "preview" {
		t.Fatalf("got %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
