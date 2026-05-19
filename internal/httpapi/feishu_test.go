package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"feed-puller/internal/config"
)

func TestFeishuPassportAuthorizeURLFor(t *testing.T) {
	url := feishuPassportAuthorizeURLFor("http://localhost:8080", "cli_test", "login")
	if !strings.Contains(url, "client_id=cli_test") {
		t.Fatalf("expected client_id in url, got %q", url)
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Fatalf("expected redirect_uri in url, got %q", url)
	}
	if !strings.Contains(url, "state=login") {
		t.Fatalf("expected state=login in url, got %q", url)
	}
	if !strings.HasPrefix(url, feishuPassportAuthorizeURL) {
		t.Fatalf("expected passport authorize prefix, got %q", url)
	}
}

func TestFeishuLoginCallbackHTML(t *testing.T) {
	html := feishuLoginCallbackHTML("feishu_login_success", `{"id":1,"email":"a@test.dev"}`, "")
	if !strings.Contains(html, "feishu_login_success") {
		t.Fatalf("expected success postMessage type")
	}
	if !strings.Contains(html, `"email":"a@test.dev"`) {
		t.Fatalf("expected user payload in html")
	}

	errHTML := feishuLoginCallbackHTML("feishu_login_error", "", jsonString("未绑定"))
	if !strings.Contains(errHTML, "feishu_login_error") {
		t.Fatalf("expected error postMessage type")
	}
	if !strings.Contains(errHTML, "未绑定") {
		t.Fatalf("expected error message in html")
	}
}

func TestFeishuBindCallbackHTML(t *testing.T) {
	html := feishuBindCallbackHTML("feishu_bind_success", "")
	if !strings.Contains(html, "feishu_bind_success") {
		t.Fatalf("expected bind success postMessage type")
	}

	errHTML := feishuBindCallbackHTML("feishu_bind_error", jsonString("绑定失败"))
	if !strings.Contains(errHTML, "feishu_bind_error") {
		t.Fatalf("expected bind error postMessage type")
	}
}

func TestFetchFeishuUserInfo(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/suite/passport/oauth/token" {
			t.Fatalf("unexpected token path: %s", r.URL.Path)
		}
		_ = r.ParseForm()
		if r.Form.Get("code") != "good-code" {
			t.Fatalf("unexpected code: %q", r.Form.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token-abc"}`))
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/suite/passport/oauth/userinfo" {
			t.Fatalf("unexpected userinfo path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-abc" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"open_id":"ou_test","name":"Alice"}`))
	}))
	defer userServer.Close()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.String(), "/suite/passport/oauth/token"):
				req.URL.Scheme = "http"
				req.URL.Host = strings.TrimPrefix(tokenServer.URL, "http://")
			case strings.HasSuffix(req.URL.String(), "/suite/passport/oauth/userinfo"):
				req.URL.Scheme = "http"
				req.URL.Host = strings.TrimPrefix(userServer.URL, "http://")
			}
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	identity, err := fetchFeishuUserInfo(client, "app-id", "app-secret", "http://localhost:8080/api/auth/feishu/callback", "good-code")
	if err != nil {
		t.Fatalf("fetchFeishuUserInfo failed: %v", err)
	}
	if identity.OpenID != "ou_test" || identity.Name != "Alice" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestHandleFeishuLoginURL(t *testing.T) {
	server := &Server{
		cfg: config.Config{
			BaseURL:         "http://localhost:8080",
			FeishuAppID:     "feishu-app",
			FeishuAppSecret: "feishu-secret",
			SessionSecret:   strings.Repeat("x", 32),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/feishu/login-url", nil)
	rec := httptest.NewRecorder()
	server.handleFeishuLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["url"] != "/api/auth/feishu/login?state=login" {
		t.Fatalf("unexpected url: %q", payload["url"])
	}
	if !strings.Contains(payload["goto"], "client_id=feishu-app") {
		t.Fatalf("unexpected goto: %q", payload["goto"])
	}
}
