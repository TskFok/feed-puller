package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"feed-puller/internal/config"
)

func TestHandleAuthOptions(t *testing.T) {
	server := &Server{
		cfg: config.Config{
			PasswordLoginEnabled: false,
			FeishuAppID:          "feishu-app",
			FeishuAppSecret:      "feishu-secret",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/options", nil)
	rec := httptest.NewRecorder()
	server.handleAuthOptions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["password_login_enabled"] {
		t.Fatal("expected password login disabled")
	}
	if !payload["feishu_login_enabled"] {
		t.Fatal("expected feishu login enabled")
	}
}

func TestHandleLoginDisabled(t *testing.T) {
	server := &Server{
		cfg: config.Config{PasswordLoginEnabled: false},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"a@test.dev","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["error"] != "账号密码登录已禁用" {
		t.Fatalf("unexpected error: %q", payload["error"])
	}
}
