package store

import (
	"context"
	"strconv"
	"strings"
)

const (
	settingFeishuNotifyType            = "feishu_notify_type"
	settingFeishuBotWebhook            = "feishu_bot_webhook"
	settingFeishuReceiveOpenID         = "feishu_notify_receive_open_id"
	settingFeishuReceiveTargets        = "feishu_notify_receive_targets"
	settingFeishuCompleteTitle         = "feishu_notify_complete_title"
	settingFeishuFailTitle             = "feishu_notify_fail_title"
	settingFeishuIncludeSubscription   = "feishu_notify_include_subscription"
	settingFeishuIncludeTitle          = "feishu_notify_include_title"
	settingFeishuIncludePath           = "feishu_notify_include_path"
	settingFeishuNotifyOnFail          = "feishu_notify_on_fail"
	settingFeishuUseInteractiveCard    = "feishu_notify_use_card"
	settingFeishuBatchWindowSeconds    = "feishu_notify_batch_seconds"
	settingFeishuProwlarrCompleteTitle = "feishu_prowlarr_complete_title"
	settingFeishuProwlarrFailTitle     = "feishu_prowlarr_fail_title"
	settingFeishuProwlarrCompleteBody  = "feishu_prowlarr_complete_body"
	settingFeishuProwlarrFailBody      = "feishu_prowlarr_fail_body"

	defaultFeishuCompleteTitle           = "[Feed Puller 下载完成]"
	defaultFeishuFailTitle               = "[Feed Puller 下载失败]"
	defaultFeishuProwlarrCompleteTitle  = "[Feed Puller Prowlarr 下载完成]"
	defaultFeishuProwlarrFailTitle       = "[Feed Puller Prowlarr 下载失败]"
	defaultFeishuProwlarrCompleteBody   = "**类型**: {{media_type}}\n**标题**: {{title}}\n**路径**: {{path}}"
	defaultFeishuProwlarrFailBody        = "**类型**: {{media_type}}\n**标题**: {{title}}\n**错误**: {{error}}"
	defaultFeishuBatchWindowSeconds      = 30
)

// FeishuNotifyConfig 飞书下载通知配置。
type FeishuNotifyConfig struct {
	NotifyType            string `json:"feishu_notify_type"`
	Webhook               string `json:"feishu_bot_webhook"`
	ReceiveOpenID         string `json:"feishu_receive_open_id"`
	ReceiveTargets        string `json:"feishu_receive_targets"`
	CompleteTitleTemplate         string `json:"feishu_complete_title"`
	FailTitleTemplate             string `json:"feishu_fail_title"`
	ProwlarrCompleteTitleTemplate string `json:"feishu_prowlarr_complete_title"`
	ProwlarrFailTitleTemplate     string `json:"feishu_prowlarr_fail_title"`
	ProwlarrCompleteBodyTemplate  string `json:"feishu_prowlarr_complete_body"`
	ProwlarrFailBodyTemplate      string `json:"feishu_prowlarr_fail_body"`
	IncludeSubscription   bool   `json:"feishu_include_subscription"`
	IncludeTitle          bool   `json:"feishu_include_title"`
	IncludePath           bool   `json:"feishu_include_path"`
	NotifyOnFail          bool   `json:"feishu_notify_on_fail"`
	UseInteractiveCard    bool   `json:"feishu_use_interactive_card"`
	BatchWindowSeconds    int    `json:"feishu_batch_window_seconds"`
	Configured            bool   `json:"configured"`
}

