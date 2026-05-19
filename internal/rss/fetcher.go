package rss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Fetcher struct {
	directClient *http.Client
	proxyClient  *http.Client
	proxyURL     string
}

func NewFetcher(proxyURL string) (*Fetcher, error) {
	direct := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	fetcher := &Fetcher{
		directClient: direct,
		proxyURL:     strings.TrimSpace(proxyURL),
	}
	if fetcher.proxyURL == "" {
		return fetcher, nil
	}

	parsed, err := url.Parse(fetcher.proxyURL)
	if err != nil {
		return nil, fmt.Errorf("代理地址无效: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("仅支持 HTTP/HTTPS 代理")
	}
	fetcher.proxyClient = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(parsed),
		},
	}
	return fetcher, nil
}

func (f *Fetcher) Fetch(ctx context.Context, feedURL string, useProxy bool) (Feed, error) {
	if strings.TrimSpace(feedURL) == "" {
		return Feed{}, fmt.Errorf("订阅地址不能为空")
	}

	client := f.directClient
	if useProxy {
		if f.proxyURL == "" || f.proxyClient == nil {
			return Feed{}, fmt.Errorf("订阅要求使用代理，但全局代理未配置")
		}
		client = f.proxyClient
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return Feed{}, fmt.Errorf("创建订阅请求失败: %w", err)
	}
	request.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")
	request.Header.Set("User-Agent", "feed-puller/1.0")

	response, err := client.Do(request)
	if err != nil {
		return Feed{}, fmt.Errorf("获取订阅失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Feed{}, fmt.Errorf("获取订阅失败: HTTP %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return Feed{}, fmt.Errorf("读取订阅内容失败: %w", err)
	}
	return ParseFeed(body)
}

// ProbeContentLength 通过 HEAD 探测资源的 Content-Length；不支持或失败时返回 ok=false。
func (f *Fetcher) ProbeContentLength(ctx context.Context, resourceURL string, useProxy bool) (int64, bool) {
	raw := strings.TrimSpace(resourceURL)
	if raw == "" {
		return 0, false
	}
	client := f.directClient
	if useProxy {
		if f.proxyURL == "" || f.proxyClient == nil {
			return 0, false
		}
		client = f.proxyClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return 0, false
	}
	request.Header.Set("User-Agent", "feed-puller/1.0")
	response, err := client.Do(request)
	if err != nil {
		return 0, false
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, false
	}
	if response.ContentLength >= 0 {
		return response.ContentLength, true
	}
	return 0, false
}
