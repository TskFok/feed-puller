package rss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// IsMikanTorrentURL 判断是否为蜜柑计划的 .torrent 下载链接。
func IsMikanTorrentURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Host)
	if !strings.Contains(host, "mikanani.me") && !strings.Contains(host, "mikanime.me") {
		return false
	}
	path := strings.ToLower(parsed.Path)
	return strings.HasSuffix(path, ".torrent") || strings.Contains(path, "/download/")
}

// ResolveMikanDownloadURL 将 Mikan .torrent 链接解析为 magnet；已是 magnet 或非 Mikan 种子链则原样返回。
func ResolveMikanDownloadURL(ctx context.Context, f *Fetcher, rawURL string, useProxy bool) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("下载地址不能为空")
	}
	if strings.HasPrefix(strings.ToLower(rawURL), "magnet:") {
		return rawURL, nil
	}
	if !IsMikanTorrentURL(rawURL) {
		return rawURL, nil
	}

	body, err := f.GetBytes(ctx, rawURL, useProxy, mikanDownloadHeaders(rawURL))
	if err != nil {
		return "", err
	}
	magnet, err := MagnetFromTorrent(body)
	if err != nil {
		return "", fmt.Errorf("从 torrent 生成 magnet 失败: %w", err)
	}
	return magnet, nil
}

// EnrichMikanDownloads 将条目中的 Mikan .torrent 链接替换为 magnet，便于 aria2 拉取 BT 元数据。
func EnrichMikanDownloads(ctx context.Context, f *Fetcher, items []FeedItem, useProxy bool) []FeedItem {
	if len(items) == 0 {
		return items
	}
	out := make([]FeedItem, len(items))
	copy(out, items)
	for i := range out {
		if strings.TrimSpace(out[i].DownloadURL) == "" {
			continue
		}
		resolved, err := ResolveMikanDownloadURL(ctx, f, out[i].DownloadURL, useProxy)
		if err == nil && resolved != "" {
			out[i].DownloadURL = resolved
		}
	}
	return out
}

func mikanDownloadHeaders(torrentURL string) map[string]string {
	return map[string]string{
		"Referer": "https://mikanani.me/",
		"Accept":  "application/x-bittorrent, */*",
	}
}

// GetBytes 下载资源内容（用于解析 torrent 等）；最大 4MB。
func (f *Fetcher) GetBytes(ctx context.Context, resourceURL string, useProxy bool, headers map[string]string) ([]byte, error) {
	raw := strings.TrimSpace(resourceURL)
	if raw == "" {
		return nil, fmt.Errorf("资源地址不能为空")
	}
	client := f.directClient
	if useProxy {
		if f.proxyURL == "" || f.proxyClient == nil {
			return nil, fmt.Errorf("需要代理但全局代理未配置")
		}
		client = f.proxyClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	request.Header.Set("User-Agent", "feed-puller/1.0")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("下载资源失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("下载资源失败: HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("读取资源失败: %w", err)
	}
	return body, nil
}
