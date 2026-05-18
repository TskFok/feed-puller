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
  download_dir: string;
  use_proxy: boolean;
  last_fetched_at?: string;
  last_error?: string;
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

