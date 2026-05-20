package httpapi

import (
	"bytes"
	"database/sql"
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

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

const testHookSecret = "test-hook-secret-1234567890"

func newHookServer(t *testing.T, secret string) (*Server, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := app.NewService(repo, downloader.NewAria2Client("", ""), log)
	cfg := config.Config{Aria2HookSecret: secret}
	srv := New(cfg, repo, svc, log)
	return srv, mock, func() { _ = db.Close() }
}

func doHookRequest(t *testing.T, srv *Server, secretHeader string, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/downloads/aria2-hook", bytes.NewReader(buf))
	if secretHeader != "" {
		req.Header.Set("Authorization", "Bearer "+secretHeader)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestAria2Hook_RejectsWhenSecretMissing(t *testing.T) {
	srv, _, cleanup := newHookServer(t, "")
	defer cleanup()
	rec := doHookRequest(t, srv, "anything", map[string]any{"gid": "x", "event": "complete"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestAria2Hook_RejectsWrongBearer(t *testing.T) {
	srv, _, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	rec := doHookRequest(t, srv, "wrong-secret", map[string]any{"gid": "x", "event": "complete"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestAria2Hook_AcceptsXHookSecretHeader(t *testing.T) {
	srv, mock, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("gid-h-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(1, 10, 2, "https://example.test/a.mp4", "/data", "submitted", "gid-h-1", "", now, now))
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
		}).AddRow(10, 2, "", "第1话", "", "https://example.test/a.mp4", "k", nil, "submitted", now, now))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	body, _ := json.Marshal(map[string]any{"gid": "gid-h-1", "event": "on-bt-download-complete", "file_path": "/data/a.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/api/downloads/aria2-hook", bytes.NewReader(body))
	req.Header.Set("X-Hook-Secret", testHookSecret)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"matched":true`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAria2Hook_ReturnsMatchedFalseForUnknownGID(t *testing.T) {
	srv, mock, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("unknown-gid").
		WillReturnError(sql.ErrNoRows)
	rec := doHookRequest(t, srv, testHookSecret, map[string]any{"gid": "unknown-gid", "event": "complete"})
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["matched"] != false {
		t.Fatalf("matched = %v, want false", resp["matched"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAria2Hook_ReturnsInternalErrorOnDBError(t *testing.T) {
	srv, mock, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("db-error-gid").
		WillReturnError(errors.New("db boom"))
	rec := doHookRequest(t, srv, testHookSecret, map[string]any{"gid": "db-error-gid", "event": "complete"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAria2Hook_RejectsNonPost(t *testing.T) {
	srv, _, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/downloads/aria2-hook", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestAria2Hook_RejectsUnknownEvent(t *testing.T) {
	srv, _, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	rec := doHookRequest(t, srv, testHookSecret, map[string]any{"gid": "x", "event": "pause"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAria2Hook_RejectsEmptyGID(t *testing.T) {
	srv, _, cleanup := newHookServer(t, testHookSecret)
	defer cleanup()
	rec := doHookRequest(t, srv, testHookSecret, map[string]any{"gid": "  ", "event": "complete"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
}
