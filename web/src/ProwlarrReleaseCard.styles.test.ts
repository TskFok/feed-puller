// 项目未直接依赖 Node 类型；Vitest 运行时仍提供该内置模块。
// @ts-expect-error 缺少 @types/node 是测试编译配置的既有限制。
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

declare const process: { cwd(): string };

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('ProwlarrReleaseCard styles', () => {
  it('为标题、复选框和状态建立可收缩的三列网格', () => {
    expect(styles).toContain('grid-template-columns: 18px minmax(0, 1fr) auto;');
    expect(styles).toContain('margin-top: 0;');
  });

  it('让操作栏和下载按钮保持在卡片宽度内', () => {
    expect(styles).toContain('.prowlarr-release-actions {\n  display: flex;\n  min-width: 0;');
    expect(styles).toContain('.prowlarr-release-download {\n  flex: 0 0 auto;');
  });
});
