package prowlarr

import (
	"fmt"
	"regexp"
	"strings"
)

var tvdbIDPattern = regexp.MustCompile(`^(?:tvdb[:\s-]*)?(\d+)$`)

// NormalizeSearchQuery 将用户输入规范化为 Prowlarr 搜索语法。
func NormalizeSearchQuery(raw string, searchType SearchType) string {
	query := strings.TrimSpace(raw)
	if query == "" {
		return ""
	}
	lower := strings.ToLower(query)
	if strings.HasPrefix(lower, "{") {
		return query
	}
	if imdb := normalizeIMDbQuery(lower); imdb != "" {
		return imdb
	}
	if searchType == SearchTypeTV {
		if tvdb := normalizeTVDBQuery(query); tvdb != "" {
			return tvdb
		}
	}
	return query
}

func normalizeIMDbQuery(lower string) string {
	if !strings.HasPrefix(lower, "tt") {
		return ""
	}
	digits := strings.TrimPrefix(lower, "tt")
	if digits == "" || !regexp.MustCompile(`^\d+$`).MatchString(digits) {
		return ""
	}
	return "{ImdbId:" + lower + "}"
}

func normalizeTVDBQuery(raw string) string {
	match := tvdbIDPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 2 {
		return ""
	}
	return "{TvdbId:" + match[1] + "}"
}

func FormatIMDbID(id int64) string {
	if id <= 0 {
		return ""
	}
	return fmt.Sprintf("tt%d", id)
}
