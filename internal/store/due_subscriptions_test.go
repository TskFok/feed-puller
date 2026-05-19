package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func addSubscriptionRow(
	rows *sqlmock.Rows,
	id int64,
	lastFetched any,
	createdAt time.Time,
	pollInterval int,
	pollCron string,
) *sqlmock.Rows {
	return rows.AddRow(
		id,
		"Sub",
		"https://example.test/feed.xml",
		true,
		pollInterval,
		pollCron,
		"UTC",
		"/data",
		"",
		"",
		false,
		"generic",
		false,
		1,
		0,
		lastFetched,
		"",
		0,
		createdAt,
		createdAt,
	)
}

func TestDueSubscriptions_ExcludesFreshNeverFetchedInterval(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	created := now.Add(-2 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
		"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
		"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
		"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
	})
	addSubscriptionRow(rows, 1, nil, created, 30, "")

	query := "\n\t\tSELECT " + subscriptionColumns + "\n\t\tFROM subscriptions\n\t\tWHERE enabled = TRUE\n\t\tORDER BY sort_order ASC, id ASC\n\t"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

	s := New(db)
	due, err := s.DueSubscriptions(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("expected no due subscriptions, got %d", len(due))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDueSubscriptions_IncludesIntervalAfterFirstWindow(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	created := now.Add(-31 * time.Minute)

	rows := sqlmock.NewRows([]string{
		"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
		"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
		"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
		"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
	})
	addSubscriptionRow(rows, 1, nil, created, 30, "")

	query := "\n\t\tSELECT " + subscriptionColumns + "\n\t\tFROM subscriptions\n\t\tWHERE enabled = TRUE\n\t\tORDER BY sort_order ASC, id ASC\n\t"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

	s := New(db)
	due, err := s.DueSubscriptions(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != 1 {
		t.Fatalf("expected one due subscription, got %+v", due)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDueSubscriptions_ExcludesFreshNeverFetchedCron(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 5, 19, 10, 30, 0, 0, time.UTC)
	created := time.Date(2026, 5, 19, 10, 5, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
		"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
		"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
		"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
	})
	addSubscriptionRow(rows, 2, nil, created, 30, "0 * * * *")

	query := "\n\t\tSELECT " + subscriptionColumns + "\n\t\tFROM subscriptions\n\t\tWHERE enabled = TRUE\n\t\tORDER BY sort_order ASC, id ASC\n\t"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

	s := New(db)
	due, err := s.DueSubscriptions(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("expected no due cron subscriptions before first tick, got %d", len(due))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
