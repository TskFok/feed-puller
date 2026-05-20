#!/usr/bin/env bash
# 串联 P3TERX clean.sh 与 feed-puller 钩子（aria2 每个钩子只能配置一条命令）。
#
# aria2.conf 示例：
#   on-download-complete=/path/to/aria2-on-download-complete.sh
#   on-bt-download-complete=/path/to/feed-puller-hook.sh bt-complete
#
# aria2 传入位置参数：$1=GID  $2=NUM_FILES  $3=FILE_PATH
#
# 环境变量（可选）：
#   ARIA2_CLEAN_SCRIPT   默认 /Users/ushopal/Downloads/clean.sh
#   ARIA2_FEED_HOOK      默认与本脚本同目录下的 aria2-hook.sh
#   FEED_PULLER_URL / ARIA2_HOOK_SECRET  传给 feed-puller-hook.sh
set -euo pipefail

GID="${1:-}"
NUM_FILES="${2:-}"
FILE_PATH="${3:-}"

CLEAN_SCRIPT="${ARIA2_CLEAN_SCRIPT:-/config/script/clean.sh}"
FEED_HOOK="${ARIA2_FEED_HOOK:-/config/script/aria2-hook.sh}"

run_clean() {
  if [[ ! -x "$CLEAN_SCRIPT" ]]; then
    echo "aria2-on-download-complete: clean script not executable: $CLEAN_SCRIPT" >&2
    return 0
  fi
  # P3TERX clean.sh 期望的参数即为 aria2 原始三元组，勿在前面加 event 名。
  "$CLEAN_SCRIPT" "$GID" "$NUM_FILES" "$FILE_PATH" || {
    echo "aria2-on-download-complete: clean.sh failed (gid=$GID)" >&2
    return 0
  }
}

run_feed_puller() {
  if [[ ! -x "$FEED_HOOK" ]]; then
    echo "aria2-on-download-complete: feed-puller hook not executable: $FEED_HOOK" >&2
    return 0
  fi
  # file-complete：单文件完成；整任务是否结单由服务端结合 tellStatus 判断。
  "$FEED_HOOK" file-complete "$GID" "$NUM_FILES" "$FILE_PATH" || {
    echo "aria2-on-download-complete: feed-puller hook failed (gid=$GID)" >&2
    return 0
  }
}

# 先清理冗余文件，再通知 feed-puller（清理失败不阻断上报）。
run_clean
run_feed_puller
exit 0
