package downloader

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Aria2Progress 从 aria2.tellStatus 结果解析出的进度信息。
type Aria2Progress struct {
	Status          string   `json:"aria2_status"`
	CompletedLength int64    `json:"completed_length"`
	TotalLength     int64    `json:"total_length"`
	DownloadSpeed   int64    `json:"download_speed"`
	ProgressPercent *float64 `json:"progress_percent,omitempty"`
}

// ParseAria2Progress 解析 tellStatus 返回的全局进度字段。
// 磁力/BT 在元数据阶段顶层 totalLength 可能为 0，此时从 files 中汇总已选实体文件的长度。
func ParseAria2Progress(status map[string]any) Aria2Progress {
	raw, _ := status["status"].(string)
	p := Aria2Progress{
		Status:          strings.TrimSpace(raw),
		CompletedLength: aria2Numeric(status["completedLength"]),
		TotalLength:     aria2Numeric(status["totalLength"]),
		DownloadSpeed:   aria2Numeric(status["downloadSpeed"]),
	}
	if p.TotalLength <= 0 {
		fileCompleted, fileTotal := aria2AggregateFileProgress(status)
		if fileTotal > 0 {
			p.TotalLength = fileTotal
			if p.CompletedLength <= 0 {
				p.CompletedLength = fileCompleted
			}
		}
	}
	applyAria2ProgressPercent(&p)
	return p
}

func applyAria2ProgressPercent(p *Aria2Progress) {
	if p.TotalLength <= 0 {
		p.ProgressPercent = nil
		return
	}
	percent := float64(p.CompletedLength) / float64(p.TotalLength) * 100
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	p.ProgressPercent = &percent
}

// aria2AggregateFileProgress 汇总非 METADATA、已选中文件的 completedLength / length。
func aria2AggregateFileProgress(status map[string]any) (completed, total int64) {
	files, ok := status["files"].([]any)
	if !ok {
		return 0, 0
	}
	for _, raw := range files {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		path, _ := entry["path"].(string)
		if IsMetadataDownloadPath(strings.TrimSpace(path)) {
			continue
		}
		if !aria2FileEntrySelected(entry) {
			continue
		}
		total += aria2Numeric(entry["length"])
		completed += aria2Numeric(entry["completedLength"])
	}
	return completed, total
}

func aria2Numeric(v any) int64 {
	switch x := v.(type) {
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	default:
		return 0
	}
}
