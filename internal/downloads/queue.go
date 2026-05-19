package downloads

type DownloadStatus string

const (
	StatusPending    DownloadStatus = "pending"
	StatusSubmitting DownloadStatus = "submitting"
	StatusSubmitted  DownloadStatus = "submitted"
	StatusFailed     DownloadStatus = "failed"
	StatusSkipped    DownloadStatus = "skipped"
	StatusCompleted  DownloadStatus = "completed"
)

type FeedItem struct {
	ID             int64
	DownloadURL    string
	DownloadStatus DownloadStatus
}

type Job struct {
	ItemID int64
	URL    string
}

func PlanDownloads(items []FeedItem) []Job {
	jobs := make([]Job, 0, len(items))
	for _, item := range items {
		if item.DownloadURL == "" {
			continue
		}
		if item.DownloadStatus != "" && item.DownloadStatus != StatusPending && item.DownloadStatus != StatusFailed {
			continue
		}
		jobs = append(jobs, Job{ItemID: item.ID, URL: item.DownloadURL})
	}
	return jobs
}
