package app

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/feishu"
	"feed-puller/internal/store"
)

type mockFeishuBot struct {
	lastWebhook  string
	lastOpenID   string
	lastTitle    string
	lastContent  string
	lastCard     feishu.InteractiveCard
	useCard      bool
	recipientCnt int
	err          error
}

func (m *mockFeishuBot) SendText(webhook string, title string, content string) error {
	m.lastWebhook = webhook
	m.lastTitle = title
	m.lastContent = content
	m.useCard = false
	return m.err
}

func (m *mockFeishuBot) SendInteractiveWebhook(webhook string, card feishu.InteractiveCard) error {
	m.lastWebhook = webhook
	m.lastCard = card
	m.useCard = true
	return m.err
}

func (m *mockFeishuBot) SendToUserByOpenID(openID string, title string, content string) error {
	m.lastOpenID = openID
	m.lastTitle = title
	m.lastContent = content
	return m.err
}

func (m *mockFeishuBot) SendInteractiveViaAPI(receiveIDType, receiveID string, card feishu.InteractiveCard) error {
	m.lastOpenID = receiveID
	m.lastCard = card
	m.useCard = true
	return m.err
}

func (m *mockFeishuBot) SendToRecipients(recipients []feishu.Recipient, card feishu.InteractiveCard, title, content string, useCard bool) error {
	m.recipientCnt = len(recipients)
	m.lastTitle = title
	m.lastContent = content
	m.lastCard = card
	m.useCard = useCard
	return m.err
}

func expectFeishuNotifyConfigQueries(mock sqlmock.Sqlmock, cfg store.FeishuNotifyConfig) {
	store.MockFeishuNotifyConfigQueries(mock, cfg)
}

func expectFeishuNotifyHistoryInsert(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO feishu_notify_history (event_type, source, notify_type, title, content, item_count, status, error)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestNotifyDownloadComplete_WebhookCard(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:         "webhook",
		Webhook:            "https://hook.test",
		UseInteractiveCard: true,
		BatchWindowSeconds: 0,
	})
	expectFeishuNotifyHistoryInsert(mock)

	svc.notifyDownloadComplete(context.Background(), store.Subscription{Name: "动漫"}, store.Item{ID: 1, Title: "第1话"}, "/data/a.mp4")

	if bot.lastWebhook != "https://hook.test" || !bot.useCard {
		t.Fatalf("webhook=%q useCard=%v", bot.lastWebhook, bot.useCard)
	}
	if bot.lastCard.Title == "" {
		t.Fatal("expected card title")
	}
}

func TestNotifyDownloadComplete_Disabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{})

	svc.notifyDownloadComplete(context.Background(), store.Subscription{}, store.Item{}, "")

	if bot.lastWebhook != "" || bot.lastOpenID != "" {
		t.Fatal("expected no send")
	}
}

func TestNotifyDownloadFail_Webhook(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:         "webhook",
		Webhook:            "https://hook.test",
		NotifyOnFail:       true,
		UseInteractiveCard: true,
		BatchWindowSeconds: 0,
	})
	expectFeishuNotifyHistoryInsert(mock)

	svc.notifyDownloadFail(context.Background(), store.Subscription{Name: "动漫"}, store.Item{Title: "第1话"}, "network error")

	if bot.lastCard.Template != "red" {
		t.Fatalf("template = %q", bot.lastCard.Template)
	}
}

func TestFeishuBatchNotify(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	cfg := store.FeishuNotifyConfig{
		NotifyType:         "webhook",
		Webhook:            "https://hook.test",
		UseInteractiveCard: true,
		BatchWindowSeconds: 60,
	}
	expectFeishuNotifyConfigQueries(mock, cfg)
	expectFeishuNotifyConfigQueries(mock, cfg)
	expectFeishuNotifyConfigQueries(mock, cfg)
	expectFeishuNotifyHistoryInsert(mock)

	svc.queueFeishuNotifyComplete(context.Background(), store.Subscription{Name: "A"}, store.Item{Title: "1"}, "/a")
	svc.queueFeishuNotifyComplete(context.Background(), store.Subscription{Name: "B"}, store.Item{Title: "2"}, "/b")
	svc.FlushFeishuBatchForTest()

	if bot.lastCard.Title == "" {
		t.Fatal("expected batch card")
	}
}

