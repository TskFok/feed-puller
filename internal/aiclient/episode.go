package aiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jsonEpisodePattern = regexp.MustCompile(`"episode"\s*:\s*(\d+)`)

// ExtractEpisode 调用 OpenAI 兼容接口，从文件名与标题中识别集数。
func ExtractEpisode(ctx context.Context, baseURL, apiKey, model, requestOptions, filename, title string) (int, error) {
	info, err := ExtractAnimeInfo(ctx, baseURL, apiKey, model, requestOptions, filename, title)
	if err != nil {
		return 0, err
	}
	return info.Episode, nil
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
