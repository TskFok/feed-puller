package rename

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildTVTargetPath 生成剧集目标路径：{Show} - SxxExx.ext
func BuildTVTargetPath(filePath, showTitle string, season, episode int) (string, error) {
	from := strings.TrimSpace(filePath)
	if from == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}
	showTitle = SanitizeFilename(strings.TrimSpace(showTitle))
	if showTitle == "" {
		return "", fmt.Errorf("剧集名称不能为空")
	}
	if season < 1 {
		season = 1
	}
	if episode < 1 {
		return "", fmt.Errorf("集数无效")
	}
	ext := filepath.Ext(from)
	base := fmt.Sprintf("%s - S%02dE%02d", showTitle, season, episode)
	target := filepath.Join(filepath.Dir(from), base+ext)
	if filepath.Clean(from) == filepath.Clean(target) {
		return from, nil
	}
	return target, nil
}
