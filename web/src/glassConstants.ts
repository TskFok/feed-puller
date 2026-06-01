/** 超过该条数时为离屏玻璃表面关闭 backdrop-filter */
export const GLASS_OFFSCREEN_MIN_ITEMS = 12;

/** Prowlarr 非虚拟网格启用离屏玻璃优化的最低条数 */
export const PROWLARR_OFFSCREEN_MIN_ITEMS = 8;

/** @deprecated 请使用 resolveProwlarrVirtualizeThreshold 或 useProwlarrVirtualizeThreshold */
export { PROWLARR_VIRTUALIZE_BASE_THRESHOLD as PROWLARR_VIRTUALIZE_THRESHOLD } from './prowlarrVirtualizeThreshold';

export const GLASS_OFFSCREEN_CLASS = 'glass-surface--offscreen';
