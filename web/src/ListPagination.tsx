import { useId, useRef } from 'react';
import { resolveAppScrollElement } from './appScrollElement';
import { PAGE_SIZE_OPTIONS } from './listPaging';

export type PaginationBarProps = {
  page: number;
  pageSize: number;
  totalPages: number;
  totalItems: number;
  rangeStart: number;
  rangeEnd: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  hasMore?: boolean;
  busy?: boolean;
};

export function PaginationBar({
  page,
  pageSize,
  totalPages,
  totalItems,
  rangeStart,
  rangeEnd,
  onPageChange,
  onPageSizeChange,
  hasMore = false,
  busy = false
}: PaginationBarProps) {
  if (totalItems <= 0) {
    return null;
  }

  const pageSizeId = useId();
  const paginationRef = useRef<HTMLElement>(null);

  function handlePageChange(nextPage: number) {
    onPageChange(nextPage);
    const scrollElement = resolveAppScrollElement(paginationRef.current);
    if (scrollElement) {
      scrollElement.scrollTop = 0;
    }
  }

  return (
    <nav ref={paginationRef} className="pagination-bar" aria-label="列表分页">
      <div className="pagination-range muted">
        显示 {rangeStart}–{rangeEnd}，共 {totalItems} 条
      </div>
      <div className="pagination-controls">
        <label className="pagination-page-size" htmlFor={pageSizeId}>
          <span>每页</span>
          <select
            id={pageSizeId}
            className="form-select"
            value={pageSize}
            disabled={busy}
            onChange={(event) => onPageSizeChange(Number(event.target.value))}
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
          <span>条</span>
        </label>
        <div className="pagination-nav">
          <button
            type="button"
            className="ghost"
            disabled={page <= 1 || busy}
            aria-label="上一页"
            onClick={() => handlePageChange(page - 1)}
          >
            上一页
          </button>
          <span className="pagination-page-indicator" aria-live="polite">
            第 {page} / {totalPages} 页
          </span>
          <button
            type="button"
            className="ghost"
            disabled={(page >= totalPages && !hasMore) || busy}
            aria-label="下一页"
            onClick={() => handlePageChange(page + 1)}
          >
            下一页
          </button>
        </div>
      </div>
    </nav>
  );
}
