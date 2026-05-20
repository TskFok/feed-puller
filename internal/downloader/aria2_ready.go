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
		if !aria2FileEntrySelected(entry) {
			continue
		}
		hasRealFile = true
		if aria2FileEntryComplete(entry) {
			hasCompleteRealFile = true
		}
	}

	if hasCompleteRealFile {
		return true
	}
	// 仅有元数据文件时不要结单。
	if !hasRealFile {
		return false
	}

	// 整体 status 已为 complete 且存在实体文件：用 tellStatus 全局进度兜底，
	// 避免 per-file completedLength 与 length 因舍入/aria2 行为不一致导致永远无法结单。
	progress := ParseAria2Progress(status)
	if progress.TotalLength > 0 && progress.CompletedLength >= progress.TotalLength {
		return true
	}

	// 有实体文件路径但无可用长度信息时，信任整体 complete。
	if progress.TotalLength <= 0 {
		return true
	}
	return false
}

// aria2FileEntrySelected 未标记 selected 或 selected=true 时视为参与下载的文件。
func aria2FileEntrySelected(entry map[string]any) bool {
	raw, ok := entry["selected"]
	if !ok {
		return true
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(strings.ToLower(v)) != "false"
	case bool:
		return v
	default:
		return true
	}
}
