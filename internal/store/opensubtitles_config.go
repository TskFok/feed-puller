package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	settingOpenSubtitlesUsername    = "opensubtitles_username"
	settingOpenSubtitlesPassword    = "opensubtitles_password"
	settingOpenSubtitlesAPIKey      = "opensubtitles_api_key"
	settingOpenSubtitlesDownloadDir = "opensubtitles_download_dir"
)

var ErrOpenSubtitlesConfigIncomplete = errors.New("请填写 OpenSubtitles 用户名、密码、API Key 和下载目录")

type OpenSubtitlesConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	APIKey      string `json:"api_key"`
	DownloadDir string `json:"download_dir"`
	Configured  bool   `json:"configured"`
}

func (s *Store) GetOpenSubtitlesConfig(ctx context.Context) (OpenSubtitlesConfig, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`,
		settingOpenSubtitlesUsername, settingOpenSubtitlesPassword, settingOpenSubtitlesAPIKey, settingOpenSubtitlesDownloadDir)
	if err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
	}
	cfg := OpenSubtitlesConfig{
		Username:    strings.TrimSpace(values[settingOpenSubtitlesUsername]),
		Password:    strings.TrimSpace(values[settingOpenSubtitlesPassword]),
		APIKey:      strings.TrimSpace(values[settingOpenSubtitlesAPIKey]),
		DownloadDir: strings.TrimSpace(values[settingOpenSubtitlesDownloadDir]),
	}
	cfg.Configured = cfg.Username != "" && cfg.Password != "" && cfg.APIKey != "" && cfg.DownloadDir != ""
	return cfg, nil
}

func (s *Store) SaveOpenSubtitlesConfig(ctx context.Context, cfg OpenSubtitlesConfig) (OpenSubtitlesConfig, error) {
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.DownloadDir = strings.TrimSpace(cfg.DownloadDir)
	if cfg.Username == "" || cfg.Password == "" || cfg.APIKey == "" || cfg.DownloadDir == "" {
		return OpenSubtitlesConfig{}, ErrOpenSubtitlesConfigIncomplete
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`,
		settingOpenSubtitlesUsername, cfg.Username,
		settingOpenSubtitlesPassword, cfg.Password,
		settingOpenSubtitlesAPIKey, cfg.APIKey,
		settingOpenSubtitlesDownloadDir, cfg.DownloadDir,
	)
	if err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("保存 OpenSubtitles 配置失败: %w", err)
	}
	return s.GetOpenSubtitlesConfig(ctx)
}
