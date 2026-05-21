package store

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		feishu_open_id VARCHAR(128) NULL UNIQUE,
		feishu_name VARCHAR(255) NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token_hash CHAR(64) PRIMARY KEY,
		user_id BIGINT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_sessions_user_id (user_id),
		INDEX idx_sessions_expires_at (expires_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS settings (
		name VARCHAR(128) PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS subscriptions (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL,
		feed_url TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		poll_interval_minutes INT NOT NULL DEFAULT 30,
		download_dir TEXT NOT NULL,
		include_keywords TEXT NULL,
		exclude_keywords TEXT NULL,
		use_proxy BOOLEAN NOT NULL DEFAULT FALSE,
		last_fetched_at TIMESTAMP NULL,
		last_error TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_subscriptions_enabled (enabled),
		INDEX idx_subscriptions_last_fetched_at (last_fetched_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS feed_items (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		subscription_id BIGINT NOT NULL,
		guid TEXT NULL,
		title TEXT NOT NULL,
		link TEXT NULL,
		download_url TEXT NULL,
		dedupe_key VARCHAR(768) NOT NULL,
		published_at TIMESTAMP NULL,
		download_status ENUM('pending', 'submitting', 'submitted', 'failed', 'skipped') NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY uniq_feed_items_subscription_dedupe (subscription_id, dedupe_key),
		INDEX idx_feed_items_status (download_status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS download_tasks (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		item_id BIGINT NOT NULL,
		subscription_id BIGINT NOT NULL,
		url TEXT NOT NULL,
		dir TEXT NOT NULL,
		status ENUM('pending', 'submitting', 'submitted', 'failed', 'skipped') NOT NULL,
		aria2_gid VARCHAR(128) NULL,
		error TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_download_tasks_item_id (item_id),
		INDEX idx_download_tasks_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS include_keywords TEXT NULL,
		ADD COLUMN IF NOT EXISTS exclude_keywords TEXT NULL`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS poll_cron VARCHAR(512) NOT NULL DEFAULT ''`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS poll_cron_timezone VARCHAR(128) NOT NULL DEFAULT 'UTC'`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0`,
	`UPDATE subscriptions SET sort_order = -id WHERE sort_order = 0 AND id > 0`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS rss_parser VARCHAR(32) NOT NULL DEFAULT 'generic'`,
	`ALTER TABLE feed_items
		MODIFY COLUMN download_status ENUM('pending', 'submitting', 'submitted', 'failed', 'skipped', 'completed') NOT NULL DEFAULT 'pending'`,
	`ALTER TABLE download_tasks
		MODIFY COLUMN status ENUM('pending', 'submitting', 'submitted', 'failed', 'skipped', 'completed') NOT NULL`,
	`CREATE TABLE IF NOT EXISTS ai_configs (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL,
		base_url TEXT NOT NULL,
		model VARCHAR(255) NOT NULL,
		api_key TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`ALTER TABLE subscriptions
		ADD COLUMN IF NOT EXISTS ai_rename_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		ADD COLUMN IF NOT EXISTS ai_rename_season INT NOT NULL DEFAULT 1,
		ADD COLUMN IF NOT EXISTS ai_rename_episode_offset INT NOT NULL DEFAULT 0`,
	`ALTER TABLE download_tasks
		ADD COLUMN IF NOT EXISTS final_path TEXT NULL`,
}
