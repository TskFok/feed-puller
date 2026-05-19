package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func TestListActiveDownloadsWithProgress(t *testing.T) {
	aria2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := map[string]any{"status": "active"}
		if req.Method == "aria2.tellStatus" {
			result["completedLength"] = "250"
			result["totalLength"] = "1000"
			result["downloadSpeed"] = "2048"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      "1",
			"result":  result,
		})
	}))
	defer aria2.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	now := time.Now().UTC()
	// SyncAria2DownloadStatus: list submitted
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}))
	// ListActiveDownloads
	mock.ExpectQuery(regexp.QuoteMeta(`WHERE dt.status = 'submitted'`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "subscription_name", "title", "url", "dir", "aria2_gid", "submitted_at",
		}).AddRow(5, 10, 2, "动漫", "测试", "https://example.test/a.mp4", "/data", "gid-5", now))

	rows, err := svc.ListActiveDownloadsWithProgress(context.Background())
	if err != nil {
		t.Fatalf("ListActiveDownloadsWithProgress: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len = %d", len(rows))
	}
	if rows[0].Aria2Status != "active" || rows[0].CompletedLength != 250 {
		t.Fatalf("progress = %+v", rows[0])
	}
	if rows[0].ProgressPercent == nil || *rows[0].ProgressPercent != 25 {
		t.Fatalf("percent = %v", rows[0].ProgressPercent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
