import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('modal surface styles', () => {
  it('弹窗使用比列表玻璃面板更不透明的专用背景', () => {
    expect(styles.includes('--modal-panel-bg: rgba(255, 255, 255, 0.96)')).toBe(true);
    expect(styles.includes('--modal-panel-bg: rgba(20, 28, 46, 0.98)')).toBe(true);
    expect(styles.includes('background: var(--modal-panel-bg, var(--glass-panel-solid))')).toBe(true);
  });

  it('弹窗遮罩使用专用背景以压低后方列表内容', () => {
    expect(styles.includes('background: var(--modal-overlay-bg, var(--overlay-bg))')).toBe(true);
  });
});
