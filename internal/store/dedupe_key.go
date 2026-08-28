package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

// MaxDedupeKeyLength 对应 feed_items.dedupe_key 的 VARCHAR(768) 上限。
const MaxDedupeKeyLength = 768

const hashedDedupeKeyPrefix = "sha256:"

// NormalizeDedupeKey 将去重键限制在列长以内：未超长则原样返回，超长则改为 SHA-256 摘要。
func NormalizeDedupeKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if utf8.RuneCountInString(key) <= MaxDedupeKeyLength {
		return key
	}
	sum := sha256.Sum256([]byte(key))
	return hashedDedupeKeyPrefix + hex.EncodeToString(sum[:])
}

// ProwlarrDedupeKey 由 Prowlarr release guid 生成 feed_items.dedupe_key。
func ProwlarrDedupeKey(guid string) string {
	return NormalizeDedupeKey("prowlarr:" + strings.TrimSpace(guid))
}
