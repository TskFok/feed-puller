package app

import (
	"context"
	"database/sql"
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

func TestTellStatusForDownloadTask_UpdatesGIDOnFollowedBy(t *testing.T) {
	call := 0
	aria2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		var req struct {
			Params []any `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gid, _ := req.Params[len(req.Params)-1].(string)
		result := map[string]any{"status": "active", "completedLength": "0", "totalLength": "0"}
		if gid == "meta-gid" {
			result["followedBy"] = []any{"real-gid"}
			result["files"] = []any{
				map[string]any{"path": "/data/[METADATA]x.mp4", "completedLength": "1", "length": "1", "selected": "true"},
			}
		}
		if gid == "real-gid" {
			result["completedLength"] = "500"
			result["totalLength"] = "1000"
			result["files"] = []any{
				map[string]any{"path": "/data/show E07.mp4", "completedLength": "500", "length": "1000", "selected": "true"},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": "1", "result": result})
	}))
	defer aria2.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, downloader.NewAria2Client(aria2.URL, ""), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET aria2_gid = ?`)).
		WithArgs("real-gid", int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	effective, status, err := svc.tellStatusForDownloadTask(context.Background(), store.DownloadTask{
		ID: 5, ItemID: 50, Aria2GID: "meta-gid",
	})
	if err != nil {
		t.Fatalf("tellStatusForDownloadTask: %v", err)
	}
	if effective != "real-gid" {
		t.Fatalf("effective = %q", effective)
	}
	if call < 2 {
		t.Fatalf("expected >=2 tellStatus calls, got %d", call)
	}
	progress := downloader.ParseAria2Progress(status)
	if progress.TotalLength != 1000 || progress.CompletedLength != 500 {
		t.Fatalf("progress = %+v", progress)
	}
	if progress.ProgressPercent == nil || *progress.ProgressPercent != 50 {
		t.Fatalf("percent = %v", progress.ProgressPercent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindDownloadTaskForAria2Hook_ViaFollowing(t *testing.T) {
	aria2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params []any `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gid, _ := req.Params[len(req.Params)-1].(string)
		result := map[string]any{"status": "active", "following": "meta-gid"}
		if gid == "real-gid" {
			_ = result
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": "1", "result": result})
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

	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("real-gid").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`FROM download_tasks WHERE aria2_gid = ?`)).
		WithArgs("meta-gid").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "item_id", "subscription_id", "url", "dir", "status", "aria2_gid", "error", "created_at", "updated_at",
		}).AddRow(6, 60, 2, "magnet:?xt=urn:btih:abc", "/data", "submitted", "meta-gid", "", now, now))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE download_tasks SET aria2_gid = ?`)).
		WithArgs("real-gid", int64(6)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	task, err := svc.findDownloadTaskForAria2Hook(context.Background(), "real-gid")
	if err != nil {
		t.Fatalf("findDownloadTaskForAria2Hook: %v", err)
	}
	if task.ID != 6 || task.Aria2GID != "meta-gid" {
		t.Fatalf("task = %+v", task)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
