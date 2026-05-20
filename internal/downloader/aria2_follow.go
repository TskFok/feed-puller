package downloader

import (
	"context"
	"strings"
)

const maxAria2GIDFollowHops = 8

// FirstFollowedByGID 返回 tellStatus 中 followedBy 的首个 GID。
// 磁力元数据下载完成后，aria2 会通过 followedBy 指向实体 BT 下载的新 GID。
func FirstFollowedByGID(status map[string]any) string {
	raw, ok := status["followedBy"]
	if !ok || raw == nil {
		return ""
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return ""
	}
	gid, _ := list[0].(string)
	return strings.TrimSpace(gid)
}

// FollowingGID 返回 tellStatus 中的 following（当前下载所跟随的前序 GID）。
func FollowingGID(status map[string]any) string {
	raw, ok := status["following"]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

// TellStatusEffective 沿 followedBy 链解析到当前应跟踪的 GID，并返回其 tellStatus。
func (c *Aria2Client) TellStatusEffective(ctx context.Context, startGID string) (effectiveGID string, status map[string]any, err error) {
	startGID = strings.TrimSpace(startGID)
	if startGID == "" {
		return "", nil, ErrEmptyGID
	}
	gid := startGID
	var lastStatus map[string]any
	for hop := 0; hop < maxAria2GIDFollowHops; hop++ {
		lastStatus, err = c.TellStatus(ctx, gid)
		if err != nil {
			return gid, lastStatus, err
		}
		next := FirstFollowedByGID(lastStatus)
		if next == "" || next == gid {
			return gid, lastStatus, nil
		}
		gid = next
	}
	return gid, lastStatus, nil
}
