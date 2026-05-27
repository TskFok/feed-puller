package config

import (
	"strings"
	"testing"
)

func TestEnvBool(t *testing.T) {
	t.Setenv("TEST_BOOL", "false")
	if envBool("TEST_BOOL", true) {
		t.Fatal("expected false")
	}

	t.Setenv("TEST_BOOL", "true")
	if !envBool("TEST_BOOL", false) {
		t.Fatal("expected true")
	}

	t.Setenv("TEST_BOOL", "")
	if !envBool("TEST_BOOL", true) {
		t.Fatal("expected fallback true when unset")
	}

	t.Setenv("TEST_BOOL", "invalid")
	if !envBool("TEST_BOOL", true) {
		t.Fatal("expected fallback true for invalid value")
	}
}

func TestLoadPasswordLoginEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PASSWORD_LOGIN_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.PasswordLoginEnabled {
		t.Fatal("expected password login disabled")
	}
}

func TestLoadPasswordLoginEnabledDefault(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PASSWORD_LOGIN_ENABLED", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.PasswordLoginEnabled {
		t.Fatal("expected password login enabled by default")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MYSQL_DSN", "user:pass@tcp(127.0.0.1:3306)/feed_puller?parseTime=true")
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("SESSION_SECRET", strings.Repeat("x", 32))
}
