package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const testTimeout = 15 * time.Second

// TestConnection 向 OpenAI 兼容接口发送最小 chat 请求以验证连通性。
func TestConnection(ctx context.Context, baseURL, apiKey, model string) error {
	endpoint, err := chatCompletionsURL(baseURL)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"model":      strings.TrimSpace(model),
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
	})
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setBearerAuth(req, apiKey)

	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接 API：%w", err)
	}
	defer resp.Body.Close()
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	msg := strings.TrimSpace(string(snippet))
	if msg == "" {
		return fmt.Errorf("API 返回 HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("API 返回 HTTP %d：%s", resp.StatusCode, msg)
}

func setBearerAuth(req *http.Request, apiKey string) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+key)
}

func chatCompletionsURL(baseURL string) (string, error) {
	raw := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if raw == "" {
		return "", fmt.Errorf("API 地址不能为空")
	}
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw, nil
	}
	if strings.HasSuffix(raw, "/v1") {
		return raw + "/chat/completions", nil
	}
	return raw + "/v1/chat/completions", nil
}
