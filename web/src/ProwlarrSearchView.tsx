import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { Download, Loader2, Search, Trash2, X } from 'lucide-react';
import { api } from './api';
import { useToast } from './Toast';
import type {
  ProwlarrConfig,
  ProwlarrDownloadInput,
  ProwlarrIndexer,
  ProwlarrRelease,
  ProwlarrSearchHistory,
  ProwlarrSearchType,
  ProwlarrSortBy
} from './types';

function formatBytes(n: number | null | undefined): string {
  if (n == null || n <= 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatTime(value?: string) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString('zh-CN', { hour12: false });
}

function messageOf(err: unknown) {
  return err instanceof Error ? err.message : '请求失败';
}

function releaseToDownloadInput(release: ProwlarrRelease, mediaType: ProwlarrSearchType): ProwlarrDownloadInput {
  return {
    guid: release.guid,
    title: release.title,
    media_type: mediaType,
    download_url: release.downloadUrl,
    info_hash: release.infoHash,
    indexer_id: release.indexerId,
    imdb_id: release.imdbId,
    tmdb_id: release.tmdbId,
    tvdb_id: release.tvdbId,
    season: release.season,
    episode: release.episode
  };
}

type ProwlarrSearchViewProps = {
  onGoSettings?: () => void;
  onGoActive?: () => void;
};

export function ProwlarrSearchView({ onGoSettings, onGoActive }: ProwlarrSearchViewProps) {
  const { showToast } = useToast();
  const [config, setConfig] = useState<ProwlarrConfig | null>(null);
  const [indexers, setIndexers] = useState<ProwlarrIndexer[]>([]);
  const [selectedIndexerIds, setSelectedIndexerIds] = useState<number[]>([]);
  const [searchType, setSearchType] = useState<ProwlarrSearchType>('movie');
  const [sortBy, setSortBy] = useState<ProwlarrSortBy>('seeders');
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<ProwlarrRelease[]>([]);
  const [resultsSearchType, setResultsSearchType] = useState<ProwlarrSearchType>('movie');
  const [history, setHistory] = useState<ProwlarrSearchHistory[]>([]);
  const [selectedGuids, setSelectedGuids] = useState<Set<string>>(new Set());
  const [searching, setSearching] = useState(false);
  const [downloadingGuid, setDownloadingGuid] = useState<string | null>(null);
  const [batchDownloading, setBatchDownloading] = useState(false);

  const loadHistory = useCallback(async () => {
    try {
      const data = await api.prowlarrSearchHistory();
      setHistory(data.items ?? []);
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }, [showToast]);

  useEffect(() => {
    api.prowlarrConfig().then((data) => {
      setConfig(data);
      setSelectedIndexerIds(data.indexer_ids ?? []);
    }).catch((err) => showToast(messageOf(err), 'error'));
  }, [showToast]);

  useEffect(() => {
    if (!config?.configured) return;
    api.prowlarrIndexers()
      .then((data) => setIndexers(data.items ?? []))
      .catch((err) => showToast(messageOf(err), 'error'));
    void loadHistory();
  }, [config?.configured, loadHistory, showToast]);

  const searchPlaceholder = useMemo(
    () => (searchType === 'tv' ? '例如：Breaking Bad 或 TVDB ID 80348' : '例如：Inception 或 tt1375666'),
    [searchType]
  );

  const runSearch = useCallback(async (searchQuery: string, opts?: {
    type?: ProwlarrSearchType;
    sort?: ProwlarrSortBy;
    indexerIds?: number[];
  }) => {
    const trimmed = searchQuery.trim();
    if (!trimmed) {
      showToast(opts?.type === 'tv' || searchType === 'tv' ? '请输入剧名或 TVDB ID' : '请输入片名或 IMDb ID', 'error');
      return;
    }
    const type = opts?.type ?? searchType;
    const sort = opts?.sort ?? sortBy;
    const indexerIds = opts?.indexerIds ?? selectedIndexerIds;
    setSearching(true);
    setSelectedGuids(new Set());
    try {
      const data = await api.searchProwlarr(trimmed, { type, sort, indexerIds });
      setResults(data.items ?? []);
      setResultsSearchType(type);
      await loadHistory();
      if ((data.items ?? []).length === 0) {
        showToast('未找到匹配的 Torrent 结果');
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setSearching(false);
    }
  }, [loadHistory, searchType, sortBy, selectedIndexerIds, showToast]);

  async function handleSearch(event: FormEvent) {
    event.preventDefault();
    await runSearch(query);
  }

  function applyHistory(entry: ProwlarrSearchHistory) {
    setQuery(entry.display_query);
    setSearchType(entry.media_type);
    setSortBy(entry.sort_by);
    setSelectedIndexerIds(entry.indexer_ids ?? []);
    void runSearch(entry.display_query, {
      type: entry.media_type,
      sort: entry.sort_by,
      indexerIds: entry.indexer_ids ?? []
    });
  }

  async function removeHistoryEntry(id: number) {
    try {
      await api.deleteProwlarrSearchHistory(id);
      setHistory((current) => current.filter((entry) => entry.id !== id));
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function clearHistory() {
    try {
      await api.clearProwlarrSearchHistory();
      setHistory([]);
      showToast('搜索历史已清空');
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  function toggleIndexer(id: number) {
    setSelectedIndexerIds((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id]
    );
  }

  function toggleResult(guid: string) {
    setSelectedGuids((current) => {
      const next = new Set(current);
      if (next.has(guid)) {
        next.delete(guid);
      } else {
        next.add(guid);
      }
      return next;
    });
  }

  function toggleSelectAll(checked: boolean) {
    if (!checked) {
      setSelectedGuids(new Set());
      return;
    }
    setSelectedGuids(new Set(results.map((release) => release.guid)));
  }

  async function downloadRelease(release: ProwlarrRelease) {
    setDownloadingGuid(release.guid);
    try {
      await api.downloadProwlarrRelease(releaseToDownloadInput(release, resultsSearchType));
      showToast('已提交下载');
      onGoActive?.();
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setDownloadingGuid(null);
    }
  }

  async function batchDownload() {
    const selected = results.filter((release) => selectedGuids.has(release.guid));
    if (selected.length === 0) {
      showToast('请先选择要下载的资源', 'error');
      return;
    }
    setBatchDownloading(true);
    try {
      const result = await api.batchDownloadProwlarrReleases(
        selected.map((release) => releaseToDownloadInput(release, resultsSearchType))
      );
      const successCount = result.items?.length ?? 0;
      const failureCount = result.failures?.length ?? 0;
      if (successCount > 0) {
        showToast(`已提交 ${successCount} 条下载${failureCount > 0 ? `，${failureCount} 条失败` : ''}`);
        onGoActive?.();
      } else if (failureCount > 0) {
        showToast(result.failures?.[0]?.error ?? '批量下载失败', 'error');
      }
      setSelectedGuids(new Set());
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setBatchDownloading(false);
    }
  }

  const allSelected = results.length > 0 && selectedGuids.size === results.length;

  if (config && !config.configured) {
    return (
      <section className="view">
        <header className="view-header">
          <h1>Prowlarr 搜索</h1>
          <p>通过 Prowlarr 搜索电影/剧集 Torrent，并直接提交到 aria2 下载。</p>
        </header>
        <div className="panel">
          <p className="muted">请先在设置页配置 Prowlarr 地址、API Key 与保存目录。</p>
          {onGoSettings && (
            <button type="button" className="primary" onClick={onGoSettings}>
              前往设置
            </button>
          )}
        </div>
      </section>
    );
  }

  return (
    <section className="view">
      <header className="view-header">
        <h1>Prowlarr 搜索</h1>
        <p>搜索电影或剧集 Torrent，支持搜索历史与批量下载。</p>
      </header>

      {history.length > 0 && (
        <div className="panel">
          <div className="horizontal-actions">
            <h2 className="section-title">搜索历史</h2>
            <button type="button" className="ghost" onClick={clearHistory}>
              <Trash2 size={14} aria-hidden />
              清空
            </button>
          </div>
          <div className="history-chips">
            {history.map((entry) => (
              <div key={entry.id} className="history-chip">
                <button type="button" className="history-chip-main" onClick={() => applyHistory(entry)}>
                  <span>{entry.display_query}</span>
                  <span className="muted">
                    {entry.media_type === 'tv' ? '剧集' : '电影'} · {entry.result_count} 条
                  </span>
                </button>
                <button type="button" className="history-chip-remove" aria-label="删除" onClick={() => removeHistoryEntry(entry.id)}>
                  <X size={14} aria-hidden />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <form className="panel" onSubmit={handleSearch}>
        <div className="horizontal-actions">
          <label>
            类型
            <select value={searchType} onChange={(event) => setSearchType(event.target.value as ProwlarrSearchType)}>
              <option value="movie">电影</option>
              <option value="tv">剧集</option>
            </select>
          </label>
          <label>
            排序
            <select value={sortBy} onChange={(event) => setSortBy(event.target.value as ProwlarrSortBy)}>
              <option value="seeders">做种数</option>
              <option value="size">体积</option>
              <option value="date">发布时间</option>
            </select>
          </label>
        </div>
        <label className="grow">
          关键词
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={searchPlaceholder} />
        </label>
        {indexers.length > 0 && (
          <fieldset className="indexer-fieldset">
            <legend>索引器（不选则搜索全部 Torrent 索引器）</legend>
            <div className="indexer-grid">
              {indexers.map((indexer) => (
                <label key={indexer.id} className="indexer-option">
                  <input
                    type="checkbox"
                    checked={selectedIndexerIds.includes(indexer.id)}
                    onChange={() => toggleIndexer(indexer.id)}
                  />
                  {indexer.name}
                </label>
              ))}
            </div>
          </fieldset>
        )}
        <div className="horizontal-actions">
          <button type="submit" className="primary" disabled={searching}>
            {searching ? <Loader2 size={16} className="icon-spinning" aria-hidden /> : <Search size={16} aria-hidden />}
            搜索
          </button>
        </div>
      </form>

      {results.length > 0 && (
        <div className="panel horizontal-actions">
          <label className="checkbox-row">
            <input type="checkbox" checked={allSelected} onChange={(event) => toggleSelectAll(event.target.checked)} />
            全选（{selectedGuids.size}/{results.length}）
          </label>
          <button type="button" className="primary" disabled={batchDownloading || selectedGuids.size === 0} onClick={batchDownload}>
            {batchDownloading ? <Loader2 size={16} className="icon-spinning" aria-hidden /> : <Download size={16} aria-hidden />}
            批量下载
          </button>
        </div>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>
                <span className="sr-only">选择</span>
              </th>
              <th>标题</th>
              <th>索引器</th>
              <th>大小</th>
              <th>做种</th>
              <th>下载</th>
              <th>发布时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {results.length === 0 ? (
              <tr>
                <td colSpan={8} className="empty">
                  {searching ? '搜索中…' : '输入关键词后搜索'}
                </td>
              </tr>
            ) : (
              results.map((release) => (
                <tr key={release.guid}>
                  <td>
                    <input
                      type="checkbox"
                      checked={selectedGuids.has(release.guid)}
                      onChange={() => toggleResult(release.guid)}
                      aria-label={`选择 ${release.title}`}
                    />
                  </td>
                  <td>{release.title}</td>
                  <td>{release.indexer || '—'}</td>
                  <td>{formatBytes(release.size)}</td>
                  <td>{release.seeders}</td>
                  <td>{release.leechers}</td>
                  <td>{formatTime(release.publishDate)}</td>
                  <td>
                    <button
                      type="button"
                      className="primary-link"
                      disabled={downloadingGuid === release.guid || batchDownloading}
                      onClick={() => downloadRelease(release)}
                    >
                      {downloadingGuid === release.guid ? (
                        <Loader2 size={14} className="icon-spinning" aria-hidden />
                      ) : (
                        <Download size={14} aria-hidden />
                      )}
                      下载
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
