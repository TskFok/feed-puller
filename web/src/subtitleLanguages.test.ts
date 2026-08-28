import { describe, expect, it } from 'vitest';
import {
  filterSubtitleItems,
  groupSubtitleItems,
  joinSubtitleLanguages,
  languageLabel,
  resultLanguageCodes,
  toggleSubtitleLanguage
} from './subtitleLanguages';

describe('subtitleLanguages', () => {
  it('按选项顺序拼接已选语言', () => {
    expect(joinSubtitleLanguages(['en', 'zh-CN', 'zh-TW'])).toBe('zh-CN,zh-TW,en');
  });

  it('忽略未提供的语言代码', () => {
    expect(joinSubtitleLanguages(['zh-CN', 'xx'])).toBe('zh-CN');
  });

  it('未选择时返回空字符串', () => {
    expect(joinSubtitleLanguages([])).toBe('');
  });

  it('切换语言时保持选项顺序', () => {
    expect(toggleSubtitleLanguage(['zh-CN'], 'en')).toEqual(['zh-CN', 'en']);
    expect(toggleSubtitleLanguage(['zh-CN', 'en'], 'zh-CN')).toEqual(['en']);
  });

  it('用中文标签显示已知语言，未知代码原样返回', () => {
    expect(languageLabel('zh-cn')).toBe('简体中文');
    expect(languageLabel('pt-BR')).toBe('pt-BR');
  });

  it('按选项顺序列出结果中出现的语言', () => {
    expect(resultLanguageCodes([
      { language: 'en' },
      { language: 'zh-cn' },
      { language: 'en' },
      { language: 'pt-BR' }
    ])).toEqual(['zh-CN', 'en', 'pt-BR']);
  });

  it('可按语言筛选结果且大小写不敏感', () => {
    const items = [
      { id: 1, language: 'zh-CN' },
      { id: 2, language: 'en' },
      { id: 3, language: 'zh-cn' }
    ];
    expect(filterSubtitleItems(items, '')).toEqual(items);
    expect(filterSubtitleItems(items, 'zh-CN').map((item) => item.id)).toEqual([1, 3]);
  });

  it('按语言分组并保持选项顺序', () => {
    const groups = groupSubtitleItems([
      { id: 1, language: 'en' },
      { id: 2, language: 'zh-CN' },
      { id: 3, language: 'en' }
    ]);
    expect(groups.map((group) => [group.language, group.items.map((item) => item.id)])).toEqual([
      ['zh-CN', [2]],
      ['en', [1, 3]]
    ]);
  });

  it('同一语言内按下载次数倒序', () => {
    const groups = groupSubtitleItems([
      { id: 1, language: 'en', download_count: 5 },
      { id: 2, language: 'zh-CN', download_count: 3 },
      { id: 3, language: 'en', download_count: 20 },
      { id: 4, language: 'zh-CN', download_count: 9 }
    ]);
    expect(groups.map((group) => [group.language, group.items.map((item) => item.id)])).toEqual([
      ['zh-CN', [4, 2]],
      ['en', [3, 1]]
    ]);
  });
});
