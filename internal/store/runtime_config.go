package store

import (
	"context"
	"fmt"

	"feed-puller/internal/config"
)

const (
	settingAria2RPCURL     = "aria2_rpc_url"
	settingAria2RPCSecret  = "aria2_rpc_secret"
	settingFeishuAppID     = "feishu_app_id"
	settingFeishuAppSecret = "feishu_app_secret"
	settingAria2HookSecret = "aria2_hook_secret"
)

// RuntimeServiceConfig 表示可在设置页热更新的外部服务配置。
type RuntimeServiceConfig struct {
	Aria2RPCURL     string `json:"aria2_rpc_url"`
	Aria2RPCSecret  string `json:"aria2_rpc_secret"`
	FeishuAppID     string `json:"feishu_app_id"`
	FeishuAppSecret string `json:"feishu_app_secret"`
	Aria2HookSecret string `json:"aria2_hook_secret"`
}

// GetRuntimeServiceConfig 批量读取运行时服务配置；未保存的字段为空。
func (s *Store) GetRuntimeServiceConfig(ctx context.Context) (RuntimeServiceConfig, error) {
	values, err := s.runtimeServiceConfigValues(ctx)
	if err != nil {
		return RuntimeServiceConfig{}, err
	}
	return runtimeServiceConfigFromValues(values), nil
}

// SaveRuntimeServiceConfig 以一次写入持久化全部运行时服务配置。
func (s *Store) SaveRuntimeServiceConfig(ctx context.Context, cfg RuntimeServiceConfig) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP
	`,
		settingAria2RPCURL, cfg.Aria2RPCURL,
		settingAria2RPCSecret, cfg.Aria2RPCSecret,
		settingFeishuAppID, cfg.FeishuAppID,
		settingFeishuAppSecret, cfg.FeishuAppSecret,
		settingAria2HookSecret, cfg.Aria2HookSecret,
	)
	if err != nil {
		return fmt.Errorf("保存运行时服务配置失败: %w", err)
	}
	return nil
}

// ApplyRuntimeServiceConfig 以已保存的设置覆盖环境配置；未保存字段保持环境值。
func (s *Store) ApplyRuntimeServiceConfig(ctx context.Context, base config.Config) (config.Config, error) {
	values, err := s.runtimeServiceConfigValues(ctx)
	if err != nil {
		return config.Config{}, err
	}
	if value, ok := values[settingAria2RPCURL]; ok {
		base.Aria2RPCURL = value
	}
	if value, ok := values[settingAria2RPCSecret]; ok {
		base.Aria2RPCSecret = value
	}
	if value, ok := values[settingFeishuAppID]; ok {
		base.FeishuAppID = value
	}
	if value, ok := values[settingFeishuAppSecret]; ok {
		base.FeishuAppSecret = value
	}
	if value, ok := values[settingAria2HookSecret]; ok {
		base.Aria2HookSecret = value
	}
	return base, nil
}

func (s *Store) runtimeServiceConfigValues(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, value FROM settings
		WHERE name IN (?, ?, ?, ?, ?)
	`,
		settingAria2RPCURL,
		settingAria2RPCSecret,
		settingFeishuAppID,
		settingFeishuAppSecret,
		settingAria2HookSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("读取运行时服务配置失败: %w", err)
	}
	defer rows.Close()

	values := make(map[string]string, 5)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("读取运行时服务配置失败: %w", err)
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取运行时服务配置失败: %w", err)
	}
	return values, nil
}

func runtimeServiceConfigFromValues(values map[string]string) RuntimeServiceConfig {
	return RuntimeServiceConfig{
		Aria2RPCURL:     values[settingAria2RPCURL],
		Aria2RPCSecret:  values[settingAria2RPCSecret],
		FeishuAppID:     values[settingFeishuAppID],
		FeishuAppSecret: values[settingFeishuAppSecret],
		Aria2HookSecret: values[settingAria2HookSecret],
	}
}
