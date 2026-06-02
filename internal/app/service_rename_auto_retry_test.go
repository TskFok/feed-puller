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

func TestNextRenameRetryAt(t *testing.T) {
	failedAt := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		completed int
		want      time.Time
		ok        bool
	}{
		{0, failedAt.Add(time.Minute), true},
		{1, failedAt.Add(5 * time.Minute), true},
		{2, failedAt.Add(10 * time.Minute), true},
		{3, time.Time{}, false},
	}
	for _, tc := range cases {
		got, ok := nextRenameRetryAt(failedAt, tc.completed)
		if ok != tc.ok {
			t.Fatalf("completed=%d ok=%v want=%v", tc.completed, ok, tc.ok)
		}
		if ok && !got.Equal(tc.want) {
			t.Fatalf("completed=%d got=%v want=%v", tc.completed, got, tc.want)
		}
	}
}

func TestScheduleRenameRetry_EnqueuesRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client("", ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT IGNORE INTO rename_retries (task_id, file_path, retry_count, failed_at, next_retry_at, last_error, status)`)).
		WithArgs(int64(7), "/data/anime/foo.mp4", 0, sqlmock.AnyArg(), sqlmock.AnyArg(), "识别失败", store.RenameRetryStatusPending).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc.scheduleRenameRetry(context.Background(), 7, "/data/anime/foo.mp4", "识别失败")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessDueRenameRetries_Success(t *testing.T) {
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
	failedAt := now.Add(-2 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rename_retries`)).
		WithArgs(store.RenameRetryStatusPending, sqlmock.AnyArg(), 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_id", "file_path", "retry_count", "failed_at", "next_retry_at", "last_error", "status", "created_at", "updated_at",
		}).AddRow(1, 9, from, 0, failedAt, now.Add(-time.Minute), "识别失败", store.RenameRetryStatusPending, failedAt, failedAt))

	expectDownloadTaskQuery(mock, 9, 90, 2, dir, now)
	expectSubscriptionQuery(mock, 2, dir, true, now)
	expectFeedItemQuery(mock, 90, 2, "第2话", now)
	expectAIConfigQueries(mock, ai.URL, now)

	target := filepath.Join(dir, "番剧 第02话 S01E02.mp4")
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET final_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs(target, int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rename_retries`)).
		WithArgs(0, "", store.RenameRetryStatusSucceeded, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.ProcessDueRenameRetries(context.Background()); err != nil {
		t.Fatalf("ProcessDueRenameRetries: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessDueRenameRetries_ExhaustedNotifiesFeishu(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "bad name.mp4")
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
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)
	now := time.Now().UTC()
	failedAt := now.Add(-11 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rename_retries`)).
		WithArgs(store.RenameRetryStatusPending, sqlmock.AnyArg(), 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_id", "file_path", "retry_count", "failed_at", "next_retry_at", "last_error", "status", "created_at", "updated_at",
		}).AddRow(1, 9, from, 2, failedAt, now.Add(-time.Minute), "旧错误", store.RenameRetryStatusPending, failedAt, failedAt))

	expectDownloadTaskQuery(mock, 9, 90, 2, dir, now)
	expectSubscriptionQuery(mock, 2, dir, true, now)
	expectFeedItemQuery(mock, 90, 2, "第2话", now)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}))

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rename_retries`)).
		WithArgs(3, sqlmock.AnyArg(), store.RenameRetryStatusAbandoned, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	cfg := store.FeishuNotifyConfig{
		NotifyType:            "webhook",
		Webhook:               "https://example.test/hook",
		NotifyOnFail:          true,
		UseInteractiveCard:    true,
		IncludeSubscription:   true,
		IncludeTitle:          true,
		IncludePath:           true,
		FailTitleTemplate:     "下载失败",
		CompleteTitleTemplate: "下载完成",
	}
	expectFeishuNotifyConfigQueries(mock, cfg)
	expectFeishuNotifyHistoryInsert(mock)

	if err := svc.ProcessDueRenameRetries(context.Background()); err != nil {
		t.Fatalf("ProcessDueRenameRetries: %v", err)
	}
	if bot.lastCard.Title == "" {
		t.Fatal("expected feishu notification")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectDownloadTaskQuery(mock sqlmock.Sqlmock, taskID, itemID, subID int64, dir string, now time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE id = ?`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "final_path", "created_at", "updated_at",
		}).AddRow(taskID, itemID, subID, "magnet:?xt=urn:btih:abc", dir, "completed", "", "", "", now, now))
}

func expectSubscriptionQuery(mock sqlmock.Sqlmock, subID int64, dir string, aiRename bool, now time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`FROM subscriptions WHERE id = ?`)).
		WithArgs(subID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "feed_url", "enabled", "poll_interval_minutes", "poll_cron", "poll_cron_timezone",
			"download_dir", "include_keywords", "exclude_keywords", "use_proxy", "rss_parser",
			"ai_rename_enabled", "ai_rename_season", "ai_rename_episode_offset",
			"last_fetched_at", "last_error", "sort_order", "created_at", "updated_at",
		}).AddRow(subID, "动漫", "https://example.test/feed", true, 30, "", "UTC", dir, "", "", false, "mikan", aiRename, 1, 0, nil, "", 0, now, now))
}

func expectFeedItemQuery(mock sqlmock.Sqlmock, itemID, subID int64, title string, now time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(itemID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(itemID, subID, "", title, "", "magnet:?xt=urn:btih:abc", "k", nil, "completed", now, now))
}

func expectAIConfigQueries(mock sqlmock.Sqlmock, aiURL string, now time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM ai_configs`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ai_configs ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(1, "test", aiURL+"/v1", "gpt-test", "sk-test", now, now))
}
