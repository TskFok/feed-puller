package downloader

import (
	"path/filepath"
	"strings"
)

// metadataFilenamePrefix 是 aria2 在拉取 BT 元数据时生成的占位文件名前缀。
const metadataFilenamePrefix = "[METADATA]"

// IsMetadataDownloadPath 判断路径是否为 aria2 元数据占位文件（非真实媒体）。
func IsMetadataDownloadPath(path string) bool {
	base := filepath.Base(strings.TrimSpace(path))
	return strings.HasPrefix(base, metadataFilenamePrefix)
}
