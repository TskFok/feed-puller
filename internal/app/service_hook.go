package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"feed-puller/internal/downloader"
	"feed-puller/internal/downloads"
)

// Aria2HookEvent 表示 aria2 钩子上报的事件类型。
type Aria2HookEvent string

const (
	// Aria2HookEventFileComplete 对应 on-download-complete（单文件完成，多文件 BT 会多次触发）。
	Aria2HookEventFileComplete Aria2HookEvent = "file-complete"
	// Aria2HookEventBTComplete 对应 on-bt-download-complete（整个 BT/磁力任务结束）。
	Aria2HookEventBTComplete Aria2HookEvent = "bt-complete"
	// Aria2HookEventComplete 为旧版脚本传入的通用 complete，需结合 tellStatus 判断是否整任务结束。
	Aria2HookEventComplete Aria2HookEvent = "complete"
	Aria2HookEventError    Aria2HookEvent = "error"
	Aria2HookEventStop     Aria2HookEvent = "stop"
)

// NormalizeAria2HookEvent 将外部传入的事件字符串规范化为支持的事件类型。
// 兼容 aria2 官方钩子名（on-download-complete / on-bt-download-complete / on-download-error / on-download-stop）。
func NormalizeAria2HookEvent(raw string) (Aria2HookEvent, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "file-complete", "on-download-complete":
		return Aria2HookEventFileComplete, nil
	case "bt-complete", "on-bt-download-complete":
		return Aria2HookEventBTComplete, nil
	case "complete":
		return Aria2HookEventComplete, nil
	case "error", "on-download-error":
		return Aria2HookEventError, nil
	case "stop", "on-download-stop":
		return Aria2HookEventStop, nil
	default:
		return "", fmt.Errorf("不支持的 aria2 事件: %q", raw)
	}
}

// ErrAria2HookTaskNotFound 当 gid 在数据库中不存在时返回（例如用户在 aria2 里手动添加的下载）。
// httpapi 层应据此返回 200 (no-op)，避免脚本被反复告警。
var ErrAria2HookTaskNotFound = errors.New("aria2 hook: 未找到匹配的下载任务")

// HandleAria2Hook 处理 aria2 钩子上报的事件。
// 设计要点：
//   - 幂等：已是终态（completed/failed/skipped）的任务直接返回 nil，便于脚本可重试。
//   - 推/拉双通道：与 SyncAria2DownloadStatus 并存，先到先记，事务避免冲突。
//   - 不依赖 aria2 RPC：filePath 来自钩子参数，aria2 即使已清理记录也能正确入库。
func (s *Service) HandleAria2Hook(ctx context.Context, gid string, event Aria2HookEvent, filePath, errMsg string) error {
	gid = strings.TrimSpace(gid)
	if gid == "" {
		return fmt.Errorf("aria2 gid 不能为空")
	}
	task, err := s.findDownloadTaskForAria2Hook(ctx, gid)
	if err != nil {
		if errors.Is(err, ErrAria2HookTaskNotFound) {
			return ErrAria2HookTaskNotFound
		}
		return fmt.Errorf("查询下载任务失败: %w", err)
	}

	switch downloads.DownloadStatus(task.Status) {
	case downloads.StatusCompleted, downloads.StatusFailed, downloads.StatusSkipped:
		s.log.Info("aria2 hook: 任务已是终态，跳过",
			"task_id", task.ID, "gid", gid, "status", task.Status, "event", event)
		return nil
	}

	switch event {
	case Aria2HookEventFileComplete, Aria2HookEventComplete, Aria2HookEventBTComplete:
		if !s.shouldFinalizeAria2Hook(ctx, gid, event, filePath) {
			s.log.Info("aria2 hook: 尚未达到整任务完成，跳过写库",
				"task_id", task.ID, "gid", gid, "event", event, "file_path", filePath)
			return nil
		}
		finalPath := s.resolveAria2HookFilePath(ctx, gid, filePath)
		sub, subErr := s.store.GetSubscription(ctx, task.SubscriptionID)
		itemTitle := ""
		if item, itemErr := s.store.GetItem(ctx, task.ItemID); itemErr == nil {
			itemTitle = item.Title
		}
		if subErr == nil {
			s.maybeRenameDownloadFileAt(ctx, sub, itemTitle, finalPath)
		} else {
			s.log.Warn("aria2 hook: 读取订阅失败，跳过重命名",
				"subscription_id", task.SubscriptionID, "error", subErr)
		}
		if err := s.store.CompleteDownloadTask(ctx, task.ID, task.ItemID); err != nil {
			return fmt.Errorf("记录下载完成失败: %w", err)
		}
		s.log.Info("aria2 hook: 下载已完成",
			"task_id", task.ID, "item_id", task.ItemID, "gid", gid, "file_path", finalPath)
		return nil
	case Aria2HookEventError:
		text := strings.TrimSpace(errMsg)
		if text == "" {
			text = "aria2 下载失败 (hook)"
		}
		if err := s.store.FailDownloadTaskFromAria2(ctx, task.ID, task.ItemID, text); err != nil {
			return fmt.Errorf("记录下载失败状态失败: %w", err)
		}
		s.log.Info("aria2 hook: 下载失败",
			"task_id", task.ID, "item_id", task.ItemID, "gid", gid, "error", text)
		return nil
	case Aria2HookEventStop:
		// 用户/外部主动停止：保持 submitted 不变，仅记日志，
		// 真正的状态变更由后续 complete/error 钩子或轮询兜底处理。
		s.log.Info("aria2 hook: 收到停止事件，不改变任务状态",
			"task_id", task.ID, "item_id", task.ItemID, "gid", gid)
		return nil
	default:
		return fmt.Errorf("未处理的 aria2 事件: %q", event)
	}
}

// shouldFinalizeAria2Hook 判断钩子是否应把任务写入「已完成」。
// 磁力/BT 会先完成 [METADATA] 占位文件；单文件完成事件须结合 tellStatus，避免元数据阶段提前结单。
func (s *Service) shouldFinalizeAria2Hook(ctx context.Context, gid string, event Aria2HookEvent, filePath string) bool {
	if downloader.IsMetadataDownloadPath(filePath) {
		return false
	}
	switch event {
	case Aria2HookEventFileComplete, Aria2HookEventComplete, Aria2HookEventBTComplete:
		return s.isAria2DownloadFullyComplete(ctx, gid, filePath)
	default:
		return false
	}
}

func (s *Service) isAria2DownloadFullyComplete(ctx context.Context, gid, filePath string) bool {
	_, status, err := s.aria2.TellStatusEffective(ctx, gid)
	if err != nil {
		if downloader.IsGIDNotFound(err) {
			path := strings.TrimSpace(filePath)
			return path != "" && !downloader.IsMetadataDownloadPath(path)
		}
		s.log.Warn("aria2 hook: 查询 tellStatus 失败", "gid", gid, "error", err)
		return false
	}
	return downloader.IsAria2DownloadReady(status)
}

// resolveAria2HookFilePath 在整任务完成时尽量返回真实媒体路径（跳过 [METADATA]）。
func (s *Service) resolveAria2HookFilePath(ctx context.Context, gid, hookPath string) string {
	hookPath = strings.TrimSpace(hookPath)
	if hookPath != "" && !downloader.IsMetadataDownloadPath(hookPath) {
		return hookPath
	}
	_, status, err := s.aria2.TellStatusEffective(ctx, gid)
	if err != nil {
		return hookPath
	}
	if path, err := downloader.Aria2DownloadPath(status); err == nil {
		return path
	}
	return hookPath
}
