package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const searchTimeout = 60 * time.Second

// SearchInput 表示 Prowlarr 搜索参数。
type SearchInput struct {
	Query       string
	Type        SearchType
	IndexerIDs  []int64
	Limit       int
	Offset      int
}

// Client 调用 Prowlarr REST API。
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建 Prowlarr 客户端。
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: searchTimeout,
		},
	}
}

// Search 调用 Prowlarr 搜索接口。
func (c *Client) Search(ctx context.Context, input SearchInput) ([]Release, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("Prowlarr 地址不能为空")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("Prowlarr API Key 不能为空")
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	searchType := input.Type
	if searchType == "" {
		searchType = SearchTypeMovie
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	endpoint, err := url.Parse(c.baseURL + "/api/v1/search")
	if err != nil {
		return nil, fmt.Errorf("Prowlarr 地址无效: %w", err)
	}
	q := endpoint.Query()
	q.Set("query", query)
	q.Set("type", searchType.APIType())
	q.Set("categories", searchType.Category())
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if len(input.IndexerIDs) == 0 {
		q.Set("indexerIds", "-2")
	} else {
		for _, id := range input.IndexerIDs {
			q.Add("indexerIds", strconv.FormatInt(id, 10))
		}
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return nil, fmt.Errorf("Prowlarr 返回 HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Prowlarr 返回 HTTP %d: %s", resp.StatusCode, msg)
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("解析 Prowlarr 响应失败: %w", err)
	}
	if releases == nil {
		releases = []Release{}
	}
	return releases, nil
}

// SearchMovies 兼容旧调用。
func (c *Client) SearchMovies(ctx context.Context, input SearchInput) ([]Release, error) {
	input.Type = SearchTypeMovie
	return c.Search(ctx, input)
}

// TestConnection 验证 Prowlarr 连通性。
func (c *Client) TestConnection(ctx context.Context) error {
	if c.baseURL == "" {
		return fmt.Errorf("Prowlarr 地址不能为空")
	}
	if c.apiKey == "" {
		return fmt.Errorf("Prowlarr API Key 不能为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/system/status", nil)
	if err != nil {
		return fmt.Errorf("创建 Prowlarr 请求失败: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接 Prowlarr: %w", err)
	}
	defer resp.Body.Close()
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	msg := strings.TrimSpace(string(snippet))
	if msg == "" {
		return fmt.Errorf("Prowlarr 返回 HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("Prowlarr 返回 HTTP %d: %s", resp.StatusCode, msg)
}

// FilterTorrentReleases 仅保留 Torrent 协议的结果。
func FilterTorrentReleases(releases []Release) []Release {
	out := make([]Release, 0, len(releases))
	for _, release := range releases {
		if strings.EqualFold(strings.TrimSpace(release.Protocol), "torrent") {
			out = append(out, release)
		}
	}
	return out
}
