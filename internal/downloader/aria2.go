package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Aria2Client struct {
	endpoint   string
	secret     string
	httpClient *http.Client
	nextID     atomic.Int64
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type jsonRPCResponse struct {
	ID     string          `json:"id"`
	Result any             `json:"result,omitempty"`
	Error  *jsonRPCError   `json:"error,omitempty"`
	Raw    json.RawMessage `json:"-"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewAria2Client(endpoint, secret string) *Aria2Client {
	return &Aria2Client{
		endpoint: strings.TrimSpace(endpoint),
		secret:   secret,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
	}
}

func (c *Aria2Client) AddURI(ctx context.Context, rawURL, dir string) (string, error) {
	if c.endpoint == "" {
		return "", fmt.Errorf("aria2 RPC 地址未配置")
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("下载地址不能为空")
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("下载目录不能为空")
	}

	params := []any{[]string{rawURL}, map[string]string{"dir": dir}}
	if c.secret != "" {
		params = append([]any{"token:" + c.secret}, params...)
	}

	var result string
	if err := c.call(ctx, "aria2.addUri", params, &result); err != nil {
		return "", err
	}
	return result, nil
}

func (c *Aria2Client) TellStatus(ctx context.Context, gid string) (map[string]any, error) {
	gid = strings.TrimSpace(gid)
	if gid == "" {
		return nil, fmt.Errorf("aria2 gid 不能为空")
	}
	params := []any{gid}
	if c.secret != "" {
		params = append([]any{"token:" + c.secret}, params...)
	}
	var result map[string]any
	if err := c.call(ctx, "aria2.tellStatus", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Aria2Client) call(ctx context.Context, method string, params []any, result any) error {
	id := fmt.Sprintf("feed-puller-%d", c.nextID.Add(1))
	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码 aria2 请求失败: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建 aria2 请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("请求 aria2 失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("请求 aria2 失败: HTTP %d", response.StatusCode)
	}

	var rpcResponse jsonRPCResponse
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&rpcResponse); err != nil {
		return fmt.Errorf("解析 aria2 响应失败: %w", err)
	}
	if rpcResponse.Error != nil {
		return fmt.Errorf("aria2 错误 %d: %s", rpcResponse.Error.Code, rpcResponse.Error.Message)
	}
	encoded, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return fmt.Errorf("处理 aria2 响应失败: %w", err)
	}
	if err := json.Unmarshal(encoded, result); err != nil {
		return fmt.Errorf("处理 aria2 响应失败: %w", err)
	}
	return nil
}
