import { beforeEach, describe, expect, it } from 'vitest';
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE_OPTIONS,
  PAGE_SIZE_STORAGE_KEY,
  clampPage,
  loadStoredPageSize,
  pageRange,
  paginateSlice,
  saveStoredPageSize,
  totalPages
} from './listPaging';

describe('listPaging', () => {
  it('默认每页 30 条', () => {
    expect(DEFAULT_PAGE_SIZE).toBe(30);
    expect(PAGE_SIZE_OPTIONS).toContain(30);
  });

  it('paginateSlice 按页切片', () => {
    const items = Array.from({ length: 35 }, (_, i) => i + 1);
    expect(paginateSlice(items, 1, 30)).toEqual(items.slice(0, 30));
    expect(paginateSlice(items, 2, 30)).toEqual([31, 32, 33, 34, 35]);
  });

  it('空列表返回空数组', () => {
    expect(paginateSlice([], 1, 30)).toEqual([]);
    expect(totalPages(0, 30)).toBe(1);
    expect(pageRange(0, 1, 30)).toEqual({ start: 0, end: 0 });
  });

  it('clampPage 限制在有效页码内', () => {
    expect(clampPage(99, 35, 30)).toBe(2);
    expect(clampPage(0, 35, 30)).toBe(1);
  });

  it('pageRange 计算显示区间', () => {
    expect(pageRange(35, 2, 30)).toEqual({ start: 31, end: 35 });
    expect(pageRange(5, 1, 30)).toEqual({ start: 1, end: 5 });
  });
});

describe('page size localStorage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('无记录时返回默认 30', () => {
    expect(loadStoredPageSize()).toBe(DEFAULT_PAGE_SIZE);
  });

  it('保存后刷新可读取', () => {
    saveStoredPageSize(50);
    expect(localStorage.getItem(PAGE_SIZE_STORAGE_KEY)).toBe('50');
    expect(loadStoredPageSize()).toBe(50);
  });

  it('非法值回退为默认', () => {
    localStorage.setItem(PAGE_SIZE_STORAGE_KEY, '999');
    expect(loadStoredPageSize()).toBe(DEFAULT_PAGE_SIZE);
  });
});