// GetFeishuNotifyConfig 读取飞书通知配置。
func (s *Store) GetFeishuNotifyConfig(ctx context.Context) (FeishuNotifyConfig, error) {
	read := func(key string) (string, error) {
		return s.GetSetting(ctx, key)
	}
	notifyType, err := read(settingFeishuNotifyType)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	webhook, err := read(settingFeishuBotWebhook)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	receiveOpenID, err := read(settingFeishuReceiveOpenID)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	receiveTargets, err := read(settingFeishuReceiveTargets)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	completeTitle, err := read(settingFeishuCompleteTitle)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	failTitle, err := read(settingFeishuFailTitle)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	includeSub, err := read(settingFeishuIncludeSubscription)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	includeTitle, err := read(settingFeishuIncludeTitle)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	includePath, err := read(settingFeishuIncludePath)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	notifyOnFail, err := read(settingFeishuNotifyOnFail)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	useCard, err := read(settingFeishuUseInteractiveCard)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	batchSeconds, err := read(settingFeishuBatchWindowSeconds)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	prowlarrCompleteTitle, err := read(settingFeishuProwlarrCompleteTitle)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	prowlarrFailTitle, err := read(settingFeishuProwlarrFailTitle)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	prowlarrCompleteBody, err := read(settingFeishuProwlarrCompleteBody)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}
	prowlarrFailBody, err := read(settingFeishuProwlarrFailBody)
	if err != nil {
		return FeishuNotifyConfig{}, err
	}

	cfg := FeishuNotifyConfig{
		NotifyType:            strings.TrimSpace(notifyType),
		Webhook:               strings.TrimSpace(webhook),
		ReceiveOpenID:         strings.TrimSpace(receiveOpenID),
		ReceiveTargets:        strings.TrimSpace(receiveTargets),
		CompleteTitleTemplate: strings.TrimSpace(completeTitle),
		FailTitleTemplate:     strings.TrimSpace(failTitle),
		IncludeSubscription:   parseBoolSetting(includeSub, true),
		IncludeTitle:          parseBoolSetting(includeTitle, true),
		IncludePath:           parseBoolSetting(includePath, true),
		NotifyOnFail:          parseBoolSetting(notifyOnFail, true),
		UseInteractiveCard:    parseBoolSetting(useCard, true),
		BatchWindowSeconds:            parseIntSetting(batchSeconds, defaultFeishuBatchWindowSeconds),
		ProwlarrCompleteTitleTemplate: strings.TrimSpace(prowlarrCompleteTitle),
		ProwlarrFailTitleTemplate:     strings.TrimSpace(prowlarrFailTitle),
		ProwlarrCompleteBodyTemplate:  strings.TrimSpace(prowlarrCompleteBody),
		ProwlarrFailBodyTemplate:      strings.TrimSpace(prowlarrFailBody),
	}
	if cfg.CompleteTitleTemplate == "" {
		cfg.CompleteTitleTemplate = defaultFeishuCompleteTitle
	}
	if cfg.FailTitleTemplate == "" {
		cfg.FailTitleTemplate = defaultFeishuFailTitle
	}
	if cfg.ProwlarrCompleteTitleTemplate == "" {
		cfg.ProwlarrCompleteTitleTemplate = defaultFeishuProwlarrCompleteTitle
	}
	if cfg.ProwlarrFailTitleTemplate == "" {
		cfg.ProwlarrFailTitleTemplate = defaultFeishuProwlarrFailTitle
	}
	if cfg.ProwlarrCompleteBodyTemplate == "" {
		cfg.ProwlarrCompleteBodyTemplate = defaultFeishuProwlarrCompleteBody
	}
	if cfg.ProwlarrFailBodyTemplate == "" {
		cfg.ProwlarrFailBodyTemplate = defaultFeishuProwlarrFailBody
	}
	if cfg.NotifyType == "" && cfg.Webhook != "" {
		cfg.NotifyType = "webhook"
	}
	cfg.Configured = cfg.NotifyType != ""
	return cfg, nil
}

