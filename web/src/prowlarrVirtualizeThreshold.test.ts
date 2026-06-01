import { describe, expect, it } from 'vitest';
import {
  PROWLARR_VIRTUALIZE_BASE_THRESHOLD,
  PROWLARR_VIRTUALIZE_MIN_THRESHOLD,
  resolveProwlarrVirtualizeThreshold
} from './prowlarrVirtualizeThreshold';

describe('resolveProwlarrVirtualizeThreshold', () => {
  it('无效高度时回退到默认阈值', () => {
    expect(resolveProwlarrVirtualizeThreshold(0, 3)).toBe(PROWLARR_VIRTUALIZE_BASE_THRESHOLD);
    expect(resolveProwlarrVirtualizeThreshold(Number.NaN, 3)).toBe(PROWLARR_VIRTUALIZE_BASE_THRESHOLD);
  });

  it('矮 workspace 使用更低阈值', () => {
    expect(resolveProwlarrVirtualizeThreshold(720, 3)).toBe(PROWLARR_VIRTUALIZE_MIN_THRESHOLD);
  });

  it('高 workspace 仍不超过默认阈值', () => {
    expect(resolveProwlarrVirtualizeThreshold(1600, 3)).toBeLessThanOrEqual(PROWLARR_VIRTUALIZE_BASE_THRESHOLD);
    expect(resolveProwlarrVirtualizeThreshold(1600, 3)).toBeGreaterThan(PROWLARR_VIRTUALIZE_MIN_THRESHOLD);
  });
});
