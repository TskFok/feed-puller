import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Download, Loader2, Search, Trash2, X } from 'lucide-react';
import { api } from './api';
import {
  addSessionSubmittedGuids,
  mergeSubmittedGuids,
  readSessionSubmittedGuids
} from './prowlarrSubmittedGuids';
import { useToast } from './Toast';
import { PROWLARR_OFFSCREEN_MIN_ITEMS } from './glassConstants';
import { ProwlarrReleaseCard } from './ProwlarrReleaseCard';
import { ProwlarrVirtualResultsGrid } from './ProwlarrVirtualResultsGrid';
import { useOffscreenGlassGrid } from './useOffscreenGlassGrid';
import {
  formatProwlarrBrowseProgress,
  hasBrowsedAllResults,
  mergeFurthestSeenIndex,
  type ProwlarrVisibleRange
} from './prowlarrResultsProgress';
import { useProwlarrVirtualizeThreshold } from './useProwlarrVirtualizeThreshold';
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

type BatchFailure = {
  guid: string;
  title: string;
  error: string;
};

type BatchSummary = {
  successCount: number;
  failureCount: number;
  failures: BatchFailure[];
};

function ProwlarrResultsSkeleton() {
  return (
    <>
      {Array.from({ length: 6 }, (_, index) => (
        <article key={index} className="prowlarr-release-card prowlarr-release-card--skeleton" aria-hidden="true">
          <div className="prowlarr-skeleton-line prowlarr-skeleton-line--title" />
          <div className="prowlarr-skeleton-meta">
            {Array.from({ length: 4 }, (_, metaIndex) => (
              <div key={metaIndex} className="prowlarr-skeleton-line prowlarr-skeleton-line--short" />
            ))}
          </div>
          <div className="prowlarr-skeleton-line prowlarr-skeleton-line--btn" />
        </article>
      ))}
    </>
  );
}

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
  const [activeHistoryId, setActiveHistoryId] = useState<number | null>(null);
  const [selectedGuids, setSelectedGuids] = useState<Set<string>>(new Set());
  const [submittedGuids, setSubmittedGuids] = useState<Set<string>>(new Set());
  const [batchSummary, setBatchSummary] = useState<BatchSummary | null>(null);
  const [batchFailuresExpanded, setBatchFailuresExpanded] = useState(false);
  const [searching, setSearching] = useState(false);
  const [downloadingGuid, setDownloadingGuid] = useState<string | null>(null);
  const [batchDownloading, setBatchDownloading] = useState(false);
  const [furthestSeenIndex, setFurthestSeenIndex] = useState(-1);
  const resultsGridRef = useRef<HTMLDivElement>(null);
  const hasRestoredLatestHistory = useRef(false);
  const virtualizeThreshold = useProwlarrVirtualizeThreshold();

  const useVirtualGrid = results.length > virtualizeThreshold;
  useOffscreenGlassGrid(
    resultsGridRef,
    !useVirtualGrid && results.length > PROWLARR_OFFSCREEN_MIN_ITEMS,
    [results.length, searching, useVirtualGrid]
  );

  const handleVisibleRangeChange = useCallback((range: ProwlarrVisibleRange) => {
    setFurthestSeenIndex((current) => mergeFurthestSeenIndex(current, range));
  }, []);

  useEffect(() => {
    if (results.length === 0) {
      setFurthestSeenIndex(-1);
      return;
    }
    if (!useVirtualGrid) {
      setFurthestSeenIndex(results.length - 1);
    }
  }, [results.length, useVirtualGrid]);

  const showDownloadSubmittedToast = useCallback(
    (message: string) => {
      showToast(message, 'success', onGoActive ? { action: { label: '查看进度', onClick: onGoActive } } : undefined);
    },
    [onGoActive, showToast]
  );

  const hydrateSubmittedGuids = useCallback(async (items: ProwlarrRelease[]) => {
    const guids = items.map((release) => release.guid).filter(Boolean);
    if (guids.length === 0) {
      setSubmittedGuids(new Set());
      return;
    }
    const sessionGuids = readSessionSubmittedGuids();
    try {
      const data = await api.prowlarrSubmittedGuids(guids);
      setSubmittedGuids(mergeSubmittedGuids(data.guids ?? [], sessionGuids, guids));
    } catch {
      setSubmittedGuids(mergeSubmittedGuids([], sessionGuids, guids));
    }
  }, []);

  const rememberSubmittedGuids = useCallback((guids: Iterable<string>) => {
    addSessionSubmittedGuids(guids);
    setSubmittedGuids((current) => {
      const next = new Set(current);
      for (const guid of guids) {
        if (guid) next.add(guid);
      }
      return next;
    });
  }, []);

  const clearDisplayedSearch = useCallback(() => {
    setQuery('');
    setResults([]);
    setSelectedGuids(new Set());
    setSubmittedGuids(new Set());
    setBatchSummary(null);
    setBatchFailuresExpanded(false);
    setFurthestSeenIndex(-1);
  }, []);

  const restoreHistoryEntry = useCallback(async (entry: ProwlarrSearchHistory) => {
    setActiveHistoryId(entry.id);
    setQuery(entry.display_query);
    setSearchType(entry.media_type);
    setSortBy(entry.sort_by);
    setSelectedIndexerIds(entry.indexer_ids ?? []);
    setSelectedGuids(new Set());
    setBatchSummary(null);
    setBatchFailuresExpanded(false);
    setFurthestSeenIndex(-1);
    try {
      const detail = await api.getProwlarrSearchHistory(entry.id);
      const items = detail.results ?? [];
      setResults(items);
      setResultsSearchType(entry.media_type);
      await hydrateSubmittedGuids(items);
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }, [hydrateSubmittedGuids, showToast]);

  const loadHistory = useCallback(async (options?: { restoreLatest?: boolean }): Promise<ProwlarrSearchHistory[]> => {
    try {
      const data = await api.prowlarrSearchHistory();
      const items = data.items ?? [];
      setHistory(items);
      if (options?.restoreLatest && !hasRestoredLatestHistory.current && items.length > 0) {
        hasRestoredLatestHistory.current = true;
        await restoreHistoryEntry(items[0]);
      }
      return items;
    } catch (err) {
      showToast(messageOf(err), 'error');
      return [];
    }
  }, [restoreHistoryEntry, showToast]);

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
    void loadHistory({ restoreLatest: true });
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
    setBatchSummary(null);
    setBatchFailuresExpanded(false);
    setFurthestSeenIndex(-1);
    try {
      const data = await api.searchProwlarr(trimmed, { type, sort, indexerIds });
      const items = data.items ?? [];
      setResults(items);
      setResultsSearchType(type);
      await hydrateSubmittedGuids(items);
      const historyItems = await loadHistory();
      const matched = historyItems.find(
        (entry) => entry.display_query === trimmed && entry.media_type === type
      );
      setActiveHistoryId(matched?.id ?? null);
      if (items.length === 0) {
        showToast('未找到匹配的 Torrent 结果');
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setSearching(false);
    }
  }, [hydrateSubmittedGuids, loadHistory, searchType, sortBy, selectedIndexerIds, showToast]);

  async function handleSearch(event: FormEvent) {
    event.preventDefault();
    await runSearch(query);
  }

  function applyHistory(entry: ProwlarrSearchHistory) {
    void restoreHistoryEntry(entry);
  }

  async function removeHistoryEntry(id: number) {
    try {
      await api.deleteProwlarrSearchHistory(id);
      setHistory((current) => current.filter((entry) => entry.id !== id));
      if (activeHistoryId === id) {
        setActiveHistoryId(null);
        clearDisplayedSearch();
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function clearHistory() {
    try {
      await api.clearProwlarrSearchHistory();
      setHistory([]);
      setActiveHistoryId(null);
      clearDisplayedSearch();
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
      rememberSubmittedGuids([release.guid]);
      showDownloadSubmittedToast('已提交下载');
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setDownloadingGuid(null);
    }
  }

  function markBatchSubmitted(selected: ProwlarrRelease[], failureGuids: Set<string>) {
    const successGuids = selected.filter((release) => !failureGuids.has(release.guid)).map((release) => release.guid);
    rememberSubmittedGuids(successGuids);
  }

  function buildBatchFailures(selected: ProwlarrRelease[], failures: { guid: string; error: string }[]): BatchFailure[] {
    const titles = new Map(selected.map((release) => [release.guid, release.title]));
    return failures.map((failure) => ({
      guid: failure.guid,
      title: titles.get(failure.guid) ?? failure.guid,
      error: failure.error
    }));
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
      const failureGuids = new Set(result.failures?.map((failure) => failure.guid) ?? []);
      const failures = buildBatchFailures(selected, result.failures ?? []);
      if (successCount > 0) {
        markBatchSubmitted(selected, failureGuids);
        setBatchSummary({ successCount, failureCount, failures });
        setBatchFailuresExpanded(failureCount > 0);
        showDownloadSubmittedToast(`已提交 ${successCount} 条下载${failureCount > 0 ? `，${failureCount} 条失败` : ''}`);
      } else if (failureCount > 0) {
        setBatchSummary({ successCount: 0, failureCount, failures });
        setBatchFailuresExpanded(true);
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
  const browsedAllResults = hasBrowsedAllResults(furthestSeenIndex, results.length);

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
        <div className="panel prowlarr-results-toolbar">
          <div className="horizontal-actions">
            <label className="checkbox-row">
              <input type="checkbox" checked={allSelected} onChange={(event) => toggleSelectAll(event.target.checked)} />
              全选（{selectedGuids.size}/{results.length}）
            </label>
            <button type="button" className="primary" disabled={batchDownloading || selectedGuids.size === 0} onClick={batchDownload}>
              {batchDownloading ? <Loader2 size={16} className="icon-spinning" aria-hidden /> : <Download size={16} aria-hidden />}
              批量下载
            </button>
          </div>
          <p className="prowlarr-results-progress muted" role="status">
            {formatProwlarrBrowseProgress(furthestSeenIndex, results.length)}
            {browsedAllResults && <span className="prowlarr-results-progress-complete"> · 已浏览全部结果</span>}
          </p>
          {batchSummary && (
            <div className="prowlarr-batch-summary-wrap">
              <p className="prowlarr-batch-summary" role="status">
                本次提交：成功 {batchSummary.successCount} 条
                {batchSummary.failureCount > 0 ? `，失败 ${batchSummary.failureCount} 条` : ''}
              </p>
              {batchSummary.failures.length > 0 && (
                <>
                  <button
                    type="button"
                    className="ghost prowlarr-batch-failures-toggle"
                    aria-expanded={batchFailuresExpanded}
                    onClick={() => setBatchFailuresExpanded((current) => !current)}
                  >
                    {batchFailuresExpanded ? '收起失败原因' : '查看失败原因'}
                  </button>
                  {batchFailuresExpanded && (
                    <ul className="prowlarr-batch-failures-list">
                      {batchSummary.failures.map((failure) => (
                        <li key={failure.guid}>
                          <strong>{failure.title}</strong>
                          <span>{failure.error}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      )}

      <div ref={resultsGridRef} className="prowlarr-results-host" aria-live="polite" aria-busy={searching}>
        {searching && results.length === 0 ? (
          <div className="prowlarr-results-grid">
            <ProwlarrResultsSkeleton />
          </div>
        ) : results.length === 0 ? (
          <p className="prowlarr-results-empty">输入关键词后搜索</p>
        ) : useVirtualGrid ? (
          <ProwlarrVirtualResultsGrid
            results={results}
            selectedGuids={selectedGuids}
            submittedGuids={submittedGuids}
            downloadingGuid={downloadingGuid}
            batchDownloading={batchDownloading}
            formatBytes={formatBytes}
            formatTime={formatTime}
            onToggle={toggleResult}
            onDownload={downloadRelease}
            onVisibleRangeChange={handleVisibleRangeChange}
          />
        ) : (
          <div className="prowlarr-results-grid prowlarr-results-grid--scrollable">
            {results.map((release) => (
              <ProwlarrReleaseCard
                key={release.guid}
                release={release}
                selected={selectedGuids.has(release.guid)}
                submitted={submittedGuids.has(release.guid)}
                downloading={downloadingGuid === release.guid}
                batchDownloading={batchDownloading}
                formatBytes={formatBytes}
                formatTime={formatTime}
                onToggle={() => toggleResult(release.guid)}
                onDownload={() => downloadRelease(release)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
