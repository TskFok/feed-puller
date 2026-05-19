package app

import (
	"context"
	"encoding/json"
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

func TestSyncAria2DownloadStatusMarksComplete(t *testing.T) {
	aria2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "aria2.tellStatus" {
			t.Fatalf("method = %q", req.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      "1",
			"result": map[string]any{
				"status": "complete",
				"files": []any{
					map[string]any{"path": "/data/anime/xxx 02.mp4"},
				},
			},
		})
	}))
	defer aria2.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(1, 10, 2, "https://example.test/a.mp4", "/data", "submitted", "gid-1", "", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(2, "动漫", "https://example.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 0, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(10, 2, "", "第2话", "", "https://example.test/a.mp4", "k", nil, "submitted", now, now))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.SyncAria2DownloadStatus(context.Background()); err != nil {
		t.Fatalf("SyncAria2DownloadStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMaybeRenameDownloadFile_WithLocalFallback(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "xxx 02.mp4")
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
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
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(1, "test", ai.URL+"/v1", "gpt-test", "sk-test", now, now))

	sub := store.Subscription{
		ID:               2,
		AIRenameEnabled:  true,
		AIRenameSeason:   1,
		AIRenameEpOffset: 2,
	}
	status := map[string]any{
		"files": []any{map[string]any{"path": from}},
	}
	svc.maybeRenameDownloadFile(context.Background(), sub, "第2话", status)

	target := filepath.Join(dir, "xxx S01E04.mp4")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected renamed file: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
