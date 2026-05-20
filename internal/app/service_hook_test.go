package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func newHookService(t *testing.T) (*Service, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return svc, mock, func() { _ = db.Close() }
}

func TestNormalizeAria2HookEvent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		out  Aria2HookEvent
		fail bool
	}{
		{in: "complete", out: Aria2HookEventComplete},
		{in: "file-complete", out: Aria2HookEventFileComplete},
		{in: "on-download-complete", out: Aria2HookEventFileComplete},
		{in: "on-bt-download-complete", out: Aria2HookEventBTComplete},
		{in: "bt-complete", out: Aria2HookEventBTComplete},
		{in: "error", out: Aria2HookEventError},
		{in: "on-download-error", out: Aria2HookEventError},
		{in: "stop", out: Aria2HookEventStop},
		{in: "on-download-stop", out: Aria2HookEventStop},
		{in: " ", fail: true},
		{in: "pause", fail: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := NormalizeAria2HookEvent(tc.in)
			if tc.fail {
				if err == nil {
					t.Fatalf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.out {
				t.Fatalf("got %q, want %q", got, tc.out)
			}
		})
	}
}

func TestHandleAria2Hook_CompleteWritesCompletedState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	aria2Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "1",
			"result": map[string]any{
				"status": "complete",
				"files": []any{
					map[string]any{
						"path":            "/data/anime/foo.mp4",
						"completedLength": "1000",
						"length":          "1000",
					},
				},
			},
		})
	}))
	defer aria2Srv.Close()

	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2Srv.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-hook-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(7, 70, 3, "https://example.test/a.mp4", "/data", "submitted", "gid-hook-1", "", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(3, "动漫", "https://example.test/feed", true, 30, "", "UTC", "/data", "", "", false, "generic", false, 1, 0, nil, "", 0, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(70)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(70, 3, "", "第2话", "", "https://example.test/a.mp4", "k", nil, "submitted", now, now))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(70)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.HandleAria2Hook(context.Background(), "gid-hook-1", Aria2HookEventBTComplete, "/data/anime/foo.mp4", ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_ErrorWritesFailedState(t *testing.T) {
	svc, mock, cleanup := newHookService(t)
	defer cleanup()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-hook-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(8, 80, 3, "https://example.test/b.mp4", "/data", "submitted", "gid-hook-2", "", now, now))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs("tracker timeout", int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = 'failed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(80)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.HandleAria2Hook(context.Background(), "gid-hook-2", Aria2HookEventError, "", "tracker timeout"); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_GIDMissingReturnsSentinel(t *testing.T) {
	svc, mock, cleanup := newHookService(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("unknown-gid").
		WillReturnError(sql.ErrNoRows)

	err := svc.HandleAria2Hook(context.Background(), "unknown-gid", Aria2HookEventComplete, "", "")
	if !errors.Is(err, ErrAria2HookTaskNotFound) {
		t.Fatalf("err = %v, want ErrAria2HookTaskNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_IdempotentWhenAlreadyTerminal(t *testing.T) {
	svc, mock, cleanup := newHookService(t)
	defer cleanup()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-hook-3").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(9, 90, 3, "https://example.test/c.mp4", "/data", "completed", "gid-hook-3", "", now, now))

	if err := svc.HandleAria2Hook(context.Background(), "gid-hook-3", Aria2HookEventComplete, "/data/c.mp4", ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_StopDoesNotMutateState(t *testing.T) {
	svc, mock, cleanup := newHookService(t)
	defer cleanup()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-hook-4").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(11, 110, 3, "https://example.test/d.mp4", "/data", "submitted", "gid-hook-4", "", now, now))

	if err := svc.HandleAria2Hook(context.Background(), "gid-hook-4", Aria2HookEventStop, "", ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	// 应仅做查询，不应有 begin/exec/commit 出现。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_MetadataFileDoesNotComplete(t *testing.T) {
	svc, mock, cleanup := newHookService(t)
	defer cleanup()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-meta").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(12, 120, 3, "magnet:?xt=urn:btih:abc", "/data", "submitted", "gid-meta", "", now, now))

	metaPath := "/data/[METADATA][ANi]+大賢者+-+07+.mp4"
	if err := svc.HandleAria2Hook(context.Background(), "gid-meta", Aria2HookEventBTComplete, metaPath, ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_FileCompleteWaitsWhileAria2Active(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	aria2Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "1",
			"result": map[string]any{
				"status": "active",
				"files": []any{
					map[string]any{"path": "/data/[ANi]foo.mp4", "completedLength": "1", "length": "100"},
				},
			},
		})
	}))
	defer aria2Srv.Close()

	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2Srv.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-active").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(13, 130, 3, "magnet:?xt=urn:btih:def", "/data", "submitted", "gid-active", "", now, now))

	if err := svc.HandleAria2Hook(context.Background(), "gid-active", Aria2HookEventFileComplete, "/data/[ANi]foo.mp4", ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_BTCompleteWaitsForRealFile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	aria2Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "1",
			"result": map[string]any{
				"status": "complete",
				"files": []any{
					map[string]any{
						"path":            "/data/[METADATA][ANi]+foo+.mp4",
						"completedLength": "100",
						"length":          "100",
					},
					map[string]any{
						"path":            "/data/[ANi]foo - 07.mp4",
						"completedLength": "10",
						"length":          "1000",
					},
				},
			},
		})
	}))
	defer aria2Srv.Close()

	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2Srv.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-bt").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(14, 140, 3, "magnet:?xt=urn:btih:xyz", "/data", "submitted", "gid-bt", "", now, now))

	if err := svc.HandleAria2Hook(context.Background(), "gid-bt", Aria2HookEventBTComplete, "/data/[ANi]foo - 07.mp4", ""); err != nil {
		t.Fatalf("HandleAria2Hook: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAria2Hook_EmptyGIDRejected(t *testing.T) {
	svc, _, cleanup := newHookService(t)
	defer cleanup()
	if err := svc.HandleAria2Hook(context.Background(), "  ", Aria2HookEventComplete, "", ""); err == nil {
		t.Fatal("expected error for empty gid")
	}
}
