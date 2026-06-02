package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateRenameHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO rename_history (subscription_id, original_filename, original_path, renamed_path, ai_prompt, ai_response, status, error)`)).
		WithArgs(int64(2), "番剧 第02话.mp4", "/data/anime/番剧 第02话.mp4", "/data/anime/番剧 第02话 S01E02.mp4", "prompt", `{"episode":2}`, RenameHistoryStatusSuccess, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = s.CreateRenameHistory(context.Background(), RenameHistory{
		SubscriptionID:   2,
		OriginalFilename: "番剧 第02话.mp4",
		OriginalPath:     "/data/anime/番剧 第02话.mp4",
		RenamedPath:      "/data/anime/番剧 第02话 S01E02.mp4",
		AIPrompt:         "prompt",
		AIResponse:       `{"episode":2}`,
		Status:           RenameHistoryStatusSuccess,
	})
	if err != nil {
		t.Fatalf("CreateRenameHistory: %v", err)
	}
}

func TestListRenameHistoryPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM rename_history`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM rename_history`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "original_filename", "original_path", "renamed_path", "ai_prompt", "ai_response", "status", "error", "created_at",
		}).AddRow(1, 2, "a.mp4", "/data/a.mp4", "/data/a S01E01.mp4", "prompt", "resp", RenameHistoryStatusSuccess, "", now))

	rows, total, err := s.ListRenameHistoryPage(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListRenameHistoryPage: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("total=%d len=%d", total, len(rows))
	}
	if rows[0].OriginalFilename != "a.mp4" || rows[0].Status != RenameHistoryStatusSuccess {
		t.Fatalf("row = %+v", rows[0])
	}
}
