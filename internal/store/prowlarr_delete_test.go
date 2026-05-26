package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteSubscription_RejectsProwlarrInternal(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset", "last_fetched_at", "last_error",
			"sort_order", "created_at", "updated_at",
		}).AddRow(
			int64(99), ProwlarrInternalMovieName, ProwlarrInternalFeedURLMovie, false, 1440, "", "UTC", "/movies",
			"", "", false, "generic", false, 1, 0, nil, "", 999999, now, now,
		))

	err = s.DeleteSubscription(context.Background(), 99)
	if err == nil || err.Error() != "不能删除系统 Prowlarr 订阅" {
		t.Fatalf("expected prowlarr delete error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
