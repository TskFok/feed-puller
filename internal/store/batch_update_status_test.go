package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBatchUpdateItemDownloadStatus_PendingAndSubmitted(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s := &Store{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	itemCols := []string{
		"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(itemCols).AddRow(1, 5, "", "A", "", "http://d/a.mp4", "k1", nil, "submitted", now, now))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs("pending", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(itemCols).AddRow(1, 5, "", "A", "", "http://d/a.mp4", "k1", nil, "pending", now, now))

	got, err := s.BatchUpdateItemDownloadStatus(ctx, []int64{1}, "pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DownloadStatus != "pending" {
		t.Fatalf("got %+v", got)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows(itemCols).AddRow(2, 5, "", "B", "", "", "k2", nil, "pending", now, now))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items SET download_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs("skipped", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows(itemCols).AddRow(2, 5, "", "B", "", "", "k2", nil, "skipped", now, now))

	got, err = s.BatchUpdateItemDownloadStatus(ctx, []int64{2}, "submitted")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DownloadStatus != "skipped" {
		t.Fatalf("got %+v", got)
	}
}

func TestBatchUpdateItemDownloadStatus_RejectSubmitting(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s := &Store{db: db}
	now := time.Now().UTC()
	itemCols := []string{
		"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
	}
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows(itemCols).AddRow(3, 5, "", "C", "", "http://d/c.mp4", "k3", nil, "submitting", now, now))

	_, err = s.BatchUpdateItemDownloadStatus(context.Background(), []int64{3}, "pending")
	if err == nil {
		t.Fatal("expected error for submitting item")
	}
}
