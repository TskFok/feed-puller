import { describe, expect, it } from 'vitest';
import {
  assertGlassContrastPasses,
  blendRgb,
  checkGlassContrastPairs,
  contrastRatio,
  parseHexColor,
  panelBackground
} from './glassContrast';

describe('glassContrast', () => {
  it('浅色玻璃面板正文与次要字满足 WCAG AA', () => {
    const results = checkGlassContrastPairs();
    expect(results.every((r) => r.pass)).toBe(true);
    expect(results.find((r) => r.name.includes('正文'))?.ratio).toBeGreaterThan(7);
    expect(results.find((r) => r.name.includes('次要'))?.ratio).toBeGreaterThan(4.5);
  });

  it('assertGlassContrastPasses 在不达标时抛出', () => {
    expect(() =>
      assertGlassContrastPasses([
        {
          name: 'fail',
          foreground: '#ffffff',
          background: [255, 255, 255],
          panelAlpha: 1,
          minRatio: 4.5
        }
      ])
    ).toThrow(/Glass contrast check failed/);
  });

  it('blendRgb 与 contrastRatio 计算稳定', () => {
    const bg = panelBackground([240, 253, 250], 0.68);
    expect(blendRgb([255, 255, 255], 0.68, [240, 253, 250])).toEqual(bg);
    expect(contrastRatio(parseHexColor('#134e4a'), bg)).toBeGreaterThan(9);
  });
});
