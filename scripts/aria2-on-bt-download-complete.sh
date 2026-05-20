#!/usr/bin/env bash
# BT/磁力整任务完成时通知 feed-puller（与 on-download-complete 的 file-complete 不同）。
#
# aria2.conf：
#   on-bt-download-complete=/path/to/aria2-on-bt-download-complete.sh
#
# aria2 传入：$1=GID  $2=NUM_FILES  $3=FILE_PATH
set -euo pipefail

GID="${1:-}"
NUM_FILES="${2:-}"
FILE_PATH="${3:-}"

FEED_HOOK="${ARIA2_FEED_HOOK:-/config/script/aria2-hook.sh}"

if [[ ! -x "$FEED_HOOK" ]]; then
  echo "aria2-on-bt-download-complete: feed-puller hook not executable: $FEED_HOOK" >&2
  exit 0
fi

"$FEED_HOOK" bt-complete "$GID" "$NUM_FILES" "$FILE_PATH" || {
  echo "aria2-on-bt-download-complete: feed-puller hook failed (gid=$GID)" >&2
  exit 0
}
exit 0
