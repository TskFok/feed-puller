package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"feed-puller/internal/feishu"
	"feed-puller/internal/store"
)

type feishuNotifyKind string

const (
	feishuNotifyComplete feishuNotifyKind = "complete"
	feishuNotifyFail     feishuNotifyKind = "fail"
)

type feishuNotifyPayload struct {
	source    string
	mediaType string
	subName   string
	itemTitle string
	path      string
	errMsg    string
}

type feishuNotifySender interface {
	SendText(webhook string, title string, content string) error
	SendInteractiveWebhook(webhook string, card feishu.InteractiveCard) error
	SendToUserByOpenID(openID string, title string, content string) error
	SendInteractiveViaAPI(receiveIDType, receiveID string, card feishu.InteractiveCard) error
	SendToRecipients(recipients []feishu.Recipient, card feishu.InteractiveCard, title, content string, useCard bool) error
}

func feishuPayloadFromSubscription(sub store.Subscription, item store.Item, finalPath, errMsg string) feishuNotifyPayload {
	payload := feishuNotifyPayload{
		subName:   strings.TrimSpace(sub.Name),
		itemTitle: strings.TrimSpace(item.Title),
		path:      strings.TrimSpace(finalPath),
		errMsg:    strings.TrimSpace(errMsg),
		source:    "rss",
	}
	if store.IsProwlarrInternalSubscription(sub) {
		payload.source = "prowlarr"
		payload.mediaType = prowlarrMediaType(sub)
	}
	return payload
}

func prowlarrMediaType(sub store.Subscription) string {
	switch {
	case store.IsProwlarrMovieSubscription(sub):
		return "电影"
	case store.IsProwlarrTVSubscription(sub):
		return "剧集"
	case store.IsProwlarrInternalSubscription(sub):
		return "Prowlarr"
	default:
		return ""
	}
}

func (s *Service) queueFeishuNotifyComplete(ctx context.Context, sub store.Subscription, item store.Item, finalPath string) {
	s.queueFeishuNotify(ctx, feishuNotifyComplete, feishuPayloadFromSubscription(sub, item, finalPath, ""))
}

func (s *Service) queueFeishuNotifyFail(ctx context.Context, sub store.Subscription, item store.Item, errMsg string) {
	s.queueFeishuNotify(ctx, feishuNotifyFail, feishuPayloadFromSubscription(sub, item, "", errMsg))
}

func (s *Service) queueFeishuNotify(ctx context.Context, kind feishuNotifyKind, payload feishuNotifyPayload) {
	if s.feishuBot == nil {
		return
	}
	cfg, err := s.store.GetFeishuNotifyConfig(ctx)
	if err != nil {
		s.log.Warn("读取飞书通知配置失败", "error", err)
		return
	}
	if !feishuNotifyEnabled(cfg) {
		return
	}
	if kind == feishuNotifyFail && !cfg.NotifyOnFail {
		return
	}
	if cfg.BatchWindowSeconds <= 0 {
		if err := s.deliverFeishuNotify(ctx, cfg, kind, []feishuNotifyPayload{payload}); err != nil {
			s.log.Warn("飞书通知发送失败", "kind", kind, "error", err)
		}
		return
	}
	s.feishuBatchMu.Lock()
	defer s.feishuBatchMu.Unlock()
	switch kind {
	case feishuNotifyComplete:
		s.feishuBatchComplete = append(s.feishuBatchComplete, payload)
	case feishuNotifyFail:
		s.feishuBatchFail = append(s.feishuBatchFail, payload)
	}
	if s.feishuBatchTimer != nil {
		s.feishuBatchTimer.Stop()
	}
	window := time.Duration(cfg.BatchWindowSeconds) * time.Second
	s.feishuBatchTimer = time.AfterFunc(window, s.flushFeishuBatch)
}

func (s *Service) flushFeishuBatch() {
	s.feishuBatchMu.Lock()
	complete := s.feishuBatchComplete
	fail := s.feishuBatchFail
	s.feishuBatchComplete = nil
	s.feishuBatchFail = nil
	s.feishuBatchTimer = nil
	s.feishuBatchMu.Unlock()

	ctx := context.Background()
	cfg, err := s.store.GetFeishuNotifyConfig(ctx)
	if err != nil {
		s.log.Warn("飞书批量通知: 读取配置失败", "error", err)
		return
	}
	if !feishuNotifyEnabled(cfg) {
		return
	}
	if len(complete) > 0 {
		if err := s.deliverFeishuNotify(ctx, cfg, feishuNotifyComplete, complete); err != nil {
			s.log.Warn("飞书批量通知发送失败", "kind", feishuNotifyComplete, "error", err)
		}
	}
	if len(fail) > 0 && cfg.NotifyOnFail {
		if err := s.deliverFeishuNotify(ctx, cfg, feishuNotifyFail, fail); err != nil {
			s.log.Warn("飞书批量通知发送失败", "kind", feishuNotifyFail, "error", err)
		}
	}
}

