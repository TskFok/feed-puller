/**
 * 浅色玻璃主题对比度校验（与 styles.css 中 html[data-theme='light'] 令牌保持同步）。
 * CI：`npm run check:contrast`
 */

export type Rgb = [number, number, number];

export type GlassContrastPair = {
  name: string;
  foreground: string;
  background: Rgb;
  panelAlpha: number;
  minRatio: number;
};

/** 与 styles.css 同步的浅色主题抽样 */
export const LIGHT_GLASS_CONTRAST_PAIRS: readonly GlassContrastPair[] = [
  {
    name: '正文 × 玻璃面板',
    foreground: '#134e4a',
    background: [240, 253, 250],
    panelAlpha: 0.68,
    minRatio: 4.5
  },
  {
    name: '次要文字 × 玻璃面板',
    foreground: '#4a6964',
    background: [240, 253, 250],
    panelAlpha: 0.68,
    minRatio: 4.5
  },
  {
    name: '正文 × 无模糊降级面板',
    foreground: '#134e4a',
    background: [240, 253, 250],
    panelAlpha: 0.88,
    minRatio: 4.5
  }
];

export function parseHexColor(hex: string): Rgb {
  const normalized = hex.replace('#', '').trim();
  if (normalized.length !== 6) {
    throw new Error(`Invalid hex color: ${hex}`);
  }
  return [
    Number.parseInt(normalized.slice(0, 2), 16),
    Number.parseInt(normalized.slice(2, 4), 16),
    Number.parseInt(normalized.slice(4, 6), 16)
  ];
}

export function blendRgb(foreground: Rgb, alpha: number, background: Rgb): Rgb {
  const r = Math.round(alpha * foreground[0] + (1 - alpha) * background[0]);
  const g = Math.round(alpha * foreground[1] + (1 - alpha) * background[1]);
  const b = Math.round(alpha * foreground[2] + (1 - alpha) * background[2]);
  return [r, g, b];
}

export function relativeLuminance([r, g, b]: Rgb): number {
  const [rs, gs, bs] = [r, g, b].map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

export function contrastRatio(foreground: Rgb, background: Rgb): number {
  const fg = relativeLuminance(foreground);
  const bg = relativeLuminance(background);
  const lighter = Math.max(fg, bg);
  const darker = Math.min(fg, bg);
  return (lighter + 0.05) / (darker + 0.05);
}

export function panelBackground(background: Rgb, panelAlpha: number): Rgb {
  return blendRgb([255, 255, 255], panelAlpha, background);
}

export type ContrastCheckResult = {
  name: string;
  ratio: number;
  minRatio: number;
  pass: boolean;
};

export function checkGlassContrastPairs(pairs: readonly GlassContrastPair[] = LIGHT_GLASS_CONTRAST_PAIRS): ContrastCheckResult[] {
  return pairs.map((pair) => {
    const fg = parseHexColor(pair.foreground);
    const bg = panelBackground(pair.background, pair.panelAlpha);
    const ratio = contrastRatio(fg, bg);
    return {
      name: pair.name,
      ratio,
      minRatio: pair.minRatio,
      pass: ratio >= pair.minRatio
    };
  });
}

export function assertGlassContrastPasses(pairs: readonly GlassContrastPair[] = LIGHT_GLASS_CONTRAST_PAIRS): void {
  const results = checkGlassContrastPairs(pairs);
  const failures = results.filter((result) => !result.pass);
  if (failures.length > 0) {
    const detail = failures
      .map((f) => `${f.name}: ${f.ratio.toFixed(2)}:1 (需要 ≥${f.minRatio}:1)`)
      .join('; ');
    throw new Error(`Glass contrast check failed: ${detail}`);
  }
}
