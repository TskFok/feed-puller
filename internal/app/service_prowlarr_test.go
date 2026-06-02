package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/prowlarr"
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
			int64(42), int64(9), "g1", "Movie", `{}`, "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, "submitting", now, now,
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

func TestSubmitProwlarrRelease_AllowsSubmittedReleaseAgain(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	aria2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": "1", "result": "gid-repeat"})
	}))
	defer aria2.Close()

	svc := NewService(store.New(db), downloader.NewAria2Client(aria2.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
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
	itemRows := func(status string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(
			int64(42), int64(9), "g1", "Movie", `{}`, "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, status, now, now,
		)
	}
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(itemRows("submitted"))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(itemRows("submitted"))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone", "download_dir",
			"include_keywords", "exclude_keywords", "use_proxy", "rss_parser", "ai_rename_enabled", "ai_rename_season",
			"ai_rename_episode_offset", "last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(
			int64(9), "Prowlarr", "", true, 60, "", "UTC", "/movies",
			"", "", false, "generic", false, 0, 0, nil, "", 1, now, now,
		))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = 'submitting'`)).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO download_tasks`)).
		WithArgs(int64(42), int64(9), "magnet:?xt=urn:btih:abc", "/movies", "submitted", "gid-repeat", nil).
		WillReturnResult(sqlmock.NewResult(100, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs("submitted", int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(itemRows("submitted"))

	item, err := svc.SubmitProwlarrRelease(t.Context(), ProwlarrReleaseInput{
		GUID:     "g1",
		Title:    "Movie",
		InfoHash: "abc",
	})
	if err != nil {
		t.Fatalf("SubmitProwlarrRelease returned error: %v", err)
	}
	if item.ID != 42 || item.DownloadStatus != "submitted" {
		t.Fatalf("item = %+v, want id 42 submitted", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitProwlarrReleaseRejectsLoginPageDownloadURL(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	svc.prowlarrTorrentFetcher = func(ctx context.Context, rawURL string, headers map[string]string) (prowlarrTorrentFetchResult, error) {
		if rawURL != "https://torrent9.test/download" {
			t.Fatalf("unexpected download URL %q", rawURL)
		}
		return prowlarrTorrentFetchResult{
			Body:        []byte("<html><title>login</title><body>Please login</body></html>"),
			FinalURL:    "https://torrent9.test/login",
			ContentType: "text/html; charset=utf-8",
		}, nil
	}

	expectProwlarrSettings(mock, map[string]string{
		"prowlarr_url":                "http://127.0.0.1:9696",
		"prowlarr_api_key":            "secret",
		"prowlarr_download_dir":       "/movies",
		"prowlarr_indexer_ids":        "[]",
		"prowlarr_subscription_id":    "9",
		"prowlarr_tv_subscription_id": "10",
	})

	_, err = svc.SubmitProwlarrRelease(t.Context(), ProwlarrReleaseInput{
		GUID:        "g-login",
		Title:       "Movie",
		DownloadURL: "https://torrent9.test/download",
	})
	if err == nil || !strings.Contains(err.Error(), "未返回有效 torrent") {
		t.Fatalf("expected invalid torrent error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveProwlarrDownloadURLConvertsTorrentToMagnet(t *testing.T) {
	t.Parallel()
	svc := NewService(store.New(nil), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	var gotHeaders map[string]string
	svc.prowlarrTorrentFetcher = func(ctx context.Context, rawURL string, headers map[string]string) (prowlarrTorrentFetchResult, error) {
		if rawURL != "https://prowlarr.test/api/v1/download" {
			t.Fatalf("unexpected download URL %q", rawURL)
		}
		gotHeaders = headers
		return prowlarrTorrentFetchResult{
			Body:        minimalTorrentBytes(),
			FinalURL:    rawURL,
			ContentType: "application/x-bittorrent",
		}, nil
	}

	got, err := svc.resolveProwlarrDownloadURL(t.Context(), store.ProwlarrConfig{
		URL:    "https://prowlarr.test",
		APIKey: "secret",
	}, prowlarr.Release{DownloadURL: "https://prowlarr.test/api/v1/download"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "magnet:?") {
		t.Fatalf("expected magnet URL, got %q", got)
	}
	if gotHeaders["X-Api-Key"] != "secret" {
		t.Fatalf("X-Api-Key header = %q, want secret", gotHeaders["X-Api-Key"])
	}
	if gotHeaders["Referer"] != "https://prowlarr.test/" {
		t.Fatalf("Referer header = %q", gotHeaders["Referer"])
	}
}

func TestResolveProwlarrDownloadURLReturnsMagnetRedirect(t *testing.T) {
	t.Parallel()
	want := "magnet:?xt=urn:btih:7ebeccb7a17432e22ee362a6a7544b303d2f0f1f&tr=udp://tracker.opentrackr.org:1337/announce"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, want, http.StatusFound)
	}))
	defer server.Close()
	svc := NewService(store.New(nil), downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	got, err := svc.resolveProwlarrDownloadURL(t.Context(), store.ProwlarrConfig{}, prowlarr.Release{
		DownloadURL: server.URL + "/9/download",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolved URL = %q, want %q", got, want)
	}
}

func minimalTorrentBytes() []byte {
	return []byte("d4:infod6:lengthi1e4:name4:test12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee")
}

func TestParseSeasonEpisodeFromFilename(t *testing.T) {
	t.Parallel()
	season, episode, ok := parseSeasonEpisodeFromFilename("Show.S01E02.1080p.mkv")
	if !ok || season != 1 || episode != 2 {
		t.Fatalf("unexpected parse result: %d %d %v", season, episode, ok)
	}
}
