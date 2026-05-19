export type User = {
  id: number;
  email: string;
  feishu_bound: boolean;
  feishu_name?: string;
  feishu_open_id?: string;
};

export type Subscription = {
  id: number;
  name: string;
  feed_url: string;
  enabled: boolean;
  poll_interval_minutes: number;
  /** 标准五段 crontab（分 时 日 月 周）；非空时优先于 poll_interval_minutes */
  poll_cron: string;
  /** IANA 名称，如 Asia/Shanghai；空串与后端等价于 UTC（仅 cron 模式下使用） */
  poll_cron_timezone: string;
  download_dir: string;
  include_keywords: string;
  exclude_keywords: string;
  use_proxy: boolean;
  /** RSS 解析器：generic | mikan */
  rss_parser: string;
  last_fetched_at?: string;
  last_error?: string;
  created_at?: string;
  /** 服务端根据调度配置推算的下一次计划拉取时间（UTC ISO） */
  next_poll_at?: string;
  /** 列表排序权重，越小越靠前 */
  sort_order?: number;
};

/** 推算下次拉取时间时提交的调度草稿（不含订阅名称等） */
export type PollSchedulePreviewInput = {
  enabled: boolean;
  poll_interval_minutes: number;
  poll_cron: string;
  poll_cron_timezone: string;
  last_fetched_at?: string;
  created_at?: string;
};

export type PollSchedulePreviewResult = {
  next_poll_at?: string;
  error?: string;
};

export type FeedItem = {
  id: number;
  subscription_id: number;
  title: string;
  link?: string;
  download_url?: string;
  download_status: string;
  published_at?: string;
  created_at: string;
  updated_at?: string;
};

/** 拉取订阅后返回的条目预览（含可选的远端文件大小） */
export type PolledFeedItem = FeedItem & {
  content_length?: number | null;
};

export type BatchDownloadFailure = {
  item_id: number;
  error: string;
};

export type BatchDownloadResult = {
  items: FeedItem[];
  failures?: BatchDownloadFailure[];
};

export type BatchStatusResult = {
  items: FeedItem[];
};

export type DownloadTask = {
  id: number;
  item_id: number;
  subscription_id: number;
  url: string;
  dir: string;
  status: string;
  aria2_gid?: string;
  error?: string;
  created_at: string;
};

