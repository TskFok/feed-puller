package rename

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var mediaExtensions = map[string]struct{}{
	".mp4": {}, ".mkv": {}, ".avi": {}, ".wmv": {}, ".flv": {},
	".mov": {}, ".m4v": {}, ".ts": {}, ".webm": {},
}

// FindLargestMediaFileInDir 在目录中查找体积最大的媒体文件（跳过 [METADATA] 与隐藏文件）。
func FindLargestMediaFileInDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("下载目录为空")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("读取下载目录失败: %w", err)
	}
	var bestPath string
	var bestSize int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		if strings.HasPrefix(filepath.Base(name), "[METADATA]") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := mediaExtensions[ext]; !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Size() >= bestSize {
			bestSize = info.Size()
			bestPath = full
		}
	}
	if bestPath == "" {
		return "", fmt.Errorf("目录中未找到可重命名的媒体文件: %s", dir)
	}
	return bestPath, nil
}
