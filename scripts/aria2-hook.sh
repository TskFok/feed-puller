#!/usr/bin/env bash
# aria2 → feed-puller 钩子脚本
#
# 用法（aria2.conf）：
#   若已有 P3TERX clean.sh 等 on-download-complete 脚本，请改用串联脚本：
#     on-download-complete=/path/to/aria2-on-download-complete.sh
#   否则可单独配置：
#     on-download-complete=/path/to/aria2-hook.sh file-complete
#   on-bt-download-complete=/path/to/feed-puller-hook.sh bt-complete
#   on-download-error=/etc/aria2/feed-puller-hook.sh error
#   on-download-stop=/etc/aria2/feed-puller-hook.sh stop
#
# aria2 调用约定（位置参数）：
#   $1 = event（由 aria2.conf 中传入：complete / error / stop）
#   $2 = GID
#   $3 = NUM_FILES
#   $4 = FILE_PATH（首个文件的完整本地路径，error/stop 时可能为空）
#
# 通过环境变量提供：
#   FEED_PULLER_URL    例：http://feed-puller:8080   （必填）
#   ARIA2_HOOK_SECRET  与服务端 ARIA2_HOOK_SECRET 一致（必填）
set -euo pipefail

EVENT="${1:-}"
GID="${2:-}"
# $3 NUM_FILES 暂不使用
FILE_PATH="${4:-}"

: "${FEED_PULLER_URL:?need FEED_PULLER_URL}"
: "${ARIA2_HOOK_SECRET:?need ARIA2_HOOK_SECRET}"

if [[ -z "$EVENT" || -z "$GID" ]]; then
  echo "aria2-hook: missing event or gid (event=$EVENT gid=$GID)" >&2
  exit 0
fi

# 简单 JSON 转义：转义反斜杠与双引号，足以覆盖 aria2 传入的路径。
json_escape() {
  local s="${1//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '%s' "$s"
}

payload=$(printf '{"gid":"%s","event":"%s","file_path":"%s"}' \
  "$(json_escape "$GID")" \
  "$(json_escape "$EVENT")" \
  "$(json_escape "$FILE_PATH")")

# 5s 连接超时 + 10s 总超时，钩子失败不阻塞 aria2 继续运行
curl --silent --show-error \
  --connect-timeout 5 --max-time 10 \
  --retry 2 --retry-delay 1 \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ARIA2_HOOK_SECRET}" \
  -d "$payload" \
  "${FEED_PULLER_URL%/}/api/downloads/aria2-hook" \
  >/dev/null || {
    echo "aria2-hook: notify feed-puller failed (event=$EVENT gid=$GID)" >&2
    exit 0
  }
