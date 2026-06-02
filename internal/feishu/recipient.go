package feishu

import "strings"

// Recipient 飞书消息接收者。
type Recipient struct {
	IDType string // open_id | chat_id | user_id
	ID     string
}

// ParseRecipients 解析接收者列表。
// 支持格式（每行一个）：
//   - open_id:ou_xxx / chat_id:oc_xxx / user_id:u_xxx
//   - ou_xxx（默认 open_id）/ oc_xxx（默认 chat_id）
func ParseRecipients(legacyOpenID, raw string) []Recipient {
	seen := make(map[string]struct{})
	var out []Recipient
	add := func(idType, id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		key := idType + ":" + id
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, Recipient{IDType: idType, ID: id})
	}
	if legacy := strings.TrimSpace(legacyOpenID); legacy != "" {
		add("open_id", legacy)
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			idType := strings.ToLower(strings.TrimSpace(line[:idx]))
			id := strings.TrimSpace(line[idx+1:])
			switch idType {
			case "open_id", "chat_id", "user_id":
				add(idType, id)
			default:
				add(guessRecipientType(line), line)
			}
			continue
		}
		add(guessRecipientType(line), line)
	}
	return out
}

func guessRecipientType(id string) string {
	switch {
	case strings.HasPrefix(id, "oc_"):
		return "chat_id"
	case strings.HasPrefix(id, "ou_"):
		return "open_id"
	default:
		return "open_id"
	}
}
