import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('list text vertical alignment styles', () => {
  it('让共享表格表头与单元格垂直居中', () => {
    expect(styles.includes('th,\ntd {\n  padding: 12px 14px;\n  border-bottom: 1px solid var(--glass-border-soft);\n  text-align: left;\n  vertical-align: middle;')).toBe(true);
  });
});
