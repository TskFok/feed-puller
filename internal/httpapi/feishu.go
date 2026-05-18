package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type feishuIdentity struct {
	OpenID string
	Name   string
}

func (s *Server) exchangeFeishuCode(ctx context.Context, code string) (feishuIdentity, error) {
	if strings.TrimSpace(code) == "" {
		return feishuIdentity{}, fmt.Errorf("飞书授权 code 不能为空")
	}
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     s.cfg.FeishuAppID,
		"client_secret": s.cfg.FeishuAppSecret,
		"code":          code,
		"redirect_uri":  s.cfg.BaseURL + "/api/auth/feishu/callback",
	}
	body, _ := json.Marshal(payload)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/authen/v2/oauth/token", bytes.NewReader(body))
	if err != nil {
		return feishuIdentity{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return feishuIdentity{}, fmt.Errorf("请求飞书失败: %w", err)
	}
	defer response.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OpenID string `json:"open_id"`
			Name   string `json:"name"`
			EnName string `json:"en_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return feishuIdentity{}, fmt.Errorf("解析飞书响应失败: %w", err)
	}
	if result.Code != 0 {
		return feishuIdentity{}, fmt.Errorf("飞书授权失败: %s", result.Msg)
	}
	name := result.Data.Name
	if name == "" {
		name = result.Data.EnName
	}
	if result.Data.OpenID == "" {
		return feishuIdentity{}, fmt.Errorf("飞书响应缺少 open_id")
	}
	return feishuIdentity{OpenID: result.Data.OpenID, Name: name}, nil
}
