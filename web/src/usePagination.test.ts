import { renderHook, act } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { PAGE_SIZE_STORAGE_KEY } from './listPaging';
import { usePagination } from './usePagination';

describe('usePagination', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('切换每页条数会写入 localStorage', () => {
    const { result } = renderHook(() => usePagination(100));

    act(() => {
      result.current.setPageSize(50);
    });

    expect(result.current.pageSize).toBe(50);
    expect(localStorage.getItem(PAGE_SIZE_STORAGE_KEY)).toBe('50');
  });

  it('初始化时从 localStorage 读取每页条数', () => {
    localStorage.setItem(PAGE_SIZE_STORAGE_KEY, '10');
    const { result } = renderHook(() => usePagination(100));

    expect(result.current.pageSize).toBe(10);
  });
});
