package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
)

const aiConfigColumns = `id, name, base_url, model, api_key, created_at, updated_at`

func validateAIConfig(cfg AIConfig) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("API 地址不能为空")
	}
	parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("API 地址格式无效")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("模型不能为空")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	return nil
}

func scanAIConfig(row rowScanner) (AIConfig, error) {
	var cfg AIConfig
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.BaseURL, &cfg.Model, &cfg.APIKey, &cfg.CreatedAt, &cfg.UpdatedAt)
	return cfg, err
}

func scanAIConfigs(rows *sql.Rows) ([]AIConfig, error) {
	configs := make([]AIConfig, 0)
	for rows.Next() {
		cfg, err := scanAIConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

func (s *Store) ListAIConfigs(ctx context.Context) ([]AIConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+aiConfigColumns+`
		FROM ai_configs ORDER BY id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询 AI 配置失败: %w", err)
	}
	defer rows.Close()
	return scanAIConfigs(rows)
}

func (s *Store) GetAIConfig(ctx context.Context, id int64) (AIConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+aiConfigColumns+`
		FROM ai_configs WHERE id = ?
	`, id)
	cfg, err := scanAIConfig(row)
	if err == sql.ErrNoRows {
		return AIConfig{}, fmt.Errorf("AI 配置不存在")
	}
	if err != nil {
		return AIConfig{}, fmt.Errorf("查询 AI 配置失败: %w", err)
	}
	return cfg, nil
}

func (s *Store) CreateAIConfig(ctx context.Context, cfg AIConfig) (AIConfig, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if err := validateAIConfig(cfg); err != nil {
		return AIConfig{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_configs (name, base_url, model, api_key)
		VALUES (?, ?, ?, ?)
	`, cfg.Name, cfg.BaseURL, cfg.Model, cfg.APIKey)
	if err != nil {
		return AIConfig{}, fmt.Errorf("创建 AI 配置失败: %w", err)
	}
	id, _ := result.LastInsertId()
	return s.GetAIConfig(ctx, id)
}

func (s *Store) UpdateAIConfig(ctx context.Context, id int64, cfg AIConfig) (AIConfig, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if err := validateAIConfig(cfg); err != nil {
		return AIConfig{}, err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE ai_configs
		SET name = ?, base_url = ?, model = ?, api_key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, cfg.Name, cfg.BaseURL, cfg.Model, cfg.APIKey, id)
	if err != nil {
		return AIConfig{}, fmt.Errorf("更新 AI 配置失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return AIConfig{}, fmt.Errorf("AI 配置不存在")
	}
	return s.GetAIConfig(ctx, id)
}

func (s *Store) DeleteAIConfig(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除 AI 配置失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("AI 配置不存在")
	}
	return nil
}
