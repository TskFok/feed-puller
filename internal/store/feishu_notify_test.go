package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetFeishuNotifyConfig_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	MockFeishuNotifyConfigQueries(mock, FeishuNotifyConfig{})

	cfg, err := s.GetFeishuNotifyConfig(context.Background())
	if err != nil {
		t.Fatalf("GetFeishuNotifyConfig: %v", err)
	}
	if cfg.Configured {
		t.Fatal("expected not configured")
	}
	if cfg.CompleteTitleTemplate != defaultFeishuCompleteTitle {
		t.Fatalf("complete title = %q", cfg.CompleteTitleTemplate)
	}
	if cfg.ProwlarrCompleteBodyTemplate != defaultFeishuProwlarrCompleteBody {
		t.Fatalf("prowlarr complete body = %q", cfg.ProwlarrCompleteBodyTemplate)
	}
	if cfg.ProwlarrFailBodyTemplate != defaultFeishuProwlarrFailBody {
		t.Fatalf("prowlarr fail body = %q", cfg.ProwlarrFailBodyTemplate)
	}
}

func TestGetFeishuNotifyConfig_Webhook(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	MockFeishuNotifyConfigQueries(mock, FeishuNotifyConfig{
		NotifyType: "webhook",
		Webhook:    "https://open.feishu.cn/hook/test",
	})

	cfg, err := s.GetFeishuNotifyConfig(context.Background())
	if err != nil {
		t.Fatalf("GetFeishuNotifyConfig: %v", err)
	}
	if cfg.NotifyType != "webhook" {
		t.Fatalf("notify_type = %q", cfg.NotifyType)
	}
	if !cfg.Configured {
		t.Fatal("expected configured")
	}
}

func TestSaveFeishuNotifyConfig_InvalidType(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	err = s.SaveFeishuNotifyConfig(context.Background(), FeishuNotifyConfig{NotifyType: "invalid"})
	if !IsInvalidFeishuNotifyType(err) {
		t.Fatalf("expected invalid type error, got %v", err)
	}
}

func TestSaveFeishuNotifyConfig_InvalidBatchWindow(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	err = s.SaveFeishuNotifyConfig(context.Background(), FeishuNotifyConfig{NotifyType: "webhook", BatchWindowSeconds: 999})
	if !IsInvalidFeishuBatchWindow(err) {
		t.Fatalf("expected batch window error, got %v", err)
	}
}

func TestSaveFeishuNotifyConfig_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	for _, args := range [][2]string{
		{settingFeishuNotifyType, "api"},
		{settingFeishuBotWebhook, ""},
		{settingFeishuReceiveOpenID, "ou_test"},
		{settingFeishuReceiveTargets, "chat_id:oc_group"},
		{settingFeishuCompleteTitle, "[完成]"},
		{settingFeishuFailTitle, "[失败]"},
		{settingFeishuProwlarrCompleteTitle, "[Prowlarr 完成]"},
		{settingFeishuProwlarrFailTitle, "[Prowlarr 失败]"},
		{settingFeishuProwlarrCompleteBody, "**类型**: {{media_type}}"},
		{settingFeishuProwlarrFailBody, "**错误**: {{error}}"},
		{settingFeishuIncludeSubscription, "true"},
		{settingFeishuIncludeTitle, "true"},
		{settingFeishuIncludePath, "false"},
		{settingFeishuNotifyOnFail, "true"},
		{settingFeishuUseInteractiveCard, "true"},
		{settingFeishuBatchWindowSeconds, "15"},
	} {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`)).
			WithArgs(args[0], args[1]).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	if err := s.SaveFeishuNotifyConfig(context.Background(), FeishuNotifyConfig{
		NotifyType:            "api",
		ReceiveOpenID:         "ou_test",
		ReceiveTargets:        "chat_id:oc_group",
		CompleteTitleTemplate:         "[完成]",
		FailTitleTemplate:             "[失败]",
		ProwlarrCompleteTitleTemplate: "[Prowlarr 完成]",
		ProwlarrFailTitleTemplate:     "[Prowlarr 失败]",
		ProwlarrCompleteBodyTemplate:  "**类型**: {{media_type}}",
		ProwlarrFailBodyTemplate:      "**错误**: {{error}}",
		IncludeSubscription:   true,
		IncludeTitle:          true,
		IncludePath:           false,
		NotifyOnFail:          true,
		UseInteractiveCard:    true,
		BatchWindowSeconds:    15,
	}); err != nil {
		t.Fatalf("SaveFeishuNotifyConfig: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
