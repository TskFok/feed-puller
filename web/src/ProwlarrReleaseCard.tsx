import { Download, Loader2 } from 'lucide-react';
import type { ProwlarrRelease } from './types';

export type ProwlarrReleaseCardProps = {
  release: ProwlarrRelease;
  selected: boolean;
  submitted: boolean;
  downloading: boolean;
  batchDownloading: boolean;
  formatBytes: (n: number | null | undefined) => string;
  formatTime: (value?: string) => string;
  onToggle: () => void;
  onDownload: () => void;
};

export function ProwlarrReleaseCard({
  release,
  selected,
  submitted,
  downloading,
  batchDownloading,
  formatBytes,
  formatTime,
  onToggle,
  onDownload
}: ProwlarrReleaseCardProps) {
  const cardClass = [
    'prowlarr-release-card',
    selected ? 'prowlarr-release-card--selected' : '',
    submitted ? 'prowlarr-release-card--submitted' : ''
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <article className={cardClass}>
      <div className="prowlarr-release-card-head">
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggle}
          aria-label={`选择 ${release.title}`}
        />
        <h3 className="prowlarr-release-title">{release.title}</h3>
        {submitted && <span className="status status-submitted prowlarr-release-status">已提交</span>}
      </div>
      <dl className="prowlarr-release-meta">
        <div>
          <dt>索引器</dt>
          <dd>{release.indexer || '—'}</dd>
        </div>
        <div>
          <dt>大小</dt>
          <dd>{formatBytes(release.size)}</dd>
        </div>
        <div>
          <dt>做种</dt>
          <dd>{release.seeders}</dd>
        </div>
        <div>
          <dt>下载</dt>
          <dd>{release.leechers}</dd>
        </div>
        <div>
          <dt>发布时间</dt>
          <dd>{formatTime(release.publishDate)}</dd>
        </div>
      </dl>
      <div className="prowlarr-release-actions">
        <button
          type="button"
          className="primary-link"
          disabled={downloading || batchDownloading}
          onClick={onDownload}
        >
          {downloading ? (
            <Loader2 size={14} className="icon-spinning" aria-hidden />
          ) : (
            <Download size={14} aria-hidden />
          )}
          {submitted ? '重新下载' : '下载'}
        </button>
      </div>
    </article>
  );
}