func TestSendFeishuTestNotification_NoBot(t *testing.T) {
	svc := NewService(nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	err := svc.SendFeishuTestNotification(context.Background(), store.FeishuNotifyConfig{NotifyType: "webhook", Webhook: "https://x"})
	if err == nil || err.Error() != "飞书机器人服务不可用" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendFeishuTestNotification_APIRequiresRecipient(t *testing.T) {
	svc := NewService(nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	svc.SetFeishuBot(&mockFeishuBot{})
	err := svc.SendFeishuTestNotification(context.Background(), store.FeishuNotifyConfig{NotifyType: "api"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNotifyDownloadComplete_MultiRecipients(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:         "api",
		ReceiveOpenID:      "ou_a",
		ReceiveTargets:     "chat_id:oc_b",
		UseInteractiveCard: true,
		BatchWindowSeconds: 0,
	})
	expectFeishuNotifyHistoryInsert(mock)

	svc.notifyDownloadComplete(context.Background(), store.Subscription{Name: "动漫"}, store.Item{Title: "第1话"}, "/data/a.mp4")
	if bot.recipientCnt != 2 {
		t.Fatalf("recipientCnt = %d", bot.recipientCnt)
	}
}

func TestNotifyProwlarrComplete_UsesProwlarrTemplate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:                    "webhook",
		Webhook:                       "https://hook.test",
		ProwlarrCompleteTitleTemplate: "[Prowlarr 完成] {{media_type}}",
		UseInteractiveCard:            true,
		BatchWindowSeconds:            0,
	})
	expectFeishuNotifyHistoryInsert(mock)

	sub := store.Subscription{Name: "Prowlarr 电影", FeedURL: store.ProwlarrInternalFeedURLMovie}
	svc.notifyDownloadComplete(context.Background(), sub, store.Item{Title: "Inception (2010)"}, "/movies/inception.mkv")

	if !strings.Contains(bot.lastCard.Title, "电影") {
		t.Fatalf("title = %q", bot.lastCard.Title)
	}
	bodyText := strings.Join(bot.lastCard.Lines, "\n")
	if !strings.Contains(bodyText, "电影") || !strings.Contains(bodyText, "Inception") || !strings.Contains(bodyText, "/movies/inception.mkv") {
		t.Fatalf("card lines = %v", bot.lastCard.Lines)
	}
}

func TestNotifyProwlarrComplete_UsesCustomBodyTemplate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := store.New(db)
	svc := NewService(repo, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	bot := &mockFeishuBot{}
	svc.SetFeishuBot(bot)

	expectFeishuNotifyConfigQueries(mock, store.FeishuNotifyConfig{
		NotifyType:                   "webhook",
		Webhook:                      "https://hook.test",
		ProwlarrCompleteBodyTemplate: "已下载 {{title}} → {{path}}",
		UseInteractiveCard:           true,
		BatchWindowSeconds:           0,
	})
	expectFeishuNotifyHistoryInsert(mock)

	sub := store.Subscription{Name: "Prowlarr 电影", FeedURL: store.ProwlarrInternalFeedURLMovie}
	svc.notifyDownloadComplete(context.Background(), sub, store.Item{Title: "Inception (2010)"}, "/movies/inception.mkv")

	if len(bot.lastCard.Lines) != 1 || !strings.Contains(bot.lastCard.Lines[0], "Inception") || !strings.Contains(bot.lastCard.Lines[0], "/movies/inception.mkv") {
		t.Fatalf("card lines = %v", bot.lastCard.Lines)
	}
}
