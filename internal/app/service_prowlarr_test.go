package app

import (
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

var prowlarrSettingKeys = []string{
	"prowlarr_url", "prowlarr_api_key", "prowlarr_download_dir", "prowlarr_tv_download_dir",
	"prowlarr_movie_rename_enabled", "prowlarr_tmdb_api_key", "prowlarr_indexer_ids",
	"prowlarr_subscription_id", "prowlarr_tv_subscription_id",
}

func expectProwlarrSettings(mock sqlmock.Sqlmock, values map[string]string) {
	for _, key := range prowlarrSettingKeys {
		value := values[key]
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value))
	}
}

func expectEmptyProwlarrSettings(mock sqlmock.Sqlmock) {
	for _, key := range prowlarrSettingKeys {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}))
	}
}

func TestSearchProwlarrMovies_NotConfigured(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	expectEmptyProwlarrSettings(mock)

	_, err = svc.SearchProwlarrMovies(t.Context(), "inception", 100, 0)
	if !errors.Is(err, ErrProwlarrNotConfigured) {
		t.Fatalf("expected ErrProwlarrNotConfigured, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTestProwlarrConnection_UsesProvidedEmptyFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     store.ProwlarrConfig
		wantErr string
	}{
		{
			name: "empty api key",
			cfg: store.ProwlarrConfig{
				URL: "http://127.0.0.1:9696",
			},
			wantErr: "Prowlarr API Key 不能为空",
		},
		{
			name: "empty url",
			cfg: store.ProwlarrConfig{
				APIKey: "secret",
			},
			wantErr: "Prowlarr 地址不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

			err = svc.TestProwlarrConnection(t.Context(), tt.cfg)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q error, got %v", tt.wantErr, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSubmitProwlarrRelease_InProgress(t *testing.T) {
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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(int64(9), "prowlarr:g1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(
			int64(42), int64(9), "g1", "Movie", `{}`, "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, "submitted", now, now,
		))

	_, err = svc.SubmitProwlarrRelease(t.Context(), ProwlarrReleaseInput{
		GUID:     "g1",
		Title:    "Movie",
		InfoHash: "abc",
	})
	if !errors.Is(err, ErrProwlarrReleaseInProgress) {
		t.Fatalf("expected ErrProwlarrReleaseInProgress, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestParseSeasonEpisodeFromFilename(t *testing.T) {
	t.Parallel()
	season, episode, ok := parseSeasonEpisodeFromFilename("Show.S01E02.1080p.mkv")
	if !ok || season != 1 || episode != 2 {
		t.Fatalf("unexpected parse result: %d %d %v", season, episode, ok)
	}
}
