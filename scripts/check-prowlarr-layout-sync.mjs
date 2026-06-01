#!/usr/bin/env node
/**
 * 校验 styles.css 中 Prowlarr 布局 CSS 回退值与 prowlarrLayoutConstants.ts 一致。
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const styles = readFileSync(join(root, 'web/src/styles.css'), 'utf8');
const constants = readFileSync(join(root, 'web/src/prowlarrLayoutConstants.ts'), 'utf8');

const EXPECTED = {
  rowGap: 14,
  cardIntrinsic: 301,
  rowEstimate: 301,
  virtualThreshold: 30
};

function readNumber(name) {
  const match = constants.match(new RegExp(`export const ${name} = (\\d+)`));
  if (!match) {
    console.error(`prowlarrLayoutConstants.ts 缺少 ${name}`);
    process.exit(1);
  }
  return Number(match[1]);
}

function assertCssFallback(varName, px) {
  const pattern = new RegExp(`${varName}[^;]*,\\s*${px}px`);
  if (!pattern.test(styles)) {
    console.error(`styles.css 中 ${varName} 回退值应为 ${px}px`);
    process.exit(1);
  }
}

const rowGap = readNumber('PROWLARR_ROW_GAP_PX');
const cardIntrinsic = readNumber('PROWLARR_CARD_INTRINSIC_HEIGHT_PX');
const rowEstimate = readNumber('PROWLARR_ROW_ESTIMATE_PX');

if (rowGap !== EXPECTED.rowGap || cardIntrinsic !== EXPECTED.cardIntrinsic || rowEstimate !== EXPECTED.rowEstimate) {
  console.error('prowlarrLayoutConstants.ts 数值与 check 脚本期望值不一致，请同步 EXPECTED');
  process.exit(1);
}

const thresholdMatch = readFileSync(join(root, 'web/src/glassConstants.ts'), 'utf8').match(
  /PROWLARR_VIRTUALIZE_THRESHOLD = (\d+)/
);
if (!thresholdMatch || Number(thresholdMatch[1]) !== EXPECTED.virtualThreshold) {
  console.error(`glassConstants.ts PROWLARR_VIRTUALIZE_THRESHOLD 应为 ${EXPECTED.virtualThreshold}`);
  process.exit(1);
}

assertCssFallback('--prowlarr-row-gap', rowGap);
assertCssFallback('--prowlarr-card-intrinsic-height', cardIntrinsic);

console.log('Prowlarr 布局常量与 styles.css 回退值一致');
