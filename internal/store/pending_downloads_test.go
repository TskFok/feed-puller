package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPendingDownloads_IncludesPreviewStatus(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	query := `
		SELECT i.id, i.subscription_id, i.download_url, sub.download_dir
		FROM feed_items i
		JOIN subscriptions sub ON sub.id = i.subscription_id
		WHERE i.download_status IN ('pending', 'preview', 'failed') AND i.download_url IS NOT NULL AND i.download_url <> ''
		ORDER BY i.id ASC LIMIT ?
	`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "subscription_id", "url", "dir"}).
			AddRow(int64(1), int64(9), "http://d/a.mp4", "/downloads"))

	s := New(db)
	got, err := s.PendingDownloads(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ItemID != 1 {
		t.Fatalf("expected preview/pending items included, got %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
