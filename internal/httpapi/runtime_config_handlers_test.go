package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func newRuntimeConfigServer(t *testing.T, cfg config.Config) (*Server, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	service := app.NewService(repo, downloader.NewAria2Client(cfg.Aria2RPCURL, cfg.Aria2RPCSecret), slog.Default())
	return New(cfg, repo, service, slog.Default()), mock, func() { _ = db.Close() }
}

func TestRuntimeServiceConfigRequiresAuth(t *testing.T) {
	srv, _, cleanup := newRuntimeConfigServer(t, config.Config{})
	defer cleanup()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/settings/runtime-config", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRuntimeServiceConfigGetReturnsCurrentValues(t *testing.T) {
	cfg := config.Config{
		Aria2RPCURL:     "https://aria2.test/jsonrpc",
		Aria2RPCSecret:  "rpc-secret",
		FeishuAppID:     "cli_test",
		FeishuAppSecret: "app-secret",
		Aria2HookSecret: "hook-secret",
	}
	srv, _, cleanup := newRuntimeConfigServer(t, cfg)
	defer cleanup()

	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/settings/runtime-config", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusOK)
	}
	var got store.RuntimeServiceConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Aria2RPCURL != cfg.Aria2RPCURL || got.Aria2HookSecret != cfg.Aria2HookSecret || got.FeishuAppSecret != cfg.FeishuAppSecret {
		t.Fatalf("response = %+v", got)
	}
}

func TestRuntimeServiceConfigPutRejectsInvalidAria2URL(t *testing.T) {
	srv, mock, cleanup := newRuntimeConfigServer(t, config.Config{})
	defer cleanup()

	body := strings.NewReader(`{"aria2_rpc_url":"ftp://aria2.test","aria2_rpc_secret":"","feishu_app_id":"","feishu_app_secret":"","aria2_hook_secret":""}`)
	req := authRequest(httptest.NewRequest(http.MethodPut, "/api/settings/runtime-config", body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeServiceConfigPutPersistsAndHotSwapsDependencies(t *testing.T) {
	srv, mock, cleanup := newRuntimeConfigServer(t, config.Config{Aria2HookSecret: "old-hook"})
	defer cleanup()
	oldAria2 := srv.service.Aria2Client()
	oldFeishu := srv.service.FeishuBot()
	input := store.RuntimeServiceConfig{
		Aria2RPCURL:     " https://aria2.test/jsonrpc ",
		Aria2RPCSecret:  " rpc-secret ",
		FeishuAppID:     " cli_test ",
		FeishuAppSecret: " app-secret ",
		Aria2HookSecret: " hook-secret ",
	}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)`)).
		WithArgs(
			"aria2_rpc_url", "https://aria2.test/jsonrpc",
			"aria2_rpc_secret", "rpc-secret",
			"feishu_app_id", "cli_test",
			"feishu_app_secret", "app-secret",
			"aria2_hook_secret", "hook-secret",
		).
		WillReturnResult(sqlmock.NewResult(0, 5))
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	req := authRequest(httptest.NewRequest(http.MethodPut, "/api/settings/runtime-config", bytes.NewReader(body)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got store.RuntimeServiceConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Aria2RPCURL != "https://aria2.test/jsonrpc" || got.FeishuAppID != "cli_test" || got.Aria2HookSecret != "hook-secret" {
		t.Fatalf("response = %+v", got)
	}
	if srv.service.Aria2Client() == oldAria2 || srv.service.FeishuBot() == oldFeishu {
		t.Fatal("service dependencies were not replaced")
	}
	hookRequest := httptest.NewRequest(http.MethodPost, "/api/downloads/aria2-hook", nil)
	hookRequest.Header.Set("X-Hook-Secret", "hook-secret")
	if !srv.checkAria2HookSecret(hookRequest) {
		t.Fatal("new hook secret was not applied")
	}
	options := httptest.NewRecorder()
	srv.handleAuthOptions(options, httptest.NewRequest(http.MethodGet, "/api/auth/options", nil))
	var auth map[string]bool
	if err := json.Unmarshal(options.Body.Bytes(), &auth); err != nil {
		t.Fatal(err)
	}
	if !auth["feishu_login_enabled"] {
		t.Fatal("new feishu credentials were not applied")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
