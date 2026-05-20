package app

import (
	"context"
	"strings"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

// ActiveDownloadWithProgress 进行中的下载及 aria2 实时进度。
type ActiveDownloadWithProgress struct {
	ID               int64    `json:"id"`
	ItemID           int64    `json:"item_id"`
	SubscriptionID   int64    `json:"subscription_id"`
	SubscriptionName string   `json:"subscription_name"`
	Title            string   `json:"title"`
	URL              string   `json:"url"`
	Dir              string   `json:"dir"`
	Aria2GID         string   `json:"aria2_gid"`
	SubmittedAt      string   `json:"submitted_at"`
	Aria2Status      string   `json:"aria2_status"`
	CompletedLength  int64    `json:"completed_length"`
	TotalLength      int64    `json:"total_length"`
	DownloadSpeed    int64    `json:"download_speed"`
	ProgressPercent  *float64 `json:"progress_percent,omitempty"`
	StatusError      string   `json:"status_error,omitempty"`
}

// ListActiveDownloadsWithProgress 返回进行中的下载任务及 aria2 进度；查询前会先同步一次 aria2 状态。
func (s *Service) ListActiveDownloadsWithProgress(ctx context.Context) ([]ActiveDownloadWithProgress, error) {
	_ = s.SyncAria2DownloadStatus(ctx)

	rows, err := s.store.ListActiveDownloads(ctx, 100)
	if err != nil {
		return nil, err
	}
	out := make([]ActiveDownloadWithProgress, 0, len(rows))
	for _, row := range rows {
		item := ActiveDownloadWithProgress{
			ID:               row.ID,
			ItemID:           row.ItemID,
			SubscriptionID:   row.SubscriptionID,
			SubscriptionName: row.SubscriptionName,
			Title:            row.Title,
			URL:              row.URL,
			Dir:              row.Dir,
			Aria2GID:         row.Aria2GID,
			SubmittedAt:      row.SubmittedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if strings.TrimSpace(row.Aria2GID) == "" {
			out = append(out, item)
			continue
		}
		task := store.DownloadTask{
			ID:       row.ID,
			ItemID:   row.ItemID,
			Aria2GID: row.Aria2GID,
		}
		effectiveGID, status, err := s.tellStatusForDownloadTask(ctx, task)
		if err != nil {
			item.StatusError = err.Error()
			out = append(out, item)
			continue
		}
		item.Aria2GID = effectiveGID
		progress := downloader.ParseAria2Progress(status)
		item.Aria2Status = progress.Status
		item.CompletedLength = progress.CompletedLength
		item.TotalLength = progress.TotalLength
		item.DownloadSpeed = progress.DownloadSpeed
		item.ProgressPercent = progress.ProgressPercent
		out = append(out, item)
	}
	return out, nil
}
