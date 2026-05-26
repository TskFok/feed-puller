package prowlarr

import (
	"fmt"
	"strings"
)

// ResolveTorrentURL 将 release 解析为 aria2 可用的 Torrent 地址（优先 magnet）。
func ResolveTorrentURL(release Release) (string, error) {
	hash := strings.TrimSpace(release.InfoHash)
	if hash != "" {
		return "magnet:?xt=urn:btih:" + hash, nil
	}
	url := strings.TrimSpace(release.DownloadURL)
	if url != "" {
		return url, nil
	}
	return "", fmt.Errorf("无可用 Torrent 地址")
}
