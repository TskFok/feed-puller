package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListCompletedDownloadsPage_IncludesFinalPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	ctx := context.Background()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM download_tasks WHERE status = 'completed'`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks dt`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "subscription_name", "title", "url", "dir", "final_path", "ai_rename_enabled", "completed_at",
		}).AddRow(1, 10, 2, "动漫", "第1话", "https://example.test/a.mp4", "/data/anime", "/data/anime/番剧 S01E01.mp4", true, now))

	rows, total, err := s.ListCompletedDownloadsPage(ctx, 1, 20)
	if err != nil {
		t.Fatalf("ListCompletedDownloadsPage: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].FinalPath != "/data/anime/番剧 S01E01.mp4" {
		t.Fatalf("final_path = %q", rows[0].FinalPath)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
