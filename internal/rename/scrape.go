package rename

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var episodeSuffixPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\s*[\[\(（【]\s*\d{1,4}\s*[\]\)）】]\s*$`),
	regexp.MustCompile(`(?i)\s*-\s*\d{1,4}\s*$`),
	regexp.MustCompile(`(?i)\s*_\s*\d{1,4}\s*$`),
	regexp.MustCompile(`(?i)\s*第\s*\d{1,4}\s*集\s*$`),
	regexp.MustCompile(`(?i)\s*EP\s*\d{1,4}\s*$`),
	regexp.MustCompile(`(?i)\s*S\d{1,2}E\d{1,4}\s*$`),
	regexp.MustCompile(`(?i)\s+\d{1,4}\s*$`),
}

// StripEpisodeSuffix 去掉文件名主体末尾常见的集数标记。
func StripEpisodeSuffix(nameWithoutExt string) string {
	cleaned := strings.TrimSpace(nameWithoutExt)
	for {
		next := cleaned
		for _, pattern := range episodeSuffixPatterns {
			next = pattern.ReplaceAllString(next, "")
		}
		next = strings.TrimSpace(next)
		if next == cleaned {
			return cleaned
		}
		cleaned = next
	}
}

// BuildScrapeFilename 根据原始路径、季度与集数生成 SxxExx 刮削文件名。
func BuildScrapeFilename(originalPath string, season, episode int) string {
	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	cleaned := StripEpisodeSuffix(nameWithoutExt)
	if cleaned == "" {
		cleaned = nameWithoutExt
	}
	newBase := fmt.Sprintf("%s S%02dE%02d%s", cleaned, season, episode, ext)
	return filepath.Join(dir, newBase)
}

// RenameFile 将文件重命名为目标路径；目标已存在时返回错误。
func RenameFile(fromPath, toPath string) error {
	fromPath = strings.TrimSpace(fromPath)
	toPath = strings.TrimSpace(toPath)
	if fromPath == "" || toPath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if fromPath == toPath {
		return nil
	}
	if _, err := os.Stat(fromPath); err != nil {
		return fmt.Errorf("源文件不存在: %w", err)
	}
	if _, err := os.Stat(toPath); err == nil {
		return fmt.Errorf("目标文件已存在: %s", toPath)
	}
	if err := os.Rename(fromPath, toPath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	return nil
}

// FinalEpisode 计算应用偏移后的最终集数。
func FinalEpisode(detectedEpisode, offset int) (int, error) {
	final := detectedEpisode + offset
	if final < 1 {
		return 0, fmt.Errorf("偏移后集数无效: %d", final)
	}
	return final, nil
}
