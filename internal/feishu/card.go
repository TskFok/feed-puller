package feishu

import "strings"

// InteractiveCard 飞书 interactive 卡片。
type InteractiveCard struct {
	Title    string
	Template string // blue | green | red | orange | wathet
	Lines    []string
}

// BuildTextBody 构建纯文本正文（标题与正文分离发送时使用）。
func BuildTextBody(lines []string) string {
	return strings.Join(lines, "\n")
}

func cardPayload(card InteractiveCard) map[string]any {
	template := strings.TrimSpace(card.Template)
	if template == "" {
		template = "blue"
	}
	title := strings.TrimSpace(card.Title)
	if title == "" {
		title = "Feed Puller"
	}
	body := strings.TrimSpace(BuildTextBody(card.Lines))
	elements := make([]any, 0, 2)
	if body != "" {
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": body,
			},
		})
	}
	if len(elements) == 0 {
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "plain_text",
				"content": title,
			},
		})
	}
	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": title,
			},
			"template": template,
		},
		"elements": elements,
	}
}
