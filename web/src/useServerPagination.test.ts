import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useServerPagination } from './useServerPagination';

describe('useServerPagination', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('按页请求并展示总数', async () => {
    const loader = vi.fn(async (page: number, pageSize: number) => ({
      items: [{ id: page }],
      total: 35,
      page,
      page_size: pageSize
    }));

    const { result } = renderHook(() => useServerPagination(loader));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(loader).toHaveBeenCalledWith(1, 30);
    expect(result.current.items).toEqual([{ id: 1 }]);
    expect(result.current.total).toBe(35);
    expect(result.current.totalPages).toBe(2);

    act(() => {
      result.current.setPage(2);
    });

    await waitFor(() => expect(loader).toHaveBeenCalledWith(2, 30));
  });
});
