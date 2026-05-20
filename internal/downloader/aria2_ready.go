package downloader

import "strings"

// IsAria2DownloadReady 判断下载是否真正完成，可用于写库前校验。
// 磁力/BT 在元数据阶段整体 status 可能已是 complete，但仅有 [METADATA] 占位文件，此时返回 false。
func IsAria2DownloadReady(status map[string]any) bool {
	state, _ := ParseAria2TaskStatus(status)
	if state != Aria2TaskComplete {
		return false
	}
	files, ok := status["files"].([]any)
	if !ok || len(files) == 0 {
		return true
	}

	var hasRealFile bool
	var hasCompleteRealFile bool
	var hasRealFileWithLength bool
	onlyMetadata := true

	for _, raw := range files {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if IsMetadataDownloadPath(path) {
			continue
		}
		onlyMetadata = false
		hasRealFile = true
		if aria2Numeric(entry["length"]) > 0 {
			hasRealFileWithLength = true
		}
		if aria2FileEntryComplete(entry) {
			hasCompleteRealFile = true
		}
	}

	if onlyMetadata {
		return false
	}
	if hasCompleteRealFile {
		return true
	}
	if hasRealFileWithLength {
		// 有实体文件且带长度信息，但未下完——即使 status 误报 complete 也不结单。
		return false
	}
	if hasRealFile {
		// 无 per-file 长度时（部分 HTTP 任务），信任整体 status。
		return true
	}
	return true
}
