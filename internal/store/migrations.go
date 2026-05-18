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
		INDEX idx_sessions_expires_at (expires_at),
		CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
		INDEX idx_feed_items_status (download_status),
		CONSTRAINT fk_feed_items_subscription FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
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
		INDEX idx_download_tasks_status (status),
		CONSTRAINT fk_download_tasks_item FOREIGN KEY (item_id) REFERENCES feed_items(id) ON DELETE CASCADE,
		CONSTRAINT fk_download_tasks_subscription FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}