// SaveFeishuNotifyConfig 保存飞书通知配置。
func (s *Store) SaveFeishuNotifyConfig(ctx context.Context, cfg FeishuNotifyConfig) error {
	notifyType := strings.TrimSpace(cfg.NotifyType)
	webhook := strings.TrimSpace(cfg.Webhook)
	receiveOpenID := strings.TrimSpace(cfg.ReceiveOpenID)
	receiveTargets := strings.TrimSpace(cfg.ReceiveTargets)
	completeTitle := strings.TrimSpace(cfg.CompleteTitleTemplate)
	failTitle := strings.TrimSpace(cfg.FailTitleTemplate)
	prowlarrCompleteTitle := strings.TrimSpace(cfg.ProwlarrCompleteTitleTemplate)
	prowlarrFailTitle := strings.TrimSpace(cfg.ProwlarrFailTitleTemplate)
	prowlarrCompleteBody := strings.TrimSpace(cfg.ProwlarrCompleteBodyTemplate)
	prowlarrFailBody := strings.TrimSpace(cfg.ProwlarrFailBodyTemplate)
	if notifyType != "" && notifyType != "webhook" && notifyType != "api" {
		return errInvalidFeishuNotifyType
	}
	if cfg.BatchWindowSeconds < 0 || cfg.BatchWindowSeconds > 300 {
		return errInvalidFeishuBatchWindow
	}
	if completeTitle == "" {
		completeTitle = defaultFeishuCompleteTitle
	}
	if failTitle == "" {
		failTitle = defaultFeishuFailTitle
	}
	if prowlarrCompleteTitle == "" {
		prowlarrCompleteTitle = defaultFeishuProwlarrCompleteTitle
	}
	if prowlarrFailTitle == "" {
		prowlarrFailTitle = defaultFeishuProwlarrFailTitle
	}
	if prowlarrCompleteBody == "" {
		prowlarrCompleteBody = defaultFeishuProwlarrCompleteBody
	}
	if prowlarrFailBody == "" {
		prowlarrFailBody = defaultFeishuProwlarrFailBody
	}
	settings := []struct {
		key   string
		value string
	}{
		{settingFeishuNotifyType, notifyType},
		{settingFeishuBotWebhook, webhook},
		{settingFeishuReceiveOpenID, receiveOpenID},
		{settingFeishuReceiveTargets, receiveTargets},
		{settingFeishuCompleteTitle, completeTitle},
		{settingFeishuFailTitle, failTitle},
		{settingFeishuProwlarrCompleteTitle, prowlarrCompleteTitle},
		{settingFeishuProwlarrFailTitle, prowlarrFailTitle},
		{settingFeishuProwlarrCompleteBody, prowlarrCompleteBody},
		{settingFeishuProwlarrFailBody, prowlarrFailBody},
		{settingFeishuIncludeSubscription, boolSetting(cfg.IncludeSubscription)},
		{settingFeishuIncludeTitle, boolSetting(cfg.IncludeTitle)},
		{settingFeishuIncludePath, boolSetting(cfg.IncludePath)},
		{settingFeishuNotifyOnFail, boolSetting(cfg.NotifyOnFail)},
		{settingFeishuUseInteractiveCard, boolSetting(cfg.UseInteractiveCard)},
		{settingFeishuBatchWindowSeconds, strconv.Itoa(cfg.BatchWindowSeconds)},
	}
	for _, item := range settings {
		if err := s.SetSetting(ctx, item.key, item.value); err != nil {
			return err
		}
	}
	return nil
}

func parseBoolSetting(raw string, fallback bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseIntSetting(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func boolSetting(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

var errInvalidFeishuNotifyType = &feishuNotifyTypeError{}

type feishuNotifyTypeError struct{}

func (e *feishuNotifyTypeError) Error() string {
	return "feishu_notify_type 必须是 webhook 或 api"
}

var errInvalidFeishuBatchWindow = &feishuBatchWindowError{}

type feishuBatchWindowError struct{}

func (e *feishuBatchWindowError) Error() string {
	return "feishu_batch_window_seconds 必须在 0 到 300 之间"
}

// IsInvalidFeishuNotifyType 判断是否为无效通知类型错误。
func IsInvalidFeishuNotifyType(err error) bool {
	_, ok := err.(*feishuNotifyTypeError)
	return ok
}

// IsInvalidFeishuBatchWindow 判断是否为无效批量窗口错误。
func IsInvalidFeishuBatchWindow(err error) bool {
	_, ok := err.(*feishuBatchWindowError)
	return ok
}
