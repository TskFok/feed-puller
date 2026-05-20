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

const animeExtractTimeout = 30 * time.Second

// AnimeInfo AI 识别的番剧信息（季数由订阅配置决定，不在此识别）。
type AnimeInfo struct {
	AnimeName string `json:"anime_name"`
	Episode   int    `json:"episode"`
}

var jsonAnimeNamePattern = regexp.MustCompile(`"anime_name"\s*:\s*"([^"]*)"`)

// ExtractAnimeInfo 调用 OpenAI 兼容接口，从文件名与标题中识别番剧名与集数。
func ExtractAnimeInfo(ctx context.Context, baseURL, apiKey, model, filename, title string) (*AnimeInfo, error) {
	endpoint, err := chatCompletionsURL(baseURL)
	if err != nil {
		return nil, err
	}
	filename = strings.TrimSpace(filename)
	title = strings.TrimSpace(title)
	if filename == "" && title == "" {
		return nil, fmt.Errorf("文件名与标题不能同时为空")
	}
	prompt := fmt.Sprintf(`从以下动漫资源信息中提取番剧信息，返回 JSON，仅包含 anime_name（番剧名）、episode（集数）。
只返回 JSON，不要其他内容。若无法识别集数则 episode 为 0。格式示例：{"anime_name":"鬼灭之刃","episode":1}

文件名: %s
标题: %s`, filename, title)
	body, err := json.Marshal(map[string]any{
		"model": strings.TrimSpace(model),
		"messages": []map[string]string{
			{"role": "system", "content": "你是文件名解析助手，只输出 JSON。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
		"max_tokens":  128,
	})
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	client := &http.Client{Timeout: animeExtractTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 AI 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			return nil, fmt.Errorf("AI 返回 HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("AI 返回 HTTP %d：%s", resp.StatusCode, msg)
	}
	content, err := parseChatCompletionContent(raw)
	if err != nil {
		return nil, err
	}
	return parseAnimeInfo(content)
}

func parseAnimeInfo(content string) (*AnimeInfo, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed AnimeInfo
	if err := json.Unmarshal([]byte(content), &parsed); err == nil {
		if parsed.Episode <= 0 {
			return nil, fmt.Errorf("AI 未能识别集数")
		}
		parsed.AnimeName = strings.TrimSpace(parsed.AnimeName)
		return &parsed, nil
	}

	info := &AnimeInfo{}
	if m := jsonEpisodePattern.FindStringSubmatch(content); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("AI 未能识别集数")
		}
		info.Episode = n
	} else {
		return nil, fmt.Errorf("AI 未能识别集数")
	}
	if m := jsonAnimeNamePattern.FindStringSubmatch(content); len(m) == 2 {
		info.AnimeName = strings.TrimSpace(m[1])
	}
	return info, nil
}
