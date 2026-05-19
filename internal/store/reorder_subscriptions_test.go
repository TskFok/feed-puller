package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReorderSubscriptions_UpdatesSortOrder(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := sqlmock.NewRows([]string{
		"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
		"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
		"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
		"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
	}).
		AddRow(1, "A", "https://a.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 0, nowUTC(), nowUTC()).
		AddRow(2, "B", "https://b.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 1, nowUTC(), nowUTC())

	listQuery := "\n\t\tSELECT " + subscriptionColumns + "\n\t\tFROM subscriptions ORDER BY sort_order ASC, id DESC\n\t"
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).WillReturnRows(now)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(0, int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(1, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := New(db)
	if err := s.ReorderSubscriptions(context.Background(), []int64{2, 1}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReorderSubscriptions_RejectsPartialIDs(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := sqlmock.NewRows([]string{
		"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
		"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
		"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
		"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
	}).
		AddRow(1, "A", "https://a.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 0, nowUTC(), nowUTC()).
		AddRow(2, "B", "https://b.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 1, nowUTC(), nowUTC())

	listQuery := "\n\t\tSELECT " + subscriptionColumns + "\n\t\tFROM subscriptions ORDER BY sort_order ASC, id DESC\n\t"
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).WillReturnRows(now)

	s := New(db)
	if err := s.ReorderSubscriptions(context.Background(), []int64{2}); err == nil {
		t.Fatal("expected error for partial subscription ids")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
