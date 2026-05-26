package store

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetProwlarrConfig_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	for _, key := range []string{
		"prowlarr_url", "prowlarr_api_key", "prowlarr_download_dir", "prowlarr_tv_download_dir",
		"prowlarr_movie_rename_enabled", "prowlarr_tmdb_api_key", "prowlarr_indexer_ids",
		"prowlarr_subscription_id", "prowlarr_tv_subscription_id",
	} {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}))
	}

	cfg, err := s.GetProwlarrConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Configured {
		t.Fatal("expected not configured")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertProwlarrItem_Insert(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(int64(9), "prowlarr:g1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO feed_items`)).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(
			int64(42), int64(9), "g1", "Movie", "", "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, "pending", time.Now(), time.Now(),
		))

	item, err := s.UpsertProwlarrItem(context.Background(), 9, "Movie", "magnet:?xt=urn:btih:abc", "prowlarr:g1", "g1", `{"media_type":"movie"}`)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != 42 || item.Title != "Movie" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIsProwlarrInternalSubscription(t *testing.T) {
	t.Parallel()
	if !IsProwlarrInternalSubscription(Subscription{FeedURL: ProwlarrInternalFeedURLMovie}) {
		t.Fatal("expected internal subscription")
	}
	if IsProwlarrInternalSubscription(Subscription{FeedURL: "https://example.com/feed"}) {
		t.Fatal("expected normal subscription")
	}
}
