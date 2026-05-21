import { useCallback, useEffect, useState } from 'react';
import {
  DEFAULT_PAGE_SIZE,
  isPageSizeOption,
  loadStoredPageSize,
  paginateSlice,
  pageRange,
  saveStoredPageSize,
  totalPages,
  type PageSizeOption
} from './listPaging';

export function usePagination(itemCount: number, resetDeps: readonly unknown[] = []) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<PageSizeOption>(loadStoredPageSize);

  const pages = totalPages(itemCount, pageSize);
  const { start: rangeStart, end: rangeEnd } = pageRange(itemCount, page, pageSize);

  useEffect(() => {
    setPage(1);
  }, [pageSize, ...resetDeps]);

  useEffect(() => {
    setPage((current) => Math.min(current, pages));
  }, [pages]);

  const slice = useCallback(
    <T,>(items: T[]) => paginateSlice(items, page, pageSize),
    [page, pageSize]
  );

  const handlePageSizeChange = useCallback((next: number) => {
    const size: PageSizeOption = isPageSizeOption(next) ? next : DEFAULT_PAGE_SIZE;
    setPageSize(size);
    saveStoredPageSize(size);
  }, []);

  return {
    page,
    setPage,
    pageSize,
    setPageSize: handlePageSizeChange,
    totalPages: pages,
    totalItems: itemCount,
    rangeStart,
    rangeEnd,
    slice
  };
}
