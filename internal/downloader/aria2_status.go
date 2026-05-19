package downloader

import "strings"

// Aria2TaskState 表示 aria2.tellStatus 返回的 status 字段语义。
type Aria2TaskState int

const (
	Aria2TaskActive Aria2TaskState = iota
	Aria2TaskComplete
	Aria2TaskError
	Aria2TaskRemoved
)

// ParseAria2TaskStatus 根据 aria2.tellStatus 结果判断任务是否完成或失败。
func ParseAria2TaskStatus(status map[string]any) (state Aria2TaskState, errMsg string) {
	raw, _ := status["status"].(string)
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "complete":
		return Aria2TaskComplete, ""
	case "error":
		if msg, ok := status["errorMessage"].(string); ok && strings.TrimSpace(msg) != "" {
			return Aria2TaskError, strings.TrimSpace(msg)
		}
		return Aria2TaskError, "aria2 下载失败"
	case "removed":
		return Aria2TaskRemoved, ""
	default:
		return Aria2TaskActive, ""
	}
}
