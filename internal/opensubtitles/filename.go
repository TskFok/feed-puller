package opensubtitles

import (
	"fmt"
	"path/filepath"
	"strings"
)

func SanitizeFileName(name string) (string, error) {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == ".." {
		return "", fmt.Errorf("文件名无效")
	}
	return base, nil
}
