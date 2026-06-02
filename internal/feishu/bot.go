package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	botTimeout          = 10 * time.Second
	botRetries            = 2
	tokenRefreshBefore    = 5 * time.Minute
)

// AppConfig 飞书开放平台应用凭证，用于服务端 API 模式。
type AppConfig struct {
	AppID     string
	AppSecret string
}

// BotClient 抽象飞书机器人 HTTP 调用，便于测试替换。
type BotClient interface {
	SendText(webhook string, title string, content string) error
	SendViaAPI(appID, appSecret, receiveIDType, receiveID, title, content string) error
	SendToUserByOpenID(openID string, title string, content string) error
	SendInteractiveWebhook(webhook string, card InteractiveCard) error
	SendInteractiveViaAPI(receiveIDType, receiveID string, card InteractiveCard) error
}

// BotService 封装发送文本消息到飞书机器人的逻辑（Webhook + 服务端 API）。
type BotService struct {
	client     *http.Client
	tokenMu    sync.RWMutex
	tokenCache map[string]*tokenEntry
	appCfg     AppConfig
}

type tokenEntry struct {
	token   string
	expires time.Time
}

// NewBotService 创建 BotService，appCfg 用于服务端 API 模式的 app_id/app_secret。
func NewBotService(appCfg AppConfig) *BotService {
	return &BotService{
		client:     &http.Client{Timeout: botTimeout},
		tokenCache: make(map[string]*tokenEntry),
		appCfg:     appCfg,
	}
}

type botRequest struct {
	MsgType string         `json:"msg_type"`
	Content botTextContent `json:"content"`
}

type botTextContent struct {
	Text string `json:"text"`
}

type botResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendText 发送文本消息到飞书机器人 Webhook。
func (s *BotService) SendText(webhook string, title string, content string) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return fmt.Errorf("飞书 Webhook 不能为空")
	}

	text := title
	if content != "" {
		text = title + "\n\n" + content
	}

	body := botRequest{
		MsgType: "text",
		Content: botTextContent{Text: text},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化飞书消息失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= botRetries; attempt++ {
		lastErr = s.doPost(webhook, payload)
		if lastErr == nil {
			return nil
		}
		if attempt < botRetries {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	return lastErr
}

func (s *BotService) doPost(webhook string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求飞书 Webhook 失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取飞书响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书 Webhook 返回非 200: status=%d, body=%s", resp.StatusCode, string(data))
	}

	var r botResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("解析飞书响应失败: %w", err)
	}
	if r.Code != 0 {
		return fmt.Errorf("飞书返回错误: code=%d, msg=%s", r.Code, r.Msg)
	}
	return nil
}

// SendViaAPI 通过飞书开放平台「发送消息」API 发送文本。
func (s *BotService) SendViaAPI(appID, appSecret, receiveIDType, receiveID, title, content string) error {
	appID = strings.TrimSpace(appID)
	appSecret = strings.TrimSpace(appSecret)
	receiveIDType = strings.TrimSpace(receiveIDType)
	receiveID = strings.TrimSpace(receiveID)
	if appID == "" || appSecret == "" || receiveID == "" {
		return fmt.Errorf("飞书 API 配置不完整：app_id、app_secret、receive_id 不能为空")
	}
	validTypes := map[string]bool{"chat_id": true, "user_id": true, "open_id": true}
	if !validTypes[receiveIDType] {
		receiveIDType = "open_id"
	}

	text := title
	if content != "" {
		text = title + "\n\n" + content
	}

	token, err := s.getTenantAccessToken(appID, appSecret)
	if err != nil {
		return err
	}

	contentJSON, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("序列化消息内容失败: %w", err)
	}

	body := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveIDType
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	var lastErr error
	for attempt := 0; attempt <= botRetries; attempt++ {
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求飞书 API 失败: %w", err)
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("飞书 API 返回非 200: status=%d, body=%s", resp.StatusCode, string(data))
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		var apiResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.Unmarshal(data, &apiResp); err != nil {
			lastErr = fmt.Errorf("解析飞书 API 响应失败: %w", err)
			break
		}
		if apiResp.Code != 0 {
			lastErr = fmt.Errorf("飞书 API 错误: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
			if apiResp.Code == 99991663 || apiResp.Code == 99991664 {
				s.invalidateToken(appID)
			}
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		return nil
	}
	return lastErr
}

// SendToUserByOpenID 使用配置的 app_id/app_secret 向指定 open_id 用户发送消息。
func (s *BotService) SendToUserByOpenID(openID string, title string, content string) error {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return fmt.Errorf("飞书 open_id 不能为空")
	}
	if strings.TrimSpace(s.appCfg.AppID) == "" || strings.TrimSpace(s.appCfg.AppSecret) == "" {
		return fmt.Errorf("飞书应用配置未设置（app_id、app_secret）")
	}
	return s.SendViaAPI(s.appCfg.AppID, s.appCfg.AppSecret, "open_id", openID, title, content)
}

