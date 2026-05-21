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
	"feed-puller/internal/paths"
	"feed-puller/internal/store"
)

func TestRetryCompletedDownloadRename_PathMap(t *testing.T) {
	hostRoot := t.TempDir()
	containerRoot := t.TempDir()
	from := filepath.Join(containerRoot, "番剧 第02话.mp4")
	if err := os.WriteFile(from, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	mapper := paths.NewMapper(hostRoot, containerRoot)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)), mapper)
	now := time.Now().UTC()
	hostDir := hostRoot

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE id = ?`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "final_path", "created_at", "updated_at",
		}).AddRow(10, 91, 2, "magnet:?xt=urn:btih:abc", hostDir, "completed", "", "", "", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(2, "动漫", "https://example.test/feed", true, 30, "", "UTC", hostDir, "", "", false, "mikan", true, 1, 0, nil, "", 0, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(91, 2, "", "第2话", "", "magnet:?xt=urn:btih:abc", "k", nil, "completed", now, now))

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 2}"}}]}`))
	}))
	defer ai.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(1, "test", ai.URL+"/v1", "gpt-test", "sk-test", now, now))

	target := filepath.Join(containerRoot, "番剧 第02话 S01E02.mp4")
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET final_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(target, int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := svc.RetryCompletedDownloadRename(context.Background(), 10)
	if err != nil {
		t.Fatalf("RetryCompletedDownloadRename: %v", err)
	}
	if result.ToPath != target {
		t.Fatalf("to = %q, want %q", result.ToPath, target)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
