export const LANGUAGE_OPTIONS = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁体中文' },
  { value: 'en', label: '英语' },
  { value: 'ja', label: '日语' },
  { value: 'ko', label: '韩语' }
] as const;

export function joinSubtitleLanguages(selected: readonly string[]): string {
  const chosen = new Set(selected);
  return LANGUAGE_OPTIONS.map((option) => option.value).filter((value) => chosen.has(value)).join(',');
}

export function toggleSubtitleLanguage(selected: readonly string[], value: string): string[] {
  const chosen = new Set(selected);
  if (chosen.has(value)) {
    chosen.delete(value);
  } else {
    chosen.add(value);
  }
  return LANGUAGE_OPTIONS.map((option) => option.value).filter((code) => chosen.has(code));
}

export function normalizeLanguageCode(code: string): string {
  const raw = code.trim();
  if (!raw) {
    return '';
  }
  const match = LANGUAGE_OPTIONS.find((option) => option.value.toLowerCase() === raw.toLowerCase());
  return match?.value ?? raw;
}

export function languageLabel(code: string): string {
  const normalized = normalizeLanguageCode(code);
  return LANGUAGE_OPTIONS.find((option) => option.value === normalized)?.label ?? (normalized || code);
}

export function resultLanguageCodes(items: readonly { language: string }[]): string[] {
  const present = new Set(items.map((item) => normalizeLanguageCode(item.language)).filter(Boolean));
  const known = LANGUAGE_OPTIONS.map((option) => option.value).filter((code) => present.has(code));
  const unknown = [...present]
    .filter((code) => !LANGUAGE_OPTIONS.some((option) => option.value === code))
    .sort((left, right) => left.localeCompare(right));
  return [...known, ...unknown];
}

export function filterSubtitleItems<T extends { language: string }>(items: readonly T[], language: string): T[] {
  if (!language) {
    return [...items];
  }
  const wanted = normalizeLanguageCode(language);
  return items.filter((item) => normalizeLanguageCode(item.language) === wanted);
}

export function groupSubtitleItems<T extends { language: string }>(items: readonly T[]): { language: string; items: T[] }[] {
  const buckets = new Map<string, T[]>();
  for (const item of items) {
    const key = normalizeLanguageCode(item.language);
    const list = buckets.get(key);
    if (list) {
      list.push(item);
    } else {
      buckets.set(key, [item]);
    }
  }
  return resultLanguageCodes(items)
    .map((language) => ({ language, items: buckets.get(language) ?? [] }))
    .filter((group) => group.items.length > 0);
}
