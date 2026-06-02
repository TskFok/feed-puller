export type User = {
  id: number;
  email: string;
  feishu_bound: boolean;
  feishu_name?: string;
  feishu_open_id?: string;
};

export type AuthOptions = {
  password_login_enabled: boolean;
  feishu_login_enabled: boolean;
};

export type FeishuNotifyConfig = {
  feishu_notify_type: '' | 'webhook' | 'api';
  feishu_bot_webhook: string;
  feishu_receive_open_id: string;
  feishu_receive_targets: string;
  feishu_complete_title: string;
  feishu_fail_title: string;
  feishu_prowlarr_complete_title: string;
  feishu_prowlarr_fail_title: string;
  feishu_prowlarr_complete_body: string;
  feishu_prowlarr_fail_body: string;
  feishu_include_subscription: boolean;
  feishu_include_title: boolean;
  feishu_include_path: boolean;
  feishu_notify_on_fail: boolean;
  feishu_use_interactive_card: boolean;
  feishu_batch_window_seconds: number;
  configured: boolean;
};

export type FeishuNotifyHistory = {
  id: number;
  event_type: 'complete' | 'fail' | 'test';
  source: 'rss' | 'prowlarr' | 'test';
  notify_type: string;
  title: string;
  content: string;
  item_count: number;
  status: 'sent' | 'failed';
  error?: string;
  created_at: string;
};

export type RenameHistory = {
  id: number;
  subscription_id?: number;
  original_filename: string;
  original_path: string;
  renamed_path?: string;
  ai_prompt: string;
  ai_response?: string;
  status: 'success' | 'skipped' | 'failed';
  error?: string;
  created_at: string;
};

export type PaginatedResult<T> = {
  items: T[];
  total: number;
  page: number;
  page_size: number;
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
  /** 下载完成后是否使用 AI 重命名为 SxxExx 格式 */
  ai_rename_enabled: boolean;
  /** 刮削用季度，从 1 开始 */
  ai_rename_season: number;
  /** 集数偏移，可为负数，默认 0 */
  ai_rename_episode_offset: number;
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

export type ActiveDownload = {
  id: number;
  item_id: number;
  subscription_id: number;
  subscription_name: string;
  title: string;
  url: string;
  dir: string;
  aria2_gid: string;
  submitted_at: string;
  aria2_status: string;
  completed_length: number;
  total_length: number;
  download_speed: number;
  progress_percent?: number | null;
  status_error?: string;
};

export type CompletedDownload = {
  id: number;
  item_id: number;
  subscription_id: number;
  subscription_name: string;
  title: string;
  url: string;
  dir: string;
  final_path?: string;
  ai_rename_enabled: boolean;
  completed_at: string;
};

export type RenameDownloadResult = {
  from_path?: string;
  to_path?: string;
  skipped?: boolean;
  message?: string;
};

export type AIConfig = {
  id: number;
  /** 模型名称（展示用） */
  name: string;
  /** API 基础地址 */
  url: string;
  /** 模型标识 */
  model: string;
  api_key: string;
  request_options: string;
  created_at?: string;
  updated_at?: string;
};

export type AIConfigTestResult = {
  ok: boolean;
  message?: string;
  error?: string;
};

export type AIConfigModelsResult = {
  models: string[];
};

export type ProwlarrConfig = {
  url: string;
  api_key: string;
  download_dir: string;
  tv_download_dir: string;
  movie_rename_enabled: boolean;
  tmdb_api_key: string;
  indexer_ids: number[];
  subscription_id?: number;
  tv_subscription_id?: number;
  configured: boolean;
};

export type ProwlarrIndexer = {
  id: number;
  name: string;
  enable: boolean;
  protocol: string;
};

export type ProwlarrSearchType = 'movie' | 'tv';
export type ProwlarrSortBy = 'seeders' | 'size' | 'date';

export type ProwlarrTestResult = {
  ok: boolean;
  message?: string;
  error?: string;
};

export type ProwlarrRelease = {
  guid: string;
  title: string;
  indexer: string;
  indexerId: number;
  size: number;
  seeders: number;
  leechers: number;
  publishDate?: string;
  downloadUrl?: string;
  infoUrl?: string;
  infoHash?: string;
  protocol: string;
  imdbId?: number;
  tmdbId?: number;
  tvdbId?: number;
  season?: number;
  episode?: number;
};

export type ProwlarrSearchResult = {
  items: ProwlarrRelease[];
};

export type ProwlarrDownloadInput = {
  guid: string;
  title: string;
  media_type?: ProwlarrSearchType;
  download_url?: string;
  info_hash?: string;
  indexer_id?: number;
  imdb_id?: number;
  tmdb_id?: number;
  tvdb_id?: number;
  season?: number;
  episode?: number;
};

export type ProwlarrIndexerList = {
  items: ProwlarrIndexer[];
};

export type ProwlarrSearchHistory = {
  id: number;
  display_query: string;
  query: string;
  media_type: ProwlarrSearchType;
  sort_by: ProwlarrSortBy;
  indexer_ids: number[];
  result_count: number;
  searched_at: string;
};

export type ProwlarrSearchHistoryDetail = ProwlarrSearchHistory & {
  results: ProwlarrRelease[];
};

export type ProwlarrBatchDownloadFailure = {
  guid: string;
  error: string;
};

export type ProwlarrBatchDownloadResult = {
  items: FeedItem[];
  failures?: ProwlarrBatchDownloadFailure[];
};

export type ProwlarrSubmittedGuidsResult = {
  guids: string[];
};
