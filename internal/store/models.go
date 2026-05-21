package store

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FeishuOpenID string    `json:"feishu_open_id,omitempty"`
	FeishuName   string    `json:"feishu_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Subscription struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	FeedURL             string     `json:"feed_url"`
	Enabled             bool       `json:"enabled"`
	PollIntervalMinutes int        `json:"poll_interval_minutes"`
	PollCron            string     `json:"poll_cron"`
	PollCronTimezone    string     `json:"poll_cron_timezone"`
	DownloadDir         string     `json:"download_dir"`
	IncludeKeywords     string     `json:"include_keywords"`
	ExcludeKeywords     string     `json:"exclude_keywords"`
	UseProxy            bool       `json:"use_proxy"`
	RSSParser           string     `json:"rss_parser"`
	AIRenameEnabled     bool       `json:"ai_rename_enabled"`
	AIRenameSeason      int        `json:"ai_rename_season"`
	AIRenameEpOffset    int        `json:"ai_rename_episode_offset"`
	LastFetchedAt       *time.Time `json:"last_fetched_at,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
	SortOrder           int        `json:"sort_order"`
	NextPollAt          *time.Time `json:"next_poll_at,omitempty"` // 由 ApplySubscriptionNextPoll 计算，不入库
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type Item struct {
	ID             int64      `json:"id"`
	SubscriptionID int64      `json:"subscription_id"`
	GUID           string     `json:"guid,omitempty"`
	Title          string     `json:"title"`
	Link           string     `json:"link,omitempty"`
	DownloadURL    string     `json:"download_url,omitempty"`
	DedupeKey      string     `json:"dedupe_key"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	DownloadStatus string     `json:"download_status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DownloadTask struct {
	ID             int64     `json:"id"`
	ItemID         int64     `json:"item_id"`
	SubscriptionID int64     `json:"subscription_id"`
	URL            string    `json:"url"`
	Dir            string    `json:"dir"`
	Status         string    `json:"status"`
	Aria2GID       string    `json:"aria2_gid,omitempty"`
	Error          string    `json:"error,omitempty"`
	FinalPath      string    `json:"final_path,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ActiveDownload 表示进行中的下载任务（status=submitted，已提交 aria2）。
type ActiveDownload struct {
	ID               int64     `json:"id"`
	ItemID           int64     `json:"item_id"`
	SubscriptionID   int64     `json:"subscription_id"`
	SubscriptionName string    `json:"subscription_name"`
	Title            string    `json:"title"`
	URL              string    `json:"url"`
	Dir              string    `json:"dir"`
	Aria2GID         string    `json:"aria2_gid"`
	SubmittedAt      time.Time `json:"submitted_at"`
}

// CompletedDownload 表示一条已完成的 aria2 下载记录（用于列表展示）。
type CompletedDownload struct {
	ID               int64     `json:"id"`
	ItemID           int64     `json:"item_id"`
	SubscriptionID   int64     `json:"subscription_id"`
	SubscriptionName string    `json:"subscription_name"`
	Title            string    `json:"title"`
	URL              string    `json:"url"`
	Dir              string    `json:"dir"`
	FinalPath        string    `json:"final_path,omitempty"`
	AIRenameEnabled  bool      `json:"ai_rename_enabled"`
	CompletedAt      time.Time `json:"completed_at"`
}

type PendingDownload struct {
	ItemID         int64
	SubscriptionID int64
	URL            string
	Dir            string
}

// AIConfig 表示一条 OpenAI 兼容的 AI 服务配置。
type AIConfig struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"url"`
	Model     string    `json:"model"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
