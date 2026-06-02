package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateFeishuNotifyHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO feishu_notify_history (event_type, source, notify_type, title, content, item_count, status, error)`)).
		WithArgs("complete", "prowlarr", "webhook", "标题", "正文", 2, "sent", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = s.CreateFeishuNotifyHistory(context.Background(), FeishuNotifyHistory{
		EventType:  "complete",
		Source:     "prowlarr",
		NotifyType: "webhook",
		Title:      "标题",
		Content:    "正文",
		ItemCount:  2,
		Status:     "sent",
	})
	if err != nil {
		t.Fatalf("CreateFeishuNotifyHistory: %v", err)
	}
}

func TestListFeishuNotifyHistoryPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM feishu_notify_history`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feishu_notify_history`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "event_type", "source", "notify_type", "title", "content", "item_count", "status", "error", "created_at",
		}).AddRow(1, "fail", "rss", "api", "失败", "内容", 1, "failed", "timeout", now))

	rows, total, err := s.ListFeishuNotifyHistoryPage(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListFeishuNotifyHistoryPage: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("total=%d len=%d", total, len(rows))
	}
	if rows[0].Source != "rss" || rows[0].Status != "failed" {
		t.Fatalf("row = %+v", rows[0])
	}
}
