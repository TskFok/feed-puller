package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func TestRecordRenameHistory_Success(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "番剧 第02话.mp4")
	if err := os.WriteFile(from, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"anime_name\":\"番剧\",\"episode\": 2}"}}]}`))
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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "request_options", "created_at", "updated_at"}).
			AddRow(1, "test", ai.URL+"/v1", "gpt-test", "sk-test", "", now, now))

	target := filepath.Join(dir, "番剧 S01E02.mp4")
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO rename_history (subscription_id, original_filename, original_path, renamed_path, ai_prompt, ai_response, status, error)`)).
		WithArgs(int64(2), "番剧 第02话.mp4", from, target, sqlmock.AnyArg(), `{"anime_name":"番剧","episode": 2}`, store.RenameHistoryStatusSuccess, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sub := store.Subscription{
		ID:              2,
		AIRenameEnabled: true,
		AIRenameSeason:  1,
	}
	fromPath, toPath, skipped, renameErr := svc.renameDownloadFileAt(context.Background(), sub, "第2话", from)
	if renameErr != nil {
		t.Fatalf("renameDownloadFileAt: %v", renameErr)
	}
	if skipped || toPath != target || fromPath != from {
		t.Fatalf("from=%q to=%q skipped=%v", fromPath, toPath, skipped)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRenameDownloadFileAt_AIHTTPFailureDoesNotUseLocalFallback(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "番剧 第02集.mp4")
	if err := os.WriteFile(from, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "request_options", "created_at", "updated_at"}).
			AddRow(1, "test", "http://127.0.0.1:1/v1", "gpt-test", "sk-test", "", now, now))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO rename_history (subscription_id, original_filename, original_path, renamed_path, ai_prompt, ai_response, status, error)`)).
		WithArgs(int64(2), "番剧 第02集.mp4", from, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), store.RenameHistoryStatusFailed, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sub := store.Subscription{
		ID:              2,
		AIRenameEnabled: true,
		AIRenameSeason:  1,
	}
	fromPath, toPath, skipped, renameErr := svc.renameDownloadFileAt(context.Background(), sub, "第2集", from)
	if renameErr == nil {
		t.Fatalf("renameDownloadFileAt expected error, got nil with from=%q to=%q skipped=%v", fromPath, toPath, skipped)
	}
	if !strings.Contains(renameErr.Error(), "请求 AI 失败") {
		t.Fatalf("rename error = %q, want API connection failure", renameErr)
	}
	if fromPath != from || toPath != "" || skipped {
		t.Fatalf("from=%q to=%q skipped=%v", fromPath, toPath, skipped)
	}
	if _, err := os.Stat(from); err != nil {
		t.Fatalf("original file should remain: %v", err)
	}
	target := filepath.Join(dir, "番剧 第02集 S01E02.mp4")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target should not be created when AI fails, stat err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
