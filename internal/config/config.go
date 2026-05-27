package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                 string
	MySQLDSN             string
	AdminEmail           string
	AdminPassword        string
	SessionSecret        string
	BaseURL              string
	Aria2RPCURL          string
	Aria2RPCSecret       string
	Aria2HookSecret      string
	FeishuAppID          string
	FeishuAppSecret      string
	PasswordLoginEnabled bool
	HTTPTimeout                  time.Duration
	StaticDir                    string
	DownloadPathHostPrefix       string
	DownloadPathContainerPrefix  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:            env("PORT", "8080"),
		MySQLDSN:        strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		AdminEmail:      strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		SessionSecret:   os.Getenv("SESSION_SECRET"),
		BaseURL:         strings.TrimRight(env("BASE_URL", "http://localhost:8080"), "/"),
		Aria2RPCURL:     strings.TrimSpace(os.Getenv("ARIA2_RPC_URL")),
		Aria2RPCSecret:  os.Getenv("ARIA2_RPC_SECRET"),
		Aria2HookSecret: strings.TrimSpace(os.Getenv("ARIA2_HOOK_SECRET")),
		FeishuAppID:          strings.TrimSpace(os.Getenv("FEISHU_APP_ID")),
		FeishuAppSecret:      os.Getenv("FEISHU_APP_SECRET"),
		PasswordLoginEnabled: envBool("PASSWORD_LOGIN_ENABLED", true),
		HTTPTimeout:          20 * time.Second,
		StaticDir:                   env("STATIC_DIR", "web/dist"),
		DownloadPathHostPrefix:      env("DOWNLOAD_PATH_HOST_PREFIX", ""),
		DownloadPathContainerPrefix: env("DOWNLOAD_PATH_CONTAINER_PREFIX", ""),
	}
	if raw := strings.TrimSpace(os.Getenv("HTTP_TIMEOUT_SECONDS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("HTTP_TIMEOUT_SECONDS 必须是正整数")
		}
		cfg.HTTPTimeout = time.Duration(value) * time.Second
	}
	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN 不能为空")
	}
	if cfg.AdminEmail == "" {
		return Config{}, fmt.Errorf("ADMIN_EMAIL 不能为空")
	}
	if cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD 不能为空")
	}
	if len(cfg.SessionSecret) < 32 {
		return Config{}, fmt.Errorf("SESSION_SECRET 至少需要 32 个字符")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
