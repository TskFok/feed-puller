package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteSubscription_RemovesDependentRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const subID int64 = 42
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(subID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset", "last_fetched_at", "last_error",
			"sort_order", "created_at", "updated_at",
		}).AddRow(
			subID, "Sub", "https://example.test/feed.xml", true, 30, "", "UTC", "/data",
			"", "", false, "generic", false, 1, 0, nil, "", 0, now, now,
		))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM download_tasks WHERE subscription_id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM feed_items WHERE subscription_id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM subscriptions WHERE id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := New(db)
	if err := s.DeleteSubscription(context.Background(), subID); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
