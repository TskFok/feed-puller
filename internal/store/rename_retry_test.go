package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEnqueueRenameRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	failedAt := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	nextAt := failedAt.Add(time.Minute)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT IGNORE INTO rename_retries (task_id, file_path, retry_count, failed_at, next_retry_at, last_error, status)`)).
		WithArgs(int64(9), "/data/anime/foo.mp4", 0, failedAt, nextAt, "识别失败", RenameRetryStatusPending).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := s.EnqueueRenameRetry(context.Background(), RenameRetry{
		TaskID:      9,
		FilePath:    "/data/anime/foo.mp4",
		RetryCount:  0,
		FailedAt:    failedAt,
		NextRetryAt: nextAt,
		LastError:   "识别失败",
	}); err != nil {
		t.Fatalf("EnqueueRenameRetry: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListDueRenameRetries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	now := time.Date(2026, 6, 2, 10, 5, 0, 0, time.UTC)
	failedAt := now.Add(-5 * time.Minute)
	nextAt := now.Add(-time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rename_retries`)).
		WithArgs(RenameRetryStatusPending, now, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_id", "file_path", "retry_count", "failed_at", "next_retry_at", "last_error", "status", "created_at", "updated_at",
		}).AddRow(1, 9, "/data/anime/foo.mp4", 0, failedAt, nextAt, "识别失败", RenameRetryStatusPending, failedAt, failedAt))

	rows, err := s.ListDueRenameRetries(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("ListDueRenameRetries: %v", err)
	}
	if len(rows) != 1 || rows[0].TaskID != 9 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRenameRetryAfterAttempt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	nextAt := time.Date(2026, 6, 2, 10, 5, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rename_retries`)).
		WithArgs(1, "仍然失败", nextAt, RenameRetryStatusPending, int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := s.UpdateRenameRetryAfterAttempt(context.Background(), 3, 1, "仍然失败", &nextAt, RenameRetryStatusPending); err != nil {
		t.Fatalf("UpdateRenameRetryAfterAttempt: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkRenameRetrySucceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rename_retries`)).
		WithArgs(RenameRetryStatusSucceeded, int64(9), RenameRetryStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := s.MarkRenameRetrySucceeded(context.Background(), 9); err != nil {
		t.Fatalf("MarkRenameRetrySucceeded: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
