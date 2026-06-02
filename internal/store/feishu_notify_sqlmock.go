package store

import (
	"regexp"
	"strconv"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockFeishuNotifyConfigQueries 为 sqlmock 设置 GetFeishuNotifyConfig 所需的查询期望。
func MockFeishuNotifyConfigQueries(mock sqlmock.Sqlmock, cfg FeishuNotifyConfig) {
	mockFeishuSettingQuery(mock, settingFeishuNotifyType, cfg.NotifyType)
	mockFeishuSettingQuery(mock, settingFeishuBotWebhook, cfg.Webhook)
	mockFeishuSettingQuery(mock, settingFeishuReceiveOpenID, cfg.ReceiveOpenID)
	mockFeishuSettingQuery(mock, settingFeishuReceiveTargets, cfg.ReceiveTargets)
	mockFeishuSettingQuery(mock, settingFeishuCompleteTitle, cfg.CompleteTitleTemplate)
	mockFeishuSettingQuery(mock, settingFeishuFailTitle, cfg.FailTitleTemplate)
	mockFeishuSettingQuery(mock, settingFeishuIncludeSubscription, boolSetting(cfg.IncludeSubscription))
	mockFeishuSettingQuery(mock, settingFeishuIncludeTitle, boolSetting(cfg.IncludeTitle))
	mockFeishuSettingQuery(mock, settingFeishuIncludePath, boolSetting(cfg.IncludePath))
	mockFeishuSettingQuery(mock, settingFeishuNotifyOnFail, boolSetting(cfg.NotifyOnFail))
	mockFeishuSettingQuery(mock, settingFeishuUseInteractiveCard, boolSetting(cfg.UseInteractiveCard))
	mockFeishuSettingQuery(mock, settingFeishuBatchWindowSeconds, feishuBatchSettingValue(cfg))
	mockFeishuSettingQuery(mock, settingFeishuProwlarrCompleteTitle, cfg.ProwlarrCompleteTitleTemplate)
	mockFeishuSettingQuery(mock, settingFeishuProwlarrFailTitle, cfg.ProwlarrFailTitleTemplate)
	mockFeishuSettingQuery(mock, settingFeishuProwlarrCompleteBody, cfg.ProwlarrCompleteBodyTemplate)
	mockFeishuSettingQuery(mock, settingFeishuProwlarrFailBody, cfg.ProwlarrFailBodyTemplate)
}

func mockFeishuSettingQuery(mock sqlmock.Sqlmock, key, value string) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
		WithArgs(key).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value))
}

func feishuBatchSettingValue(cfg FeishuNotifyConfig) string {
	if cfg.BatchWindowSeconds > 0 {
		return strconv.Itoa(cfg.BatchWindowSeconds)
	}
	if cfg.NotifyType != "" || cfg.Webhook != "" {
		return strconv.Itoa(cfg.BatchWindowSeconds)
	}
	return ""
}
