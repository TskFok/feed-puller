package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Indexer 表示 Prowlarr 索引器。
type Indexer struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Enable   bool   `json:"enable"`
	Protocol string `json:"protocol"`
}

// ListIndexers 返回 Prowlarr 索引器列表。
func (c *Client) ListIndexers(ctx context.Context) ([]Indexer, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("Prowlarr 地址不能为空")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("Prowlarr API Key 不能为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/indexer", nil)
	if err != nil {
		return nil, fmt.Errorf("创建 Prowlarr 请求失败: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Prowlarr 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return nil, fmt.Errorf("Prowlarr 返回 HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Prowlarr 返回 HTTP %d: %s", resp.StatusCode, msg)
	}
	var indexers []Indexer
	if err := json.Unmarshal(body, &indexers); err != nil {
		return nil, fmt.Errorf("解析 Prowlarr 索引器响应失败: %w", err)
	}
	if indexers == nil {
		indexers = []Indexer{}
	}
	return indexers, nil
}

// FilterEnabledTorrentIndexers 仅保留已启用的 Torrent 索引器。
func FilterEnabledTorrentIndexers(indexers []Indexer) []Indexer {
	out := make([]Indexer, 0, len(indexers))
	for _, indexer := range indexers {
		if indexer.Enable && strings.EqualFold(strings.TrimSpace(indexer.Protocol), "torrent") {
			out = append(out, indexer)
		}
	}
	return out
}
