package httpapi

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/feishu"
	"feed-puller/internal/store"
)

type notifyTestBot struct{}

func (notifyTestBot) SendText(webhook string, title string, content string) error { return nil }
func (notifyTestBot) SendInteractiveWebhook(webhook string, card feishu.InteractiveCard) error {
	return nil
}
func (notifyTestBot) SendToUserByOpenID(openID string, title string, content string) error { return nil }
func (notifyTestBot) SendInteractiveViaAPI(receiveIDType, receiveID string, card feishu.InteractiveCard) error {
	return nil
}
func (notifyTestBot) SendToRecipients(recipients []feishu.Recipient, card feishu.InteractiveCard, title, content string, useCard bool) error {
	return nil
}

func TestHandleFeishuNotifySetting_GetPut(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := app.NewService(repo, nil, nil)
	svc.SetFeishuBot(notifyTestBot{})
	server := &Server{cfg: config.Config{}, store: repo, service: svc}

	store.MockFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType: "webhook",
		Webhook:    "https://hook.test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/settings/feishu-notify", nil)
	rec := httptest.NewRecorder()
	server.handleFeishuNotifySetting(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", rec.Code, rec.Body.String())
	}

	for _, args := range [][2]string{
		{"feishu_notify_type", "webhook"},
		{"feishu_bot_webhook", "https://hook.new"},
		{"feishu_notify_receive_open_id", ""},
		{"feishu_notify_receive_targets", "chat_id:oc_test"},
		{"feishu_notify_complete_title", "[完成]"},
		{"feishu_notify_fail_title", "[失败]"},
		{"feishu_prowlarr_complete_title", "[Prowlarr 完成]"},
		{"feishu_prowlarr_fail_title", "[Prowlarr 失败]"},
		{"feishu_prowlarr_complete_body", "**类型**: {{media_type}}\n**标题**: {{title}}\n**路径**: {{path}}"},
		{"feishu_prowlarr_fail_body", "**类型**: {{media_type}}\n**标题**: {{title}}\n**错误**: {{error}}"},
		{"feishu_notify_include_subscription", "true"},
		{"feishu_notify_include_title", "true"},
		{"feishu_notify_include_path", "true"},
		{"feishu_notify_on_fail", "true"},
		{"feishu_notify_use_card", "true"},
		{"feishu_notify_batch_seconds", "30"},
	} {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`)).
			WithArgs(args[0], args[1]).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	store.MockFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:                    "webhook",
		Webhook:                       "https://hook.new",
		ReceiveTargets:                "chat_id:oc_test",
		CompleteTitleTemplate:         "[完成]",
		FailTitleTemplate:             "[失败]",
		ProwlarrCompleteTitleTemplate: "[Prowlarr 完成]",
		ProwlarrFailTitleTemplate:     "[Prowlarr 失败]",
		ProwlarrCompleteBodyTemplate:  "**类型**: {{media_type}}\n**标题**: {{title}}\n**路径**: {{path}}",
		ProwlarrFailBodyTemplate:      "**类型**: {{media_type}}\n**标题**: {{title}}\n**错误**: {{error}}",
	})

	body := `{"feishu_notify_type":"webhook","feishu_bot_webhook":"https://hook.new","feishu_receive_targets":"chat_id:oc_test","feishu_complete_title":"[完成]","feishu_fail_title":"[失败]","feishu_prowlarr_complete_title":"[Prowlarr 完成]","feishu_prowlarr_fail_title":"[Prowlarr 失败]","feishu_prowlarr_complete_body":"**类型**: {{media_type}}\n**标题**: {{title}}\n**路径**: {{path}}","feishu_prowlarr_fail_body":"**类型**: {{media_type}}\n**标题**: {{title}}\n**错误**: {{error}}","feishu_include_subscription":true,"feishu_include_title":true,"feishu_include_path":true,"feishu_notify_on_fail":true,"feishu_use_interactive_card":true,"feishu_batch_window_seconds":30}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/settings/feishu-notify", strings.NewReader(body))
	putRec := httptest.NewRecorder()
	server.handleFeishuNotifySetting(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}
}

func TestHandleFeishuNotifyTest_RequiresConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := app.NewService(repo, nil, nil)
	svc.SetFeishuBot(notifyTestBot{})
	server := &Server{cfg: config.Config{}, store: repo, service: svc}

	store.MockFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/settings/feishu-notify/test", nil)
	rec := httptest.NewRecorder()
	server.handleFeishuNotifyTest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleFeishuNotifyHistory_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	server := &Server{store: repo}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM feishu_notify_history`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feishu_notify_history`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "event_type", "source", "notify_type", "title", "content", "item_count", "status", "error", "created_at",
		}).AddRow(1, "complete", "rss", "webhook", "标题", "内容", 1, "sent", "", time.Now()))

	req := httptest.NewRequest(http.MethodGet, "/api/feishu-notify/history?page=1&page_size=20", nil)
	rec := httptest.NewRecorder()
	server.handleFeishuNotifyHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
