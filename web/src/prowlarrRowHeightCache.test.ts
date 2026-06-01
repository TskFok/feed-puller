import { beforeEach, describe, expect, it } from 'vitest';
import {
  PROWLARR_ROW_HEIGHT_CACHE_KEY,
  getCachedProwlarrRowEstimate,
  readProwlarrRowHeightCache,
  recordProwlarrRowHeight,
  rowHeightCacheKey,
  titleLengthBucket,
  writeProwlarrRowHeightCache
} from './prowlarrRowHeightCache';
import { PROWLARR_ROW_TOTAL_ESTIMATE_PX } from './prowlarrLayoutConstants';

describe('prowlarrRowHeightCache', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('titleLengthBucket 按 40 字符分桶', () => {
    expect(titleLengthBucket(0)).toBe(40);
    expect(titleLengthBucket(41)).toBe(80);
    expect(titleLengthBucket(120)).toBe(120);
  });

  it('未命中缓存时返回默认估算高度', () => {
    expect(getCachedProwlarrRowEstimate(3, [12, 80])).toBe(PROWLARR_ROW_TOTAL_ESTIMATE_PX);
  });

  it('记录后按列数与标题长度分桶读取，并保留较大实测值', () => {
    recordProwlarrRowHeight(2, [30, 50], 300);
    expect(getCachedProwlarrRowEstimate(2, [30, 50])).toBe(300);
    recordProwlarrRowHeight(2, [30, 50], 280);
    expect(getCachedProwlarrRowEstimate(2, [30, 50])).toBe(300);
    recordProwlarrRowHeight(2, [30, 50], 320);
    expect(getCachedProwlarrRowEstimate(2, [30, 50])).toBe(320);
  });

  it('持久化到 sessionStorage', () => {
    const key = rowHeightCacheKey(1, [100]);
    writeProwlarrRowHeightCache({ [key]: 340 });
    expect(readProwlarrRowHeightCache()[key]).toBe(340);
    expect(sessionStorage.getItem(PROWLARR_ROW_HEIGHT_CACHE_KEY)).toContain(String(key));
  });
});