func feishuNotifyEnabled(cfg store.FeishuNotifyConfig) bool {
	notifyType := cfg.NotifyType
	if notifyType == "" && cfg.Webhook != "" {
		notifyType = "webhook"
	}
	return notifyType != ""
}

func (s *Service) deliverFeishuNotify(ctx context.Context, cfg store.FeishuNotifyConfig, kind feishuNotifyKind, payloads []feishuNotifyPayload) error {
	if len(payloads) == 0 {
		return nil
	}
	title, lines, cardTemplate := buildFeishuNotifyMessage(cfg, kind, payloads)
	content := feishu.BuildTextBody(lines)
	card := feishu.InteractiveCard{Title: title, Template: cardTemplate, Lines: lines}

	notifyType := cfg.NotifyType
	if notifyType == "" && cfg.Webhook != "" {
		notifyType = "webhook"
	}
	var sendErr error
	switch notifyType {
	case "webhook":
		if strings.TrimSpace(cfg.Webhook) == "" {
			return nil
		}
		if cfg.UseInteractiveCard {
			sendErr = s.feishuBot.SendInteractiveWebhook(cfg.Webhook, card)
		} else {
			sendErr = s.feishuBot.SendText(cfg.Webhook, title, content)
		}
	case "api":
		recipients := feishu.ParseRecipients(cfg.ReceiveOpenID, cfg.ReceiveTargets)
		if len(recipients) == 0 {
			err := fmt.Errorf("未配置飞书接收者")
			s.recordFeishuNotifyHistory(ctx, kind, payloads, notifyType, title, content, "failed", err.Error())
			return err
		}
		sendErr = s.feishuBot.SendToRecipients(recipients, card, title, content, cfg.UseInteractiveCard)
	default:
		return nil
	}
	status := "sent"
	errText := ""
	if sendErr != nil {
		status = "failed"
		errText = sendErr.Error()
	} else {
		s.log.Info("飞书通知已发送", "kind", kind, "count", len(payloads), "notify_type", notifyType, "card", cfg.UseInteractiveCard)
	}
	s.recordFeishuNotifyHistory(ctx, kind, payloads, notifyType, title, content, status, errText)
	return sendErr
}

func (s *Service) recordFeishuNotifyHistory(ctx context.Context, kind feishuNotifyKind, payloads []feishuNotifyPayload, notifyType, title, content, status, errText string) {
	if s.store == nil {
		return
	}
	source := feishuHistorySource(payloads)
	eventType := string(kind)
	if eventType == "" {
		eventType = "complete"
	}
	if err := s.store.CreateFeishuNotifyHistory(ctx, store.FeishuNotifyHistory{
		EventType:  eventType,
		Source:     source,
		NotifyType: notifyType,
		Title:      title,
		Content:    content,
		ItemCount:  len(payloads),
		Status:     status,
		Error:      errText,
	}); err != nil {
		s.log.Warn("写入飞书通知历史失败", "error", err)
	}
}

func feishuHistorySource(payloads []feishuNotifyPayload) string {
	if len(payloads) == 0 {
		return "rss"
	}
	source := payloads[0].source
	if source == "" {
		source = "rss"
	}
	for _, p := range payloads[1:] {
		if p.source != source {
			return "rss"
		}
	}
	return source
}

func payloadsUseProwlarrTemplate(payloads []feishuNotifyPayload) bool {
	if len(payloads) == 0 {
		return false
	}
	for _, p := range payloads {
		if p.source != "prowlarr" {
			return false
		}
	}
	return true
}

func titleTemplates(cfg store.FeishuNotifyConfig, kind feishuNotifyKind, useProwlarr bool) (completeTitle, failTitle string) {
	if useProwlarr {
		return cfg.ProwlarrCompleteTitleTemplate, cfg.ProwlarrFailTitleTemplate
	}
	return cfg.CompleteTitleTemplate, cfg.FailTitleTemplate
}

