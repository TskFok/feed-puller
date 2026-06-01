export type ProwlarrVisibleRange = {
  firstItemIndex: number;
  lastItemIndex: number;
  total: number;
};

export function mergeFurthestSeenIndex(current: number, range: ProwlarrVisibleRange): number {
  return Math.max(current, range.lastItemIndex);
}

export function browsedCount(furthestSeenIndex: number, total: number): number {
  if (total <= 0 || furthestSeenIndex < 0) {
    return 0;
  }
  return Math.min(furthestSeenIndex + 1, total);
}

export function hasBrowsedAllResults(furthestSeenIndex: number, total: number): boolean {
  return total > 0 && furthestSeenIndex >= total - 1;
}

export function formatProwlarrBrowseProgress(furthestSeenIndex: number, total: number): string {
  return `已浏览 ${browsedCount(furthestSeenIndex, total)} / 共 ${total} 条`;
}
