package rename

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var sanitizeFilenameRe = regexp.MustCompile(`[<>:"/\\|?*]`)

// SanitizeFilenamePart 清理文件名中的非法字符（对齐 ani-rename）。
func SanitizeFilenamePart(name string) string {
	s := sanitizeFilenameRe.ReplaceAllString(strings.TrimSpace(name), " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// FormatScrapeName 生成规范文件名：{番剧名} S{季}E{集}{扩展名}。
func FormatScrapeName(animeName string, season, episode int, ext string) string {
	name := SanitizeFilenamePart(animeName)
	if name == "" {
		name = "unknown"
	}
	if season < 1 {
		season = 1
	}
	if episode < 1 {
		episode = 1
	}
	return fmt.Sprintf("%s S%02dE%02d%s", name, season, episode, ext)
}

// BuildScrapeTargetPath 根据原始路径与刮削信息生成目标文件路径。
func BuildScrapeTargetPath(originalPath, animeName string, season, episode int) string {
	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	return filepath.Join(dir, FormatScrapeName(animeName, season, episode, ext))
}
