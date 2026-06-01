import { PROWLARR_ROW_TOTAL_ESTIMATE_PX } from './prowlarrLayoutConstants';

export const PROWLARR_ROW_HEIGHT_CACHE_KEY = 'feed-puller:prowlarr-row-heights';

/** 标题长度分桶步长（字符） */
export const PROWLARR_TITLE_LENGTH_BUCKET = 40;

export type ProwlarrRowHeightCache = Record<string, number>;

export function titleLengthBucket(maxTitleLength: number): number {
  if (maxTitleLength <= 0) {
    return PROWLARR_TITLE_LENGTH_BUCKET;
  }
  return Math.ceil(maxTitleLength / PROWLARR_TITLE_LENGTH_BUCKET) * PROWLARR_TITLE_LENGTH_BUCKET;
}

export function rowHeightCacheKey(columnCount: number, titleLengths: readonly number[]): string {
  const maxLen = titleLengths.length > 0 ? Math.max(...titleLengths) : 0;
  return `${columnCount}:${titleLengthBucket(maxLen)}`;
}

export function readProwlarrRowHeightCache(): ProwlarrRowHeightCache {
  if (typeof sessionStorage === 'undefined') {
    return {};
  }
  try {
    const raw = sessionStorage.getItem(PROWLARR_ROW_HEIGHT_CACHE_KEY);
    if (!raw) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return {};
    }
    const cache: ProwlarrRowHeightCache = {};
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
        cache[key] = Math.round(value);
      }
    }
    return cache;
  } catch {
    return {};
  }
}

export function writeProwlarrRowHeightCache(cache: ProwlarrRowHeightCache): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  try {
    sessionStorage.setItem(PROWLARR_ROW_HEIGHT_CACHE_KEY, JSON.stringify(cache));
  } catch {
    /* quota / private mode */
  }
}

export function getCachedProwlarrRowEstimate(
  columnCount: number,
  titleLengths: readonly number[],
  fallback = PROWLARR_ROW_TOTAL_ESTIMATE_PX
): number {
  const cache = readProwlarrRowHeightCache();
  const key = rowHeightCacheKey(columnCount, titleLengths);
  return cache[key] ?? fallback;
}

/** 记录实测行高，取历史最大值以避免虚拟行重叠 */
export function recordProwlarrRowHeight(
  columnCount: number,
  titleLengths: readonly number[],
  measuredHeight: number
): void {
  if (!Number.isFinite(measuredHeight) || measuredHeight <= 0) {
    return;
  }
  const cache = readProwlarrRowHeightCache();
  const key = rowHeightCacheKey(columnCount, titleLengths);
  const rounded = Math.round(measuredHeight);
  cache[key] = cache[key] ? Math.max(cache[key], rounded) : rounded;
  writeProwlarrRowHeightCache(cache);
}