// SendInteractiveWebhook 通过 Webhook 发送 interactive 卡片。
func (s *BotService) SendInteractiveWebhook(webhook string, card InteractiveCard) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return fmt.Errorf("飞书 Webhook 不能为空")
	}
	payload, err := json.Marshal(map[string]any{
		"msg_type": "interactive",
		"card":     cardPayload(card),
	})
	if err != nil {
		return fmt.Errorf("序列化飞书卡片失败: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt <= botRetries; attempt++ {
		lastErr = s.doPost(webhook, payload)
		if lastErr == nil {
			return nil
		}
		if attempt < botRetries {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	return lastErr
}

// SendInteractiveViaAPI 通过飞书开放平台 API 发送 interactive 卡片。
func (s *BotService) SendInteractiveViaAPI(receiveIDType, receiveID string, card InteractiveCard) error {
	receiveIDType = strings.TrimSpace(receiveIDType)
	receiveID = strings.TrimSpace(receiveID)
	if strings.TrimSpace(s.appCfg.AppID) == "" || strings.TrimSpace(s.appCfg.AppSecret) == "" {
		return fmt.Errorf("飞书应用配置未设置（app_id、app_secret）")
	}
	if receiveID == "" {
		return fmt.Errorf("飞书 receive_id 不能为空")
	}
	validTypes := map[string]bool{"chat_id": true, "user_id": true, "open_id": true}
	if !validTypes[receiveIDType] {
		receiveIDType = "open_id"
	}
	token, err := s.getTenantAccessToken(s.appCfg.AppID, s.appCfg.AppSecret)
	if err != nil {
		return err
	}
	contentJSON, err := json.Marshal(cardPayload(card))
	if err != nil {
		return fmt.Errorf("序列化卡片内容失败: %w", err)
	}
	body := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "interactive",
		"content":    string(contentJSON),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}
	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveIDType
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	var lastErr error
	for attempt := 0; attempt <= botRetries; attempt++ {
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求飞书 API 失败: %w", err)
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("飞书 API 返回非 200: status=%d, body=%s", resp.StatusCode, string(data))
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		var apiResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.Unmarshal(data, &apiResp); err != nil {
			lastErr = fmt.Errorf("解析飞书 API 响应失败: %w", err)
			break
		}
		if apiResp.Code != 0 {
			lastErr = fmt.Errorf("飞书 API 错误: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
			if apiResp.Code == 99991663 || apiResp.Code == 99991664 {
				s.invalidateToken(s.appCfg.AppID)
			}
			if attempt < botRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
			continue
		}
		return nil
	}
	return lastErr
}

// SendToRecipients 向多个接收者发送通知（API 模式）。
func (s *BotService) SendToRecipients(recipients []Recipient, card InteractiveCard, title, content string, useCard bool) error {
	if len(recipients) == 0 {
		return fmt.Errorf("飞书接收者不能为空")
	}
	var errs []string
	for _, rcpt := range recipients {
		var err error
		if useCard {
			err = s.SendInteractiveViaAPI(rcpt.IDType, rcpt.ID, card)
		} else {
			err = s.SendViaAPI(s.appCfg.AppID, s.appCfg.AppSecret, rcpt.IDType, rcpt.ID, title, content)
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s:%s: %v", rcpt.IDType, rcpt.ID, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分接收者发送失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *BotService) getTenantAccessToken(appID, appSecret string) (string, error) {
	s.tokenMu.RLock()
	e, ok := s.tokenCache[appID]
	if ok && e.expires.After(time.Now().Add(tokenRefreshBefore)) {
		token := e.token
		s.tokenMu.RUnlock()
		return token, nil
	}
	s.tokenMu.RUnlock()

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	e, ok = s.tokenCache[appID]
	if ok && e.expires.After(time.Now().Add(tokenRefreshBefore)) {
		return e.token, nil
	}

	reqBody := map[string]string{"app_id": appID, "app_secret": appSecret}
	payload, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("创建 token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 tenant_access_token 失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 token 响应失败: %w", err)
	}
	var r struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("解析 token 响应失败: %w", err)
	}
	if r.Code != 0 {
		return "", fmt.Errorf("飞书 token 接口错误: code=%d, msg=%s", r.Code, r.Msg)
	}
	expireSec := r.Expire
	if expireSec <= 0 {
		expireSec = 7200
	}
	s.tokenCache[appID] = &tokenEntry{token: r.TenantAccessToken, expires: time.Now().Add(time.Duration(expireSec) * time.Second)}
	return r.TenantAccessToken, nil
}

func (s *BotService) invalidateToken(appID string) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	delete(s.tokenCache, appID)
}
