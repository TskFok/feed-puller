package store

// subscriptionColumns 为 subscriptions 表 SELECT 列清单（顺序须与 scanSubscription 一致）。
const subscriptionColumns = `id, name, feed_url, enabled, poll_interval_minutes, COALESCE(poll_cron, ''), COALESCE(poll_cron_timezone, 'UTC'), download_dir, COALESCE(include_keywords, ''), COALESCE(exclude_keywords, ''), use_proxy, COALESCE(rss_parser, 'generic'), last_fetched_at, COALESCE(last_error, ''), sort_order, created_at, updated_at`
