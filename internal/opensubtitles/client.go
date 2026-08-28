package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var APIBaseURL = "https://api.opensubtitles.com/api/v1"

const UserAgent = "feed-puller v1.0"

var (
	ErrLoginFailed     = errors.New("OpenSubtitles 登录失败")
	ErrInvalidFileName = errors.New("文件名无效")
	ErrInvalidFileID   = errors.New("file_id 无效")
	ErrEmptyQuery      = errors.New("搜索关键词不能为空")
)

type SubtitleFile struct {
	FileID        int64   `json:"file_id"`
	FileName      string  `json:"file_name"`
	Release       string  `json:"release"`
	Language      string  `json:"language"`
	DownloadCount int     `json:"download_count"`
	Ratings       float64 `json:"ratings"`
}

type DownloadInfo struct {
	Link     string `json:"link"`
	FileName string `json:"file_name"`
	Message  string `json:"message"`
}

type Client struct {
	username   string
	password   string
	apiKey     string
	httpClient *http.Client
	mu         sync.Mutex
	token      string
}

func NewClient(username, password, apiKey string) *Client {
	return &Client{
		username: username,
		password: password,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Search(ctx context.Context, query, language string) ([]SubtitleFile, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	endpoint, err := url.Parse(strings.TrimRight(APIBaseURL, "/") + "/subtitles")
	if err != nil {
		return nil, fmt.Errorf("搜索字幕失败: %w", err)
	}
	q := endpoint.Query()
	q.Set("query", query)
	q.Set("languages", language)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("搜索字幕失败: %w", err)
	}
	c.setJSONHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索字幕失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", apiMessage(body, "搜索字幕失败"))
	}
	return FlattenSearchData(body), nil
}

func (c *Client) RequestDownload(ctx context.Context, fileID int64) (DownloadInfo, error) {
	if fileID <= 0 {
		return DownloadInfo{}, ErrInvalidFileID
	}
	if err := c.ensureToken(ctx); err != nil {
		return DownloadInfo{}, err
	}
	info, status, err := c.postDownload(ctx, fileID)
	if err != nil {
		return DownloadInfo{}, err
	}
	if status == http.StatusUnauthorized {
		c.clearToken()
		if err := c.ensureToken(ctx); err != nil {
			return DownloadInfo{}, err
		}
		info, status, err = c.postDownload(ctx, fileID)
		if err != nil {
			return DownloadInfo{}, err
		}
	}
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(info.Message)
		if msg == "" {
			msg = "下载字幕失败"
		}
		return DownloadInfo{}, fmt.Errorf("%s", msg)
	}
	return info, nil
}

func (c *Client) FetchFile(ctx context.Context, link string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("下载字幕文件失败")
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载字幕文件失败")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("下载字幕文件失败")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("下载字幕文件失败")
	}
	return body, nil
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	token := c.token
	c.mu.Unlock()
	if token != "" {
		return nil
	}
	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) error {
	payload, err := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	if err != nil {
		return ErrLoginFailed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(APIBaseURL, "/")+"/login", bytes.NewReader(payload))
	if err != nil {
		return ErrLoginFailed
	}
	c.setJSONHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrLoginFailed
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrLoginFailed
	}
	var result struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &result)
	token := strings.TrimSpace(result.Token)
	if token == "" {
		return ErrLoginFailed
	}
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
	return nil
}

func (c *Client) postDownload(ctx context.Context, fileID int64) (DownloadInfo, int, error) {
	payload, err := json.Marshal(map[string]int64{"file_id": fileID})
	if err != nil {
		return DownloadInfo{}, 0, fmt.Errorf("下载字幕失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(APIBaseURL, "/")+"/download", bytes.NewReader(payload))
	if err != nil {
		return DownloadInfo{}, 0, fmt.Errorf("下载字幕失败: %w", err)
	}
	c.setJSONHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.currentToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DownloadInfo{}, 0, fmt.Errorf("下载字幕失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var info DownloadInfo
	_ = json.Unmarshal(body, &info)
	if strings.TrimSpace(info.Message) == "" {
		info.Message = apiMessage(body, "")
	}
	return info, resp.StatusCode, nil
}

func (c *Client) setJSONHeaders(req *http.Request) {
	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
}

func (c *Client) currentToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token
}

func (c *Client) clearToken() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

func apiMessage(body []byte, fallback string) string {
	var payload struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if msg := strings.TrimSpace(payload.Message); msg != "" {
			return msg
		}
	}
	return fallback
}
