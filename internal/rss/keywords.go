package rss

import (
	"fmt"
	"regexp"
	"strings"
)

// CompileKeywordPatterns 将多行文本编译为正则列表，空行忽略；raw 为空时返回 nil。
func CompileKeywordPatterns(raw string) ([]*regexp.Regexp, error) {
	lines := splitPatternLines(raw)
	if len(lines) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(lines))
	for i, line := range lines {
		re, err := regexp.Compile(line)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行无效: %w", i+1, err)
		}
		out = append(out, re)
	}
	return out, nil
}

// ValidateKeywordPatterns 校验包含/排除字段中的正则是否可编译。
func ValidateKeywordPatterns(includeRaw, excludeRaw string) error {
	if _, err := CompileKeywordPatterns(includeRaw); err != nil {
		return fmt.Errorf("包含关键字: %w", err)
	}
	if _, err := CompileKeywordPatterns(excludeRaw); err != nil {
		return fmt.Errorf("排除关键字: %w", err)
	}
	return nil
}

// FilterFeedItems 按订阅的关键字规则过滤条目（每行一条正则）。
// 排除优先：命中任一排除正则则丢弃；若配置了包含正则，须至少命中一条包含正则。
// 匹配文本为标题、链接与下载地址拼接。
func FilterFeedItems(items []FeedItem, includeRaw, excludeRaw string) ([]FeedItem, error) {
	include, err := CompileKeywordPatterns(includeRaw)
	if err != nil {
		return nil, err
	}
	exclude, err := CompileKeywordPatterns(excludeRaw)
	if err != nil {
		return nil, err
	}
	out := make([]FeedItem, 0, len(items))
	for _, item := range items {
		if matchesKeywordFilters(itemKeywordHaystack(item), include, exclude) {
			out = append(out, item)
		}
	}
	return out, nil
}

func splitPatternLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func itemKeywordHaystack(item FeedItem) string {
	return strings.TrimSpace(item.Title) + "\n" + strings.TrimSpace(item.Link) + "\n" + strings.TrimSpace(item.DownloadURL)
}

func matchesKeywordFilters(haystack string, include, exclude []*regexp.Regexp) bool {
	for _, re := range exclude {
		if re.MatchString(haystack) {
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, re := range include {
		if re.MatchString(haystack) {
			return true
		}
	}
	return false
}
