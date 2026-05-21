package store

const downloadTaskColumns = `id, item_id, subscription_id, url, dir, status, COALESCE(aria2_gid, ''), COALESCE(error, ''), COALESCE(final_path, ''), created_at, updated_at`
