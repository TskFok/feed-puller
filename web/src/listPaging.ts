export const DEFAULT_PAGE_SIZE = 30;

export const PAGE_SIZE_OPTIONS = [10, 30, 50, 100] as const;

export const PAGE_SIZE_STORAGE_KEY = 'feed-puller.page-size';

export type PageSizeOption = (typeof PAGE_SIZE_OPTIONS)[number];

export function isPageSizeOption(value: number): value is PageSizeOption {
  return (PAGE_SIZE_OPTIONS as readonly number[]).includes(value);
}

export function loadStoredPageSize(): PageSizeOption {
  try {
    const raw = localStorage.getItem(PAGE_SIZE_STORAGE_KEY);
    if (raw == null) {
      return DEFAULT_PAGE_SIZE;
    }
    const parsed = Number(raw);
    return isPageSizeOption(parsed) ? parsed : DEFAULT_PAGE_SIZE;
  } catch {
    return DEFAULT_PAGE_SIZE;
  }
}

export function saveStoredPageSize(size: PageSizeOption): void {
  try {
    localStorage.setItem(PAGE_SIZE_STORAGE_KEY, String(size));
  } catch {
    // 隐私模式或配额满时忽略
  }
}

export function totalPages(itemCount: number, pageSize: number): number {
  if (itemCount <= 0) {
    return 1;
  }
  return Math.ceil(itemCount / pageSize);
}

export function clampPage(page: number, itemCount: number, pageSize: number): number {
  return Math.min(Math.max(1, page), totalPages(itemCount, pageSize));
}

export function paginateSlice<T>(items: T[], page: number, pageSize: number): T[] {
  if (items.length === 0) {
    return [];
  }
  const safePage = clampPage(page, items.length, pageSize);
  const start = (safePage - 1) * pageSize;
  return items.slice(start, start + pageSize);
}

export function pageRange(itemCount: number, page: number, pageSize: number): { start: number; end: number } {
  if (itemCount <= 0) {
    return { start: 0, end: 0 };
  }
  const safePage = clampPage(page, itemCount, pageSize);
  const start = (safePage - 1) * pageSize + 1;
  const end = Math.min(safePage * pageSize, itemCount);
  return { start, end };
}

export function pageOffset(page: number, pageSize: number): number {
  return (clampPage(page, Number.MAX_SAFE_INTEGER, pageSize) - 1) * pageSize;
}
