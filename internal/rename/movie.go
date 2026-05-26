package rename

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildMovieTargetPath 生成电影目标路径：{Title} ({Year}).ext
func BuildMovieTargetPath(filePath, title string, year int) (string, error) {
	from := strings.TrimSpace(filePath)
	if from == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}
	title = SanitizeFilename(strings.TrimSpace(title))
	if title == "" {
		return "", fmt.Errorf("电影标题不能为空")
	}
	ext := filepath.Ext(from)
	base := title
	if year > 0 {
		base = fmt.Sprintf("%s (%d)", title, year)
	}
	target := filepath.Join(filepath.Dir(from), base+ext)
	if filepath.Clean(from) == filepath.Clean(target) {
		return from, nil
	}
	return target, nil
}

// SanitizeFilename 移除文件名非法字符。
func SanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		"?", "",
		"*", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}
