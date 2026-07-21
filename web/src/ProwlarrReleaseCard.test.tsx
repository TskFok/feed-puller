import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ProwlarrReleaseCard } from './ProwlarrReleaseCard';

describe('ProwlarrReleaseCard', () => {
  it('将长标题作为可截断文本并保留完整悬停提示', () => {
    const title = '一个非常长的 Prowlarr 搜索结果标题，用于验证卡片标题不会换行';
    render(
      <ProwlarrReleaseCard
        release={{ guid: 'guid-1', title, indexer: 'Tracker', indexerId: 1, size: 1024, seeders: 1, leechers: 0, protocol: 'torrent' }}
        selected={false}
        submitted={false}
        downloading={false}
        batchDownloading={false}
        formatBytes={() => '1 KB'}
        formatTime={() => '—'}
        onToggle={vi.fn()}
        onDownload={vi.fn()}
      />
    );

    const heading = screen.getByRole('heading', { name: title });
    const text = screen.getByTitle(title);
    expect(heading).toContainElement(text);
    expect(text).toHaveClass('truncated-text');
    expect(heading.parentElement).toHaveClass('prowlarr-release-card-head');
    expect(screen.getByRole('button', { name: '下载' })).toHaveClass('prowlarr-release-download');
    expect(screen.getByRole('button', { name: '下载' }).parentElement).toHaveClass('prowlarr-release-actions');
  });
});