func buildFeishuNotifyMessage(cfg store.FeishuNotifyConfig, kind feishuNotifyKind, payloads []feishuNotifyPayload) (title string, lines []string, cardTemplate string) {
	useProwlarr := payloadsUseProwlarrTemplate(payloads)
	completeTitle, failTitle := titleTemplates(cfg, kind, useProwlarr)
	count := len(payloads)
	if count == 1 {
		p := payloads[0]
		vars := feishu.TemplateVars{
			Subscription: p.subName,
			Title:        p.itemTitle,
			Path:         p.path,
			Error:        p.errMsg,
			Count:        "1",
			MediaType:    p.mediaType,
		}
		switch kind {
		case feishuNotifyFail:
			title = feishu.RenderTemplate(failTitle, vars)
			cardTemplate = "red"
		default:
			title = feishu.RenderTemplate(completeTitle, vars)
			cardTemplate = "green"
		}
		lines = buildFeishuItemLines(cfg, p, kind)
		return title, lines, cardTemplate
	}

	countText := strconv.Itoa(count)
	switch kind {
	case feishuNotifyFail:
		title = feishu.RenderTemplate(failTitle, feishu.TemplateVars{Count: countText})
		if !strings.Contains(title, countText) {
			title = fmt.Sprintf("%s (%s 项)", title, countText)
		}
		cardTemplate = "red"
	default:
		title = feishu.RenderTemplate(completeTitle, feishu.TemplateVars{Count: countText})
		if !strings.Contains(title, countText) {
			title = fmt.Sprintf("%s (%s 项)", title, countText)
		}
		cardTemplate = "green"
	}
	lines = make([]string, 0, count+1)
	lines = append(lines, fmt.Sprintf("共 **%s** 项", countText))
	for i, p := range payloads {
		itemLines := buildFeishuItemLines(cfg, p, kind)
		if len(itemLines) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, strings.Join(itemLines, " · ")))
	}
	return title, lines, cardTemplate
}

func buildFeishuItemLines(cfg store.FeishuNotifyConfig, p feishuNotifyPayload, kind feishuNotifyKind) []string {
	if p.source == "prowlarr" {
		if lines := renderProwlarrBodyTemplate(cfg, p, kind); len(lines) > 0 {
			return lines
		}
	}
	var lines []string
	if p.mediaType != "" && p.source != "prowlarr" {
		if cfg.UseInteractiveCard {
			lines = append(lines, "**类型**: "+p.mediaType)
		} else {
			lines = append(lines, "类型："+p.mediaType)
		}
	}
	if cfg.IncludeSubscription && p.subName != "" {
		if cfg.UseInteractiveCard {
			lines = append(lines, "**订阅**: "+p.subName)
		} else {
			lines = append(lines, "订阅："+p.subName)
		}
	}
	if cfg.IncludeTitle && p.itemTitle != "" {
		if cfg.UseInteractiveCard {
			lines = append(lines, "**标题**: "+p.itemTitle)
		} else {
			lines = append(lines, "标题："+p.itemTitle)
		}
	}
	if cfg.IncludePath && p.path != "" {
		if cfg.UseInteractiveCard {
			lines = append(lines, "**路径**: "+p.path)
		} else {
			lines = append(lines, "路径："+p.path)
		}
	}
	if kind == feishuNotifyFail && p.errMsg != "" {
		if cfg.UseInteractiveCard {
			lines = append(lines, "**错误**: "+p.errMsg)
		} else {
			lines = append(lines, "错误："+p.errMsg)
		}
	}
	return lines
}

func renderProwlarrBodyTemplate(cfg store.FeishuNotifyConfig, p feishuNotifyPayload, kind feishuNotifyKind) []string {
	tmpl := cfg.ProwlarrCompleteBodyTemplate
	if kind == feishuNotifyFail {
		tmpl = cfg.ProwlarrFailBodyTemplate
	}
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return nil
	}
	vars := feishu.TemplateVars{
		Subscription: p.subName,
		Title:        p.itemTitle,
		Path:         p.path,
		Error:        p.errMsg,
		Count:        "1",
		MediaType:    p.mediaType,
	}
	rendered := feishu.RenderTemplate(tmpl, vars)
	parts := strings.Split(rendered, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			lines = append(lines, part)
		}
	}
	return lines
}

func (s *Service) notifyDownloadComplete(ctx context.Context, sub store.Subscription, item store.Item, finalPath string) {
	s.queueFeishuNotifyComplete(ctx, sub, item, finalPath)
}

