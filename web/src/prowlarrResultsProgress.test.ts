import { describe, expect, it } from 'vitest';
import {
  browsedCount,
  formatProwlarrBrowseProgress,
  hasBrowsedAllResults,
  mergeFurthestSeenIndex
} from './prowlarrResultsProgress';

describe('prowlarrResultsProgress', () => {
  it('mergeFurthestSeenIndex 保留最大可见索引', () => {
    expect(mergeFurthestSeenIndex(4, { firstItemIndex: 8, lastItemIndex: 11, total: 144 })).toBe(11);
    expect(mergeFurthestSeenIndex(20, { firstItemIndex: 0, lastItemIndex: 5, total: 144 })).toBe(20);
  });

  it('browsedCount 按已浏览条目数计算', () => {
    expect(browsedCount(-1, 144)).toBe(0);
    expect(browsedCount(8, 144)).toBe(9);
    expect(browsedCount(200, 144)).toBe(144);
  });

  it('hasBrowsedAllResults 在到达最后一条时为 true', () => {
    expect(hasBrowsedAllResults(142, 144)).toBe(false);
    expect(hasBrowsedAllResults(143, 144)).toBe(true);
  });

  it('formatProwlarrBrowseProgress 输出进度文案', () => {
    expect(formatProwlarrBrowseProgress(8, 144)).toBe('已浏览 9 / 共 144 条');
  });
});
