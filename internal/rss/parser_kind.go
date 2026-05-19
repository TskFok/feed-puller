package rss

import (
	"fmt"
	"strings"
)

const (
	ParserGeneric = "generic"
	ParserMikan   = "mikan"
)

// NormalizeParser 将空值或未识别的解析器归一为 generic。
func NormalizeParser(parser string) string {
	switch strings.ToLower(strings.TrimSpace(parser)) {
	case ParserMikan:
		return ParserMikan
	default:
		return ParserGeneric
	}
}

func ValidateParser(parser string) error {
	switch strings.ToLower(strings.TrimSpace(parser)) {
	case "", ParserGeneric, ParserMikan:
		return nil
	default:
		return fmt.Errorf("不支持的 RSS 解析器: %s", parser)
	}
}
