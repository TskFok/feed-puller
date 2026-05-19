package downloader

import (
	"fmt"
	"strings"
)

// Aria2DownloadPath 从 aria2.tellStatus 结果中提取首个已完成文件的路径。
func Aria2DownloadPath(status map[string]any) (string, error) {
	files, ok := status["files"].([]any)
	if !ok || len(files) == 0 {
		return "", fmt.Errorf("aria2 响应缺少 files")
	}
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
