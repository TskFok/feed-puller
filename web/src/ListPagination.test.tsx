import { render, screen, fireEvent } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PaginationBar } from './ListPagination';

describe('PaginationBar', () => {
  it('超过一页时可翻页并切换每页条数', () => {
    const onPageChange = vi.fn();
    const onPageSizeChange = vi.fn();

    render(
      <PaginationBar
        page={1}
        pageSize={30}
        totalPages={2}
        totalItems={35}
        rangeStart={1}
        rangeEnd={30}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
      />
    );

    expect(screen.getByText('显示 1–30，共 35 条')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '下一页' }));
    expect(onPageChange).toHaveBeenCalledWith(2);

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '50' } });
    expect(onPageSizeChange).toHaveBeenCalledWith(50);
  });

  it('无数据时不渲染', () => {
    const { container } = render(
      <PaginationBar
        page={1}
        pageSize={30}
        totalPages={1}
        totalItems={0}
        rangeStart={0}
        rangeEnd={0}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });
});
