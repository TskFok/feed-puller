package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func TestRetryCompletedDownloadRename_Success(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "番剧 第02话.mp4")
	if err := os.WriteFile(from, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 2}"}}]}`))
	}))
	defer ai.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE id = ?`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "final_path", "created_at", "updated_at",
		}).AddRow(9, 90, 2, "magnet:?xt=urn:btih:abc", dir, "completed", "", "", "", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(2, "动漫", "https://example.test/feed", true, 30, "", "UTC", dir, "", "", false, "mikan", true, 1, 0, nil, "", 0, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(90)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(90, 2, "", "第2话", "", "magnet:?xt=urn:btih:abc", "k", nil, "completed", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(1, "test", ai.URL+"/v1", "gpt-test", "sk-test", now, now))

	target := filepath.Join(dir, "番剧 第02话 S01E02.mp4")
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET final_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(target, int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := svc.RetryCompletedDownloadRename(context.Background(), 9)
	if err != nil {
		t.Fatalf("RetryCompletedDownloadRename: %v", err)
	}
	if result.Skipped {
		t.Fatalf("expected rename, got skipped: %+v", result)
	}
	if result.ToPath != target {
		t.Fatalf("to = %q, want %q", result.ToPath, target)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRetryCompletedDownloadRename_PrefersStoredFinalPath(t *testing.T) {
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "番剧 第02话.mp4")
	otherFile := filepath.Join(dir, "large-other.mkv")
	if err := os.WriteFile(taskFile, []byte("small"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherFile, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 2}"}}]}`))
	}))
	defer ai.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE id = ?`)).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "final_path", "created_at", "updated_at",
		}).AddRow(11, 92, 2, "magnet:?xt=urn:btih:abc", dir, "completed", "", "", taskFile, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(2, "动漫", "https://example.test/feed", true, 30, "", "UTC", dir, "", "", false, "mikan", true, 1, 0, nil, "", 0, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(92)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(92, 2, "", "第2话", "", "magnet:?xt=urn:btih:abc", "k", nil, "completed", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(1, "test", ai.URL+"/v1", "gpt-test", "sk-test", now, now))

	target := filepath.Join(dir, "番剧 第02话 S01E02.mp4")
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET final_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(target, int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := svc.RetryCompletedDownloadRename(context.Background(), 11)
	if err != nil {
		t.Fatalf("RetryCompletedDownloadRename: %v", err)
	}
	if result.FromPath != taskFile {
		t.Fatalf("from = %q, want %q", result.FromPath, taskFile)
	}
	if result.ToPath != target {
		t.Fatalf("to = %q, want %q", result.ToPath, target)
	}
	if _, err := os.Stat(otherFile); err != nil {
		t.Fatalf("other task file should remain: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
