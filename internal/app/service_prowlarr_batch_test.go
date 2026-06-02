package app

import (
	"log/slog"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func TestSubmitProwlarrReleases_TooMany(t *testing.T) {
	t.Parallel()
	svc := NewService(store.New(nil), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	inputs := make([]ProwlarrReleaseInput, maxBatchProwlarrDownloads+1)
	_, failures := svc.SubmitProwlarrReleases(t.Context(), inputs)
	if len(failures) != 1 || failures[0].GUID != "" {
		t.Fatalf("unexpected failures: %+v", failures)
	}
}

func TestSubmitProwlarrReleases_PartialFailure(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	expectProwlarrSettings(mock, map[string]string{
		"prowlarr_url":                "http://127.0.0.1:9696",
		"prowlarr_api_key":            "secret",
		"prowlarr_download_dir":       "/movies",
		"prowlarr_indexer_ids":        "[]",
		"prowlarr_subscription_id":    "9",
		"prowlarr_tv_subscription_id": "10",
	})

	// first release in progress
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(int64(9), "prowlarr:g1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(int64(42), int64(9), "g1", "Movie", `{}`, "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, "submitting", now, now))

	// second release invalid guid handled before settings

	items, failures := svc.SubmitProwlarrReleases(t.Context(), []ProwlarrReleaseInput{
		{GUID: "g1", Title: "Movie", InfoHash: "abc"},
		{GUID: "", Title: "Bad"},
	})
	if len(items) != 0 {
		t.Fatalf("expected no items, got %+v", items)
	}
	if len(failures) != 2 {
		t.Fatalf("expected 2 failures, got %+v", failures)
	}
	if failures[0].Error != ErrProwlarrReleaseInProgress.Error() {
		t.Fatalf("unexpected first failure: %+v", failures[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
