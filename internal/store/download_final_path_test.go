package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateDownloadTaskFinalPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET final_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)).
		WithArgs("/data/anime/foo S01E02.mp4", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := s.UpdateDownloadTaskFinalPath(context.Background(), 10, "/data/anime/foo S01E02.mp4"); err != nil {
		t.Fatalf("UpdateDownloadTaskFinalPath: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
