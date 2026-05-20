package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"feed-puller/internal/app"
)

// aria2HookRequest 与 scripts/aria2-hook.sh 约定的 JSON 结构。
// event 必填，gid 必填；file_path / error 可选。
type aria2HookRequest struct {
	GID      string `json:"gid"`
	Event    string `json:"event"`
	FilePath string `json:"file_path"`
	Error    string `json:"error"`
}

// handleAria2Hook 接收 aria2 钩子上报，将 gid 对应的下载任务直接置为终态。
// 该端点不走 session 鉴权（aria2 是独立进程），改用 ARIA2_HOOK_SECRET 做 Bearer 校验。
func (s *Server) handleAria2Hook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.checkAria2HookSecret(r) {
		writeError(w, http.StatusUnauthorized, "aria2 hook 鉴权失败")
		return
	}
	var input aria2HookRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	event, err := app.NormalizeAria2HookEvent(input.Event)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.GID) == "" {
		writeError(w, http.StatusBadRequest, "gid 不能为空")
		return
	}
	if err := s.service.HandleAria2Hook(r.Context(), input.GID, event, input.FilePath, input.Error); err != nil {
		if errors.Is(err, app.ErrAria2HookTaskNotFound) {
			// 未匹配到任务（用户在 aria2 里手动添加的下载等），返回 200 no-op 避免脚本反复告警。
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "matched": false})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "matched": true})
}

// checkAria2HookSecret 使用常量时间比较校验 Bearer / X-Hook-Secret，避免时序侧信道。
// 当 ARIA2_HOOK_SECRET 未配置时一律拒绝，避免无意中暴露未鉴权端点。
func (s *Server) checkAria2HookSecret(r *http.Request) bool {
	expected := strings.TrimSpace(s.cfg.Aria2HookSecret)
	if expected == "" {
		return false
	}
	candidate := strings.TrimSpace(r.Header.Get("X-Hook-Secret"))
	if candidate == "" {
		raw := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			candidate = strings.TrimSpace(raw[len("bearer "):])
		}
	}
	if candidate == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(expected)) == 1
}
