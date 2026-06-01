import { describe, expect, it } from 'vitest';
import {
  PROWLARR_CARD_INTRINSIC_HEIGHT_PX,
  PROWLARR_LAYOUT_CSS_VARS,
  PROWLARR_ROW_ESTIMATE_PX,
  PROWLARR_ROW_GAP_PX,
  PROWLARR_ROW_TOTAL_ESTIMATE_PX,
  applyProwlarrLayoutCssVars
} from './prowlarrLayoutConstants';

describe('prowlarrLayoutConstants', () => {
  it('行高常量关系一致', () => {
    expect(PROWLARR_ROW_TOTAL_ESTIMATE_PX).toBe(PROWLARR_ROW_ESTIMATE_PX + PROWLARR_ROW_GAP_PX);
    expect(PROWLARR_CARD_INTRINSIC_HEIGHT_PX).toBeGreaterThanOrEqual(PROWLARR_ROW_ESTIMATE_PX);
  });

  it('applyProwlarrLayoutCssVars 写入 CSS 变量', () => {
    const root = document.documentElement;
    applyProwlarrLayoutCssVars(root);
    expect(root.style.getPropertyValue('--prowlarr-row-gap')).toBe(PROWLARR_LAYOUT_CSS_VARS['--prowlarr-row-gap']);
    expect(root.style.getPropertyValue('--prowlarr-card-intrinsic-height')).toBe(
      PROWLARR_LAYOUT_CSS_VARS['--prowlarr-card-intrinsic-height']
    );
  });
});
