import { useCallback, useEffect, useRef, useState } from 'react';
import {
  DEFAULT_PAGE_SIZE,
  isPageSizeOption,
  loadStoredPageSize,
  pageRange,
  saveStoredPageSize,
  totalPages,
  type PageSizeOption
} from './listPaging';
import type { PaginatedResult } from './types';

type Loader<T> = (page: number, pageSize: PageSizeOption) => Promise<PaginatedResult<T>>;

type UseServerPaginationOptions = {
  resetDeps?: readonly unknown[];
  onError?: (err: unknown) => void;
};

export function useServerPagination<T>(loader: Loader<T>, options: UseServerPaginationOptions = {}) {
  const { resetDeps = [], onError } = options;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<PageSizeOption>(loadStoredPageSize);
  const [items, setItems] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const pages = totalPages(total, pageSize);
  const { start: rangeStart, end: rangeEnd } = pageRange(total, page, pageSize);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const res = await loader(page, pageSize);
      const nextTotal = res.total ?? 0;
      const maxPage = totalPages(nextTotal, pageSize);
      const safePage = Math.min(page, maxPage);
      if (safePage !== page) {
        setPage(safePage);
        return;
      }
      setItems(Array.isArray(res.items) ? res.items : []);
      setTotal(nextTotal);
    } catch (err) {
      onErrorRef.current?.(err);
    } finally {
      setLoading(false);
    }
  }, [loader, page, pageSize]);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    setPage(1);
  }, [pageSize, ...resetDeps]);

  const handlePageSizeChange = useCallback((next: number) => {
    const size: PageSizeOption = isPageSizeOption(next) ? next : DEFAULT_PAGE_SIZE;
    setPageSize(size);
    saveStoredPageSize(size);
  }, []);

  return {
    items,
    setItems,
    total,
    loading,
    reload,
    page,
    setPage,
    pageSize,
    setPageSize: handlePageSizeChange,
    totalPages: pages,
    rangeStart,
    rangeEnd
  };
}
