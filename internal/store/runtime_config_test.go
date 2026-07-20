package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/config"
)

func TestGetRuntimeServiceConfig_UsesSingleBatchQuery(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?, ?)`)).
		WithArgs("aria2_rpc_url", "aria2_rpc_secret", "feishu_app_id", "feishu_app_secret", "aria2_hook_secret").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("aria2_rpc_url", "https://aria2.test/jsonrpc").
			AddRow("aria2_rpc_secret", "rpc-secret").
			AddRow("feishu_app_id", "cli_test").
			AddRow("feishu_app_secret", "app-secret").
			AddRow("aria2_hook_secret", "hook-secret"))

	got, err := New(db).GetRuntimeServiceConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := RuntimeServiceConfig{
		Aria2RPCURL:     "https://aria2.test/jsonrpc",
		Aria2RPCSecret:  "rpc-secret",
		FeishuAppID:     "cli_test",
		FeishuAppSecret: "app-secret",
		Aria2HookSecret: "hook-secret",
	}
	if got != want {
		t.Fatalf("GetRuntimeServiceConfig() = %+v, want %+v", got, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRuntimeServiceConfig_UsesOneUpsert(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)`)).
		WithArgs(
			"aria2_rpc_url", "https://aria2.test/jsonrpc",
			"aria2_rpc_secret", "rpc-secret",
			"feishu_app_id", "cli_test",
			"feishu_app_secret", "app-secret",
			"aria2_hook_secret", "hook-secret",
		).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err = New(db).SaveRuntimeServiceConfig(context.Background(), RuntimeServiceConfig{
		Aria2RPCURL:     "https://aria2.test/jsonrpc",
		Aria2RPCSecret:  "rpc-secret",
		FeishuAppID:     "cli_test",
		FeishuAppSecret: "app-secret",
		Aria2HookSecret: "hook-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRuntimeServiceConfig_OverridesOnlySavedKeys(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?, ?)`)).
		WithArgs("aria2_rpc_url", "aria2_rpc_secret", "feishu_app_id", "feishu_app_secret", "aria2_hook_secret").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("aria2_rpc_url", "https://saved.test/jsonrpc").
			AddRow("feishu_app_secret", ""))

	base := config.Config{
		Aria2RPCURL:     "https://env.test/jsonrpc",
		Aria2RPCSecret:  "env-rpc",
		FeishuAppID:     "env-app-id",
		FeishuAppSecret: "env-app-secret",
		Aria2HookSecret: "env-hook",
	}
	got, err := New(db).ApplyRuntimeServiceConfig(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if got.Aria2RPCURL != "https://saved.test/jsonrpc" || got.FeishuAppSecret != "" {
		t.Fatalf("saved values were not applied: %+v", got)
	}
	if got.Aria2RPCSecret != "env-rpc" || got.FeishuAppID != "env-app-id" || got.Aria2HookSecret != "env-hook" {
		t.Fatalf("unsaved values should keep environment values: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
