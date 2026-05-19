package rss

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MagnetFromTorrent 从 .torrent 文件内容生成 magnet 链接。
func MagnetFromTorrent(data []byte) (string, error) {
	decoded, err := bencodeDecode(data)
	if err != nil {
		return "", fmt.Errorf("解析 torrent 失败: %w", err)
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return "", fmt.Errorf("torrent 根节点无效")
	}
	info, ok := root["info"]
	if !ok {
		return "", fmt.Errorf("torrent 缺少 info")
	}
	encoded, err := bencodeEncode(info)
	if err != nil {
		return "", fmt.Errorf("编码 torrent info 失败: %w", err)
	}
	sum := sha1.Sum(encoded)
	infoHash := strings.ToUpper(fmt.Sprintf("%x", sum))

	displayName := torrentDisplayName(info)
	query := url.Values{}
	query.Set("xt", "urn:btih:"+infoHash)
	if displayName != "" {
		query.Set("dn", displayName)
	}
	return "magnet:?" + query.Encode(), nil
}

func torrentDisplayName(info any) string {
	infoMap, ok := info.(map[string]any)
	if !ok {
		return ""
	}
	name, ok := infoMap["name"]
	if !ok {
		return ""
	}
	switch v := name.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func bencodeDecode(data []byte) (any, error) {
	v, _, err := decodeBencodeValue(data, 0)
	return v, err
}

func decodeBencodeValue(data []byte, i int) (any, int, error) {
	if i >= len(data) {
		return nil, i, fmt.Errorf("意外的输入结束")
	}
	switch data[i] {
	case 'i':
		return decodeBencodeInt(data, i+1)
	case 'l':
		return decodeBencodeList(data, i+1)
	case 'd':
		return decodeBencodeDict(data, i+1)
	default:
		if data[i] >= '0' && data[i] <= '9' {
			return decodeBencodeString(data, i)
		}
		return nil, i, fmt.Errorf("无效的 bencode 前缀 %q", data[i])
	}
}

func decodeBencodeInt(data []byte, i int) (int64, int, error) {
	end := bytes.IndexByte(data[i:], 'e')
	if end < 0 {
		return 0, i, fmt.Errorf("整数未闭合")
	}
	end += i
	n, err := strconv.ParseInt(string(data[i:end]), 10, 64)
	if err != nil {
		return 0, i, err
	}
	return n, end + 1, nil
}

func decodeBencodeString(data []byte, i int) (string, int, error) {
	colon := bytes.IndexByte(data[i:], ':')
	if colon < 0 {
		return "", i, fmt.Errorf("字符串长度无效")
	}
	colon += i
	var length int
	if _, err := fmt.Sscanf(string(data[i:colon]), "%d", &length); err != nil {
		return "", i, err
	}
	start := colon + 1
	end := start + length
	if end > len(data) {
		return "", i, fmt.Errorf("字符串越界")
	}
	return string(data[start:end]), end, nil
}

func decodeBencodeList(data []byte, i int) ([]any, int, error) {
	var list []any
	for i < len(data) && data[i] != 'e' {
		v, next, err := decodeBencodeValue(data, i)
		if err != nil {
			return nil, i, err
		}
		list = append(list, v)
		i = next
	}
	if i >= len(data) || data[i] != 'e' {
		return nil, i, fmt.Errorf("列表未闭合")
	}
	return list, i + 1, nil
}

func decodeBencodeDict(data []byte, i int) (map[string]any, int, error) {
	out := make(map[string]any)
	for i < len(data) && data[i] != 'e' {
		key, next, err := decodeBencodeString(data, i)
		if err != nil {
			return nil, i, err
		}
		val, next, err := decodeBencodeValue(data, next)
		if err != nil {
			return nil, i, err
		}
		out[key] = val
		i = next
	}
	if i >= len(data) || data[i] != 'e' {
		return nil, i, fmt.Errorf("字典未闭合")
	}
	return out, i + 1, nil
}

func bencodeEncode(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeBencode(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeBencode(buf *bytes.Buffer, v any) error {
	switch value := v.(type) {
	case int64:
		fmt.Fprintf(buf, "i%de", value)
	case int:
		fmt.Fprintf(buf, "i%de", value)
	case string:
		fmt.Fprintf(buf, "%d:%s", len(value), value)
	case []byte:
		fmt.Fprintf(buf, "%d:%s", len(value), value)
	case []any:
		buf.WriteByte('l')
		for _, item := range value {
			if err := writeBencode(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	case map[string]any:
		buf.WriteByte('d')
		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			if err := writeBencode(buf, k); err != nil {
				return err
			}
			if err := writeBencode(buf, value[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	default:
		return fmt.Errorf("不支持的 bencode 类型 %T", v)
	}
	return nil
}

func sortStrings(keys []string) {
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
}
