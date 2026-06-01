/** Prowlarr 结果卡片预估内容高度（不含行间距），与实测卡片高度对齐 */
export const PROWLARR_ROW_ESTIMATE_PX = 301;

/** 虚拟行 / 网格行间距，与 styles.css `--prowlarr-row-gap` 同步 */
export const PROWLARR_ROW_GAP_PX = 14;

/** 虚拟行默认估算总高度（内容 + 行间距） */
export const PROWLARR_ROW_TOTAL_ESTIMATE_PX = PROWLARR_ROW_ESTIMATE_PX + PROWLARR_ROW_GAP_PX;

/** `content-visibility` 占位高度，与 styles.css `--prowlarr-card-intrinsic-height` 同步 */
export const PROWLARR_CARD_INTRINSIC_HEIGHT_PX = 301;

export const PROWLARR_LAYOUT_CSS_VARS = {
  '--prowlarr-row-gap': `${PROWLARR_ROW_GAP_PX}px`,
  '--prowlarr-card-intrinsic-height': `${PROWLARR_CARD_INTRINSIC_HEIGHT_PX}px`
} as const;

/** 将布局变量写入根节点，保证 TS 常量与 CSS 一致 */
export function applyProwlarrLayoutCssVars(root: HTMLElement = document.documentElement): void {
  for (const [name, value] of Object.entries(PROWLARR_LAYOUT_CSS_VARS)) {
    root.style.setProperty(name, value);
  }
}
