package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const episodeExtractTimeout = 30 * time.Second

var jsonEpisodePattern = regexp.MustCompile(`"episode"\s*:\s*(\d+)`)

// ExtractEpisode 调用 OpenAI 兼容接口，从文件名与标题中识别集数。
func ExtractEpisode(ctx context.Context, baseURL, apiKey, model, filename, title string) (int, error) {
	endpoint, err := chatCompletionsURL(baseURL)
	if err != nil {
		return 0, err
	}
	filename = strings.TrimSpace(filename)
	title = strings.TrimSpace(title)
	if filename == "" && title == "" {
		return 0, fmt.Errorf("文件名与标题不能同时为空")
	}
	prompt := fmt.Sprintf(`从以下动漫资源信息中识别集数（episode number）。
只返回 JSON，格式为 {"episode": 数字}，不要输出其它内容。
若无法识别，返回 {"episode": 0}。

文件名: %s
标题: %s`, filename, title)
	body, err := json.Marshal(map[string]any{
		"model": strings.TrimSpace(model),
		"messages": []map[string]string{
			{"role": "system", "content": "你是文件名解析助手，只输出 JSON。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
		"max_tokens":  32,
	})
	if err != nil {
		return 0, fmt.Errorf("构建请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	client := &http.Client{Timeout: episodeExtractTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求 AI 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			return 0, fmt.Errorf("AI 返回 HTTP %d", resp.StatusCode)
		}
		return 0, fmt.Errorf("AI 返回 HTTP %d：%s", resp.StatusCode, msg)
	}
	content, err := parseChatCompletionContent(raw)
	if err != nil {
		return 0, err
	}
	return parseEpisodeNumber(content)
}

func parseChatCompletionContent(raw []byte) (string, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if len(payload.Choices) == 0 {
		return "", fmt.Errorf("AI 响应为空")
	}
	content := strings.TrimSpace(payload.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("AI 未返回内容")
	}
	return content, nil
}

func parseEpisodeNumber(content string) (int, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed struct {
		Episode int `json:"episode"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err == nil {
		if parsed.Episode <= 0 {
			return 0, fmt.Errorf("AI 未能识别集数")
		}
		return parsed.Episode, nil
	}
	match := jsonEpisodePattern.FindStringSubmatch(content)
	if len(match) == 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > 0 {
			return n, nil
		}
	}
	return 0, fmt.Errorf("无法解析 AI 返回的集数: %s", content)
}
