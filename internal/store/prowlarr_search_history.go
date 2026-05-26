package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const maxProwlarrSearchHistory = 50

// ProwlarrSearchHistory 表示一条 Prowlarr 搜索历史。
type ProwlarrSearchHistory struct {
	ID           int64     `json:"id"`
	DisplayQuery string    `json:"display_query"`
	Query        string    `json:"query"`
	MediaType    string    `json:"media_type"`
	SortBy       string    `json:"sort_by"`
	IndexerIDs   []int64   `json:"indexer_ids"`
	ResultCount  int       `json:"result_count"`
	SearchedAt   time.Time `json:"searched_at"`
}

func (s *Store) RecordProwlarrSearchHistory(ctx context.Context, entry ProwlarrSearchHistory) error {
	displayQuery := strings.TrimSpace(entry.DisplayQuery)
	query := strings.TrimSpace(entry.Query)
	mediaType := strings.TrimSpace(entry.MediaType)
	if mediaType == "" {
		mediaType = ProwlarrMediaMovie
	}
	sortBy := strings.TrimSpace(entry.SortBy)
	if sortBy == "" {
		sortBy = "seeders"
	}
	if displayQuery == "" || query == "" {
		return fmt.Errorf("搜索关键词不能为空")
	}
	indexerJSON := EncodeProwlarrIndexerIDs(entry.IndexerIDs)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO prowlarr_search_history (display_query, query, media_type, sort_by, indexer_ids, result_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			display_query = VALUES(display_query),
			sort_by = VALUES(sort_by),
			indexer_ids = VALUES(indexer_ids),
			result_count = VALUES(result_count),
			updated_at = CURRENT_TIMESTAMP
	`, displayQuery, query, mediaType, sortBy, indexerJSON, entry.ResultCount)
	if err != nil {
		return fmt.Errorf("保存搜索历史失败: %w", err)
	}
	if err := s.trimProwlarrSearchHistory(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ListProwlarrSearchHistory(ctx context.Context, limit int) ([]ProwlarrSearchHistory, error) {
	if limit <= 0 || limit > maxProwlarrSearchHistory {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, display_query, query, media_type, sort_by, COALESCE(indexer_ids, '[]'), result_count, updated_at
		FROM prowlarr_search_history
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询搜索历史失败: %w", err)
	}
	defer rows.Close()
	out := make([]ProwlarrSearchHistory, 0)
	for rows.Next() {
		var row ProwlarrSearchHistory
		var indexerRaw string
		if err := rows.Scan(&row.ID, &row.DisplayQuery, &row.Query, &row.MediaType, &row.SortBy, &indexerRaw, &row.ResultCount, &row.SearchedAt); err != nil {
			return nil, err
		}
		ids, err := ParseProwlarrIndexerIDs(indexerRaw)
		if err != nil {
			return nil, err
		}
		row.IndexerIDs = ids
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) DeleteProwlarrSearchHistory(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("历史记录 ID 无效")
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM prowlarr_search_history WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除搜索历史失败: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ClearProwlarrSearchHistory(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM prowlarr_search_history`); err != nil {
		return fmt.Errorf("清空搜索历史失败: %w", err)
	}
	return nil
}

func (s *Store) trimProwlarrSearchHistory(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM prowlarr_search_history
		WHERE id NOT IN (
			SELECT id FROM (
				SELECT id FROM prowlarr_search_history ORDER BY updated_at DESC LIMIT ?
			) AS recent
		)
	`, maxProwlarrSearchHistory)
	if err != nil {
		return fmt.Errorf("裁剪搜索历史失败: %w", err)
	}
	return nil
}

func IsProwlarrSearchHistoryNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
