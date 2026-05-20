package downloader

import (
	"fmt"
	"strings"
)

// Aria2DownloadPath 从 aria2.tellStatus 结果中提取用于后处理的真实文件路径。
// 磁力/BT 任务会先出现 [METADATA] 占位文件，此处会跳过并优先返回已完成的非元数据文件。
func Aria2DownloadPath(status map[string]any) (string, error) {
	files, ok := status["files"].([]any)
	if !ok || len(files) == 0 {
		return "", fmt.Errorf("aria2 响应缺少 files")
	}
	var fallback string
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
		fallback = path
		if aria2FileEntryComplete(entry) {
			return path, nil
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	// 仅有元数据文件时回退到首个路径，便于上层记录/排错。
	first, ok := files[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("aria2 files 格式无效")
	}
	path, _ := first["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("aria2 文件路径为空")
	}
	return path, nil
}

func aria2FileEntryComplete(entry map[string]any) bool {
	total := aria2Numeric(entry["length"])
	if total <= 0 {
		return false
	}
	return aria2Numeric(entry["completedLength"]) >= total
}
