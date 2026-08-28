// 项目未直接依赖 Node 类型；Vitest 运行时仍提供该内置模块。
// @ts-expect-error 缺少 @types/node 是测试编译配置的既有限制。
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

declare const process: { cwd(): string };

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('subtitles results column styles', () => {
  it('语言、次数、评分和操作列收窄，发行名与文件名分享剩余宽度', () => {
    expect(styles).toContain('.subtitles-results .subtitles-col-language {\n  width: 5.5rem;');
    expect(styles).toContain('.subtitles-results .subtitles-col-count {\n  width: 6.5rem;');
    expect(styles).toContain('.subtitles-results .subtitles-col-rating {\n  width: 4.5rem;');
    expect(styles).toContain('.subtitles-results .subtitles-col-actions {\n  width: 8.5rem;');
    expect(styles).not.toContain('.subtitles-results .subtitles-col-release {\n  width:');
    expect(styles).not.toContain('.subtitles-results .subtitles-col-file {\n  width:');
  });

  it('语言多选项保持可点按的最小高度', () => {
    expect(styles).toContain('.language-option {\n  display: flex;\n  align-items: center;\n  gap: 8px;\n  min-height: 44px;');
  });

  it('结果语言筛选项保持可点按的最小高度', () => {
    expect(styles).toContain('.subtitles-language-filter-btn {\n  min-height: 44px;');
  });
});
