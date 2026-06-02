package httpapi

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/store"
)

func TestHandleRenameHistory_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	server := &Server{store: repo}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM rename_history`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM rename_history`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "original_filename", "original_path", "renamed_path", "ai_prompt", "ai_response", "status", "error", "created_at",
		}).AddRow(1, 2, "a.mp4", "/data/a.mp4", "/data/a S01E01.mp4", "prompt", "resp", store.RenameHistoryStatusSuccess, "", time.Now()))

	req := httptest.NewRequest(http.MethodGet, "/api/rename-history?page=1&page_size=20", nil)
	rec := httptest.NewRecorder()
	server.handleRenameHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
