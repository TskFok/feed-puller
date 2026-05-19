package rename

import (
	"regexp"
	"strconv"
	"strings"
)

var localEpisodePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bS\d{1,2}E(\d{1,4})\b`),
	regexp.MustCompile(`(?i)\bEP?\s*(\d{1,4})\b`),
	regexp.MustCompile(`(?i)第\s*(\d{1,4})\s*集`),
	regexp.MustCompile(`[\[\(（【]\s*(\d{1,4})\s*[\]\)）】]`),
	regexp.MustCompile(`(?i)(?:^|[\s_\-])(\d{1,4})(?:[\s_\-\.]|$)`),
}

// DetectEpisodeLocally 尝试从文件名或标题中本地识别集数。
func DetectEpisodeLocally(filename, title string) (int, bool) {
	for _, text := range []string{filename, title} {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		base := text
		if idx := strings.LastIndex(text, "."); idx > 0 {
			base = text[:idx]
		}
		for _, pattern := range localEpisodePatterns {
			match := pattern.FindStringSubmatch(base)
			if len(match) >= 2 {
				n, err := strconv.Atoi(match[1])
				if err == nil && n > 0 {
					return n, true
				}
			}
		}
	}
	return 0, false
}
