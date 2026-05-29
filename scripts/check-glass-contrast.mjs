#!/usr/bin/env node
/**
 * CI 对比度门禁：与 web/src/glassContrast.ts 保持同一套公式与令牌。
 * 修改 styles.css 浅色令牌时请同步 LIGHT_GLASS_CONTRAST_PAIRS。
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const stylesPath = join(root, 'web/src/styles.css');
const styles = readFileSync(stylesPath, 'utf8');

const EXPECTED = {
  panel: 'rgba(255, 255, 255, 0.68)',
  text: '#134e4a',
  muted: '#4a6964',
  deep: '#f0fdfa',
  fallbackPanel: 'rgba(255, 255, 255, 0.88)'
};

function extractLightBlock(css) {
  const match = css.match(/html\[data-theme='light'\]\s*\{([^}]+)\}/);
  return match?.[1] ?? '';
}

function assertStylesSync() {
  const block = extractLightBlock(styles);
  const checks = [
    ['--glass-panel', EXPECTED.panel],
    ['--glass-text', EXPECTED.text],
    ['--glass-text-muted', EXPECTED.muted],
    ['--glass-deep', EXPECTED.deep]
  ];
  for (const [token, value] of checks) {
    if (!block.includes(`${token}: ${value}`)) {
      console.error(`styles.css 浅色 ${token} 与 glassContrast 令牌不一致，期望 ${value}`);
      process.exit(1);
    }
  }
  if (!styles.includes(EXPECTED.fallbackPanel)) {
    console.error('styles.css 缺少无 backdrop-filter 降级面板不透明度 0.88');
    process.exit(1);
  }
}

function parseHex(hex) {
  const n = hex.replace('#', '');
  return [parseInt(n.slice(0, 2), 16), parseInt(n.slice(2, 4), 16), parseInt(n.slice(4, 6), 16)];
}

function lum([r, g, b]) {
  const [rs, gs, bs] = [r, g, b].map((c) => {
    const x = c / 255;
    return x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

function contrast(fg, bg) {
  const L1 = lum(fg);
  const L2 = lum(bg);
  return (Math.max(L1, L2) + 0.05) / (Math.min(L1, L2) + 0.05);
}

function blend(fg, alpha, bg) {
  return fg.map((c, i) => Math.round(alpha * c + (1 - alpha) * bg[i]));
}

const body = parseHex(EXPECTED.deep);
const pairs = [
  { name: '正文 × 玻璃面板', fg: EXPECTED.text, alpha: 0.68, min: 4.5 },
  { name: '次要 × 玻璃面板', fg: EXPECTED.muted, alpha: 0.68, min: 4.5 },
  { name: '正文 × 降级面板', fg: EXPECTED.text, alpha: 0.88, min: 4.5 }
];

assertStylesSync();

let failed = false;
for (const pair of pairs) {
  const panel = blend([255, 255, 255], pair.alpha, body);
  const ratio = contrast(parseHex(pair.fg), panel);
  const pass = ratio >= pair.min;
  const mark = pass ? '✓' : '✗';
  console.log(`${mark} ${pair.name}: ${ratio.toFixed(2)}:1 (≥${pair.min}:1)`);
  if (!pass) failed = true;
}

if (failed) {
  process.exit(1);
}

console.log('Glass contrast check passed.');
