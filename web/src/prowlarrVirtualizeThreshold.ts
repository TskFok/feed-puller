import { PROWLARR_ROW_TOTAL_ESTIMATE_PX } from './prowlarrLayoutConstants';

/** 大屏默认虚拟化阈值 */
export const PROWLARR_VIRTUALIZE_BASE_THRESHOLD = 30;

/** 小屏/workspace 较矮时的最低虚拟化阈值 */
export const PROWLARR_VIRTUALIZE_MIN_THRESHOLD = 12;

/** 搜索表单、工具栏等占用 workspace 高度的估算值 */
export const PROWLARR_RESULTS_CHROME_PX = 420;

/** 根据 workspace 高度与列数动态计算虚拟化阈值 */
export function resolveProwlarrVirtualizeThreshold(workspaceHeight: number, columnCount: number): number {
  if (!Number.isFinite(workspaceHeight) || workspaceHeight <= 0) {
    return PROWLARR_VIRTUALIZE_BASE_THRESHOLD;
  }

  const columns = Math.max(columnCount, 1);
  const availableHeight = Math.max(workspaceHeight - PROWLARR_RESULTS_CHROME_PX, PROWLARR_ROW_TOTAL_ESTIMATE_PX);
  const visibleRows = Math.max(1, Math.floor(availableHeight / PROWLARR_ROW_TOTAL_ESTIMATE_PX));
  const visibleItems = visibleRows * columns;
  const adaptiveThreshold = Math.ceil(visibleItems * 1.5);

  return Math.max(
    PROWLARR_VIRTUALIZE_MIN_THRESHOLD,
    Math.min(PROWLARR_VIRTUALIZE_BASE_THRESHOLD, adaptiveThreshold)
  );
}
