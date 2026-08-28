package opensubtitles

import (
	"path/filepath"
	"strings"
)

func SanitizeFileName(name string) (string, error) {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == ".." {
		return "", ErrInvalidFileName
	}
	return base, nil
}
