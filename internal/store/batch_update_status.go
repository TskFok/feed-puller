package store

import (
	"context"
	"fmt"
	"strings"
)

const maxBatchItemStatusUpdates = 50

// BatchUpdateItemDownloadStatus 批量将条目标记为未处理（pending）或已处理（有下载地址为 submitted，否则为 skipped）。
func (s *Store) BatchUpdateItemDownloadStatus(ctx context.Context, itemIDs []int64, target string) ([]Item, error) {
	target = strings.TrimSpace(target)
	if target != "pending" && target != "submitted" {
		return nil, fmt.Errorf("无效的状态，仅支持 pending 或 submitted")
	}
	if len(itemIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一条条目")
	}
	if len(itemIDs) > maxBatchItemStatusUpdates {
		return nil, fmt.Errorf("单次最多更新 %d 条", maxBatchItemStatusUpdates)
	}
	seen := make(map[int64]struct{}, len(itemIDs))
	out := make([]Item, 0, len(itemIDs))
	for _, id := range itemIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		item, err := s.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		if item.DownloadStatus == "submitting" {
			return nil, fmt.Errorf("条目 #%d 正在提交下载，请稍候", id)
		}
		next := resolveBatchStatusTarget(item, target)
		if item.DownloadStatus == next {
			out = append(out, item)
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE feed_items SET download_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, next, id); err != nil {
			return nil, fmt.Errorf("更新条目状态失败: %w", err)
		}
		updated, err := s.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

func resolveBatchStatusTarget(item Item, target string) string {
	if target == "pending" {
		return "pending"
	}
	if strings.TrimSpace(item.DownloadURL) == "" {
		return "skipped"
	}
	return "submitted"
}
