package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	feishuPassportAuthorizeURL = "https://www.feishu.cn/suite/passport/oauth/authorize"
	feishuPassportTokenURL     = "https://passport.feishu.cn/suite/passport/oauth/token"
	feishuPassportUserInfoURL  = "https://passport.feishu.cn/suite/passport/oauth/userinfo"
)

type feishuIdentity struct {
	OpenID string
	Name   string
}

func feishuRedirectURI(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/api/auth/feishu/callback"
}

func feishuPassportAuthorizeURLFor(baseURL, appID, state string) string {
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("redirect_uri", feishuRedirectURI(baseURL))
	params.Set("response_type", "code")
	params.Set("state", state)
	return feishuPassportAuthorizeURL + "?" + params.Encode()
}

func (s *Server) exchangeFeishuCode(ctx context.Context, code string) (feishuIdentity, error) {
	return fetchFeishuUserInfo(http.DefaultClient, s.cfg.FeishuAppID, s.cfg.FeishuAppSecret, feishuRedirectURI(s.cfg.BaseURL), code)
}

func fetchFeishuUserInfo(client *http.Client, appID, appSecret, redirectURI, code string) (feishuIdentity, error) {
	if strings.TrimSpace(code) == "" {
		return feishuIdentity{}, fmt.Errorf("飞书授权 code 不能为空")
	}
	if appID == "" || appSecret == "" {
		return feishuIdentity{}, fmt.Errorf("飞书应用未配置")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", appID)
	form.Set("client_secret", appSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	request, err := http.NewRequest(http.MethodPost, feishuPassportTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return feishuIdentity{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(request)
	if err != nil {
		return feishuIdentity{}, fmt.Errorf("请求飞书 token 失败: %w", err)
	}
	defer response.Body.Close()

	tokenBody, err := io.ReadAll(response.Body)
	if err != nil {
		return feishuIdentity{}, fmt.Errorf("读取飞书 token 响应失败: %w", err)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(tokenBody, &tokenData); err != nil {
		return feishuIdentity{}, fmt.Errorf("解析飞书 token 响应失败: %w", err)
	}
	if tokenData.AccessToken == "" {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.Unmarshal(tokenBody, &errResp)
		msg := errResp.ErrorDescription
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = string(tokenBody)
		}
		return feishuIdentity{}, fmt.Errorf("飞书授权失败: %s", msg)
	}

	userRequest, err := http.NewRequest(http.MethodGet, feishuPassportUserInfoURL, nil)
	if err != nil {
		return feishuIdentity{}, err
	}
	userRequest.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	userResponse, err := client.Do(userRequest)
	if err != nil {
		return feishuIdentity{}, fmt.Errorf("请求飞书 userinfo 失败: %w", err)
	}
	defer userResponse.Body.Close()

	userBody, err := io.ReadAll(userResponse.Body)
	if err != nil {
		return feishuIdentity{}, fmt.Errorf("读取飞书 userinfo 响应失败: %w", err)
	}

	var raw struct {
		OpenID string `json:"open_id"`
		Sub    string `json:"sub"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(userBody, &raw); err != nil {
		return feishuIdentity{}, fmt.Errorf("解析飞书 userinfo 失败: %w", err)
	}
	openID := raw.OpenID
	if openID == "" {
		openID = raw.Sub
	}
	if openID == "" {
		return feishuIdentity{}, fmt.Errorf("飞书响应缺少 open_id")
	}
	return feishuIdentity{OpenID: openID, Name: raw.Name}, nil
}

func feishuBindCallbackHTML(msgType, messageJSON string) string {
	const successScript = `(function(){
		var target = window.opener || window.parent;
		try { target.postMessage({type:'feishu_bind_success'}, '*'); } catch(e) {}
		if (window.opener) try { window.close(); } catch(e) {}
	})();`
	const errorScriptFmt = `(function(){
		var target = window.opener || window.parent;
		try { target.postMessage({type:'feishu_bind_error',message:%s}, '*'); } catch(e) {}
		if (window.opener) try { window.close(); } catch(e) {}
	})();`
	if msgType == "feishu_bind_success" {
		return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><p>绑定成功，窗口将自动关闭。</p><script>` + successScript + `</script></body></html>`
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><p>绑定失败</p><script>%s</script></body></html>`, fmt.Sprintf(errorScriptFmt, messageJSON))
}

func feishuLoginCallbackHTML(msgType, userJSON, messageJSON string) string {
	if msgType == "feishu_login_success" {
		return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><script>
(function(){
  var target = window.opener || window.parent;
  try { target.postMessage({type:'feishu_login_success',user:%s}, '*'); } catch(e) {}
  if (window.opener) try { window.close(); } catch(e) {}
})();
</script></body></html>`, userJSON)
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><script>
(function(){
  var target = window.opener || window.parent;
  try { target.postMessage({type:'feishu_login_error',message:%s}, '*'); } catch(e) {}
  if (window.opener) try { window.close(); } catch(e) {}
})();
</script></body></html>`, messageJSON)
}

func jsonString(value string) string {
	buf := bytes.NewBuffer(nil)
	_ = json.NewEncoder(buf).Encode(value)
	out := strings.TrimSpace(buf.String())
	if len(out) >= 2 && out[0] == '"' {
		return out
	}
	fallback, _ := json.Marshal(value)
	return string(fallback)
}
