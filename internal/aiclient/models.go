package aiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// ListModels 从 OpenAI 兼容接口 GET /v1/models 拉取模型 ID 列表。
func ListModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	endpoint, err := modelsURL(baseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	setBearerAuth(req, apiKey)

	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接 API：%w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			return nil, fmt.Errorf("API 返回 HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("API 返回 HTTP %d：%s", resp.StatusCode, msg)
	}
	return parseModelsPayload(raw), nil
}

func modelsURL(baseURL string) (string, error) {
	raw := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if raw == "" {
		return "", fmt.Errorf("API 地址不能为空")
	}
	raw = strings.TrimSuffix(raw, "/chat/completions")
	if strings.HasSuffix(raw, "/models") {
		return raw, nil
	}
	if strings.HasSuffix(raw, "/v1") {
		return raw + "/models", nil
	}
	return raw + "/v1/models", nil
}

func parseModelsPayload(raw []byte) []string {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(payload.Data))
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		models = append(models, id)
	}
	sort.Strings(models)
	return models
}
