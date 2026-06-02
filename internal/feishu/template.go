package feishu

import "strings"

// TemplateVars 通知模板可用变量。
type TemplateVars struct {
	Subscription string
	Title        string
	Path         string
	Error        string
	Count        string
	MediaType    string
}

// RenderTemplate 将 {{key}} 替换为变量值。
func RenderTemplate(tmpl string, vars TemplateVars) string {
	replacer := strings.NewReplacer(
		"{{subscription}}", vars.Subscription,
		"{{title}}", vars.Title,
		"{{path}}", vars.Path,
		"{{error}}", vars.Error,
		"{{count}}", vars.Count,
		"{{media_type}}", vars.MediaType,
	)
	return replacer.Replace(tmpl)
}