func (s *Service) notifyDownloadFail(ctx context.Context, sub store.Subscription, item store.Item, errMsg string) {
	s.queueFeishuNotifyFail(ctx, sub, item, errMsg)
}

func (s *Service) notifyDownloadCompleteAfterSync(ctx context.Context, task store.DownloadTask, finalPath string) {
	sub, subErr := s.store.GetSubscription(ctx, task.SubscriptionID)
	if subErr != nil {
		s.log.Warn("飞书通知: 读取订阅失败", "subscription_id", task.SubscriptionID, "error", subErr)
		return
	}
	item := store.Item{Title: ""}
	if fetched, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
		item = fetched
	}
	s.notifyDownloadComplete(ctx, sub, item, finalPath)
}

func (s *Service) notifyDownloadFailAfterSync(ctx context.Context, task store.DownloadTask, errMsg string) {
	sub, subErr := s.store.GetSubscription(ctx, task.SubscriptionID)
	if subErr != nil {
		s.log.Warn("飞书通知: 读取订阅失败", "subscription_id", task.SubscriptionID, "error", subErr)
		return
	}
	item := store.Item{Title: ""}
	if fetched, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
		item = fetched
	}
	s.notifyDownloadFail(ctx, sub, item, errMsg)
}

// SendFeishuTestNotification 发送测试消息，供设置页验证配置。
func (s *Service) SendFeishuTestNotification(ctx context.Context, cfg store.FeishuNotifyConfig) error {
	if s.feishuBot == nil {
		return fmt.Errorf("飞书机器人服务不可用")
	}
	if !feishuNotifyEnabled(cfg) {
		return fmt.Errorf("请先配置飞书通知方式（Webhook 或服务端 API）")
	}
	testCfg := cfg
	testCfg.CompleteTitleTemplate = "Feed Puller 测试消息"
	testPayload := feishuNotifyPayload{
		source:    "test",
		subName:   "测试订阅",
		itemTitle: "测试条目",
		path:      "/data/example.mp4",
	}
	title, lines, cardTemplate := buildFeishuNotifyMessage(testCfg, feishuNotifyComplete, []feishuNotifyPayload{testPayload})
	content := feishu.BuildTextBody(lines)
	card := feishu.InteractiveCard{Title: title, Template: cardTemplate, Lines: lines}

	notifyType := cfg.NotifyType
	if notifyType == "" && cfg.Webhook != "" {
		notifyType = "webhook"
	}
	var sendErr error
	switch notifyType {
	case "webhook":
		if strings.TrimSpace(cfg.Webhook) == "" {
			return fmt.Errorf("请先填写 Webhook URL")
		}
		if cfg.UseInteractiveCard {
			sendErr = s.feishuBot.SendInteractiveWebhook(cfg.Webhook, card)
		} else {
			sendErr = s.feishuBot.SendText(cfg.Webhook, title, content)
		}
	case "api":
		recipients := feishu.ParseRecipients(cfg.ReceiveOpenID, cfg.ReceiveTargets)
		if len(recipients) == 0 {
			return fmt.Errorf("服务端 API 模式需配置接收者 open_id，或先绑定飞书账号")
		}
		sendErr = s.feishuBot.SendToRecipients(recipients, card, title, content, cfg.UseInteractiveCard)
	default:
		return fmt.Errorf("不支持的通知方式: %q", notifyType)
	}
	status := "sent"
	errText := ""
	if sendErr != nil {
		status = "failed"
		errText = sendErr.Error()
	}
	_ = s.store.CreateFeishuNotifyHistory(ctx, store.FeishuNotifyHistory{
		EventType:  "test",
		Source:     "test",
		NotifyType: notifyType,
		Title:      title,
		Content:    content,
		ItemCount:  1,
		Status:     status,
		Error:      errText,
	})
	return sendErr
}

// SetFeishuBot 注入飞书机器人客户端（测试或运行时配置）。
func (s *Service) SetFeishuBot(bot feishuNotifySender) {
	s.feishuBot = bot
}

// FlushFeishuBatchForTest 立即刷新批量通知队列（仅测试使用）。
func (s *Service) FlushFeishuBatchForTest() {
	s.feishuBatchMu.Lock()
	if s.feishuBatchTimer != nil {
		s.feishuBatchTimer.Stop()
		s.feishuBatchTimer = nil
	}
	s.feishuBatchMu.Unlock()
	s.flushFeishuBatch()
}

var _ feishuNotifySender = (*feishu.BotService)(nil)
