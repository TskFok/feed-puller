package aiclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildChatCompletionBody(model string, messages []map[string]string, maxTokens int, requestOptions string) ([]byte, error) {
	payload := map[string]any{
		"model":      strings.TrimSpace(model),
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	options, err := parseRequestOptions(requestOptions)
	if err != nil {
		return nil, err
	}
	for key, value := range options {
		switch key {
		case "model", "messages":
			return nil, fmt.Errorf("请求参数不能覆盖 %s", key)
		default:
			payload[key] = value
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	return body, nil
}

func parseRequestOptions(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var options map[string]any
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return nil, fmt.Errorf("请求参数 JSON 无效: %w", err)
	}
	if options == nil {
		return nil, fmt.Errorf("请求参数必须是 JSON 对象")
	}
	return options, nil
}
