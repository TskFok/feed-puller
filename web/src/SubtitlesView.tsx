import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Download, Loader2, Search } from 'lucide-react';
import { api } from './api';
import { TruncatedText } from './TruncatedText';
import { useToast } from './Toast';
import { PaginationBar } from './ListPagination';
import { usePagination } from './usePagination';
import {
  filterSubtitleItems,
  joinSubtitleLanguages,
  LANGUAGE_OPTIONS,
  groupSubtitleItems,
  languageLabel,
  toggleSubtitleLanguage
} from './subtitleLanguages';
import type { OpenSubtitlesConfig, SubtitleSearchItem } from './types';

function messageOf(err: unknown) {
  return err instanceof Error ? err.message : '请求失败';
}

type SubtitlesViewProps = {
  onGoSettings?: () => void;
};

export function SubtitlesView({ onGoSettings }: SubtitlesViewProps) {
  const { showToast } = useToast();
  const [config, setConfig] = useState<OpenSubtitlesConfig | null>(null);
  const [query, setQuery] = useState('');
  const [languages, setLanguages] = useState<string[]>(['zh-CN']);
  const [items, setItems] = useState<SubtitleSearchItem[]>([]);
  const [osPage, setOsPage] = useState(0);
  const [osTotalPages, setOsTotalPages] = useState(0);
  const [searching, setSearching] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState('');
  const [downloadingFileId, setDownloadingFileId] = useState<number | null>(null);
  const [resultLanguage, setResultLanguage] = useState('');

  const allGroups = useMemo(() => groupSubtitleItems(items), [items]);
  const visibleGroups = useMemo(
    () => (resultLanguage === '' ? allGroups : allGroups.filter((group) => group.language === resultLanguage)),
    [allGroups, resultLanguage]
  );
  const flatVisible = useMemo(() => visibleGroups.flatMap((group) => group.items), [visibleGroups]);
  const pagination = usePagination(flatVisible.length, [resultLanguage]);
  const pagedItems = pagination.slice(flatVisible);
  const pagedGroups = groupSubtitleItems(pagedItems);
  const hasMore = osPage >= 1 && osPage < osTotalPages;
  const showLanguageFilter = allGroups.length > 1;

  useEffect(() => {
    api
      .openSubtitlesConfig()
      .then(setConfig)
      .catch((err) => setError(messageOf(err)));
  }, []);

  async function handleSearch(event: FormEvent) {
    event.preventDefault();
    const trimmed = query.trim();
    const languageParam = joinSubtitleLanguages(languages);
    if (!trimmed || !languageParam) {
      return;
    }
    setSearching(true);
    setError('');
    setResultLanguage('');
    setItems([]);
    setOsPage(0);
    setOsTotalPages(0);
    pagination.setPage(1);
    try {
      const data = await api.searchSubtitles(trimmed, languageParam, 1);
      setItems(data.items ?? []);
      setOsPage(data.page);
      setOsTotalPages(data.total_pages);
      setSearched(true);
    } catch (err) {
      setItems([]);
      setOsPage(0);
      setOsTotalPages(0);
      setError(messageOf(err));
    } finally {
      setSearching(false);
    }
  }

  async function loadMore() {
    const languageParam = joinSubtitleLanguages(languages);
    const trimmed = query.trim();
    if (loadingMore || !hasMore || !trimmed || !languageParam) {
      return;
    }
    const visibleBefore = filterSubtitleItems(items, resultLanguage).length;
    setLoadingMore(true);
    try {
      const data = await api.searchSubtitles(trimmed, languageParam, osPage + 1);
      const seen = new Set(items.map((item) => item.file_id));
      const appended = (data.items ?? []).filter((item) => !seen.has(item.file_id));
      const nextItems = [...items, ...appended];
      setItems(nextItems);
      setOsPage(data.page);
      setOsTotalPages(data.total_pages);
      const nextVisible = filterSubtitleItems(nextItems, resultLanguage).length;
      if (nextVisible > pagination.page * pagination.pageSize) {
        pagination.setPage(pagination.page + 1);
      } else if (resultLanguage !== '' && nextVisible === visibleBefore) {
        showToast('没有更多该语言结果');
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setLoadingMore(false);
    }
  }

  function handlePageChange(next: number) {
    if (next <= pagination.totalPages) {
      pagination.setPage(next);
      return;
    }
    void loadMore();
  }

  async function handleDownload(item: SubtitleSearchItem) {
    if (downloadingFileId != null) {
      return;
    }
    setDownloadingFileId(item.file_id);
    try {
      const result = await api.downloadSubtitle({ file_id: item.file_id, file_name: item.file_name });
      showToast(`已保存到 ${result.path}`);
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setDownloadingFileId(null);
    }
  }

  if (config === null && !error) {
    return (
      <section className="view">
        <header className="view-header">
          <h1>字幕</h1>
          <p>通过 OpenSubtitles 搜索字幕并保存到服务器目录。</p>
        </header>
        <div className="panel">
          <p className="muted">正在加载…</p>
        </div>
      </section>
    );
  }

  if (!config?.configured) {
    return (
      <section className="view">
        <header className="view-header">
          <h1>字幕</h1>
          <p>通过 OpenSubtitles 搜索字幕并保存到服务器目录。</p>
        </header>
        <div className="panel">
          {error ? <p role="alert">{error}</p> : (
            <p className="muted">请先在设置页填写 OpenSubtitles 用户名、密码、API Key 和下载目录。</p>
          )}
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
        <h1>字幕</h1>
        <p>按名称与语言搜索字幕，将选中文件保存到配置的服务器目录。</p>
      </header>

      <form className="panel" onSubmit={handleSearch}>
        <label className="grow">
          名称
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="例如：Inception" />
        </label>
        <fieldset className="language-fieldset">
          <legend>语言</legend>
          <div className="language-options">
            {LANGUAGE_OPTIONS.map((option) => (
              <label key={option.value} className="language-option">
                <input
                  type="checkbox"
                  checked={languages.includes(option.value)}
                  onChange={() => setLanguages((current) => toggleSubtitleLanguage(current, option.value))}
                />
                {option.label}
              </label>
            ))}
          </div>
        </fieldset>
        <div className="horizontal-actions">
          <button type="submit" className="primary" disabled={searching || !joinSubtitleLanguages(languages)}>
            {searching ? <Loader2 size={16} className="icon-spinning" aria-hidden /> : <Search size={16} aria-hidden />}
            搜索
          </button>
        </div>
      </form>

      {error ? <p role="alert">{error}</p> : null}

      {items.length > 0 ? (
        <div className="subtitles-results-wrap">
          {showLanguageFilter ? (
            <div className="subtitles-language-filter" role="group" aria-label="按语言筛选">
              <button
                type="button"
                className={`subtitles-language-filter-btn${resultLanguage === '' ? ' is-active' : ''}`}
                aria-pressed={resultLanguage === ''}
                onClick={() => setResultLanguage('')}
              >
                全部（{items.length}）
              </button>
              {allGroups.map((group) => (
                <button
                  key={group.language}
                  type="button"
                  className={`subtitles-language-filter-btn${resultLanguage === group.language ? ' is-active' : ''}`}
                  aria-pressed={resultLanguage === group.language}
                  onClick={() => setResultLanguage(group.language)}
                >
                  {languageLabel(group.language)}（{group.items.length}）
                </button>
              ))}
            </div>
          ) : null}
          <div className="table-wrap">
            <table className="subtitles-results">
              <colgroup>
                <col className="subtitles-col-release" />
                <col className="subtitles-col-language" />
                <col className="subtitles-col-file" />
                <col className="subtitles-col-count" />
                <col className="subtitles-col-rating" />
                <col className="subtitles-col-actions" />
              </colgroup>
              <thead>
                <tr>
                  <th>发行名</th>
                  <th>语言</th>
                  <th>文件名</th>
                  <th>下载次数</th>
                  <th>评分</th>
                  <th>操作</th>
                </tr>
              </thead>
              {pagedGroups.map((group) => (
                <tbody key={group.language || 'unknown'}>
                  {showLanguageFilter && resultLanguage === '' ? (
                    <tr className="subtitles-language-group">
                      <th scope="colgroup" colSpan={6}>
                        {languageLabel(group.language)}（{group.items.length}）
                      </th>
                    </tr>
                  ) : null}
                  {group.items.map((item) => (
                    <tr key={item.file_id}>
                      <td><TruncatedText>{item.release}</TruncatedText></td>
                      <td>{languageLabel(item.language)}</td>
                      <td><TruncatedText>{item.file_name}</TruncatedText></td>
                      <td>{item.download_count}</td>
                      <td>{item.ratings}</td>
                      <td>
                        <button
                          type="button"
                          className="primary"
                          disabled={downloadingFileId != null}
                          onClick={() => void handleDownload(item)}
                        >
                          {downloadingFileId === item.file_id ? (
                            <>
                              <Loader2 size={14} className="icon-spinning" aria-hidden />
                              下载中…
                            </>
                          ) : (
                            <>
                              <Download size={14} aria-hidden />
                              下载
                            </>
                          )}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              ))}
            </table>
          </div>
          <PaginationBar
            page={pagination.page}
            pageSize={pagination.pageSize}
            totalPages={pagination.totalPages}
            totalItems={pagination.totalItems}
            rangeStart={pagination.rangeStart}
            rangeEnd={pagination.rangeEnd}
            hasMore={hasMore}
            busy={loadingMore}
            onPageChange={handlePageChange}
            onPageSizeChange={pagination.setPageSize}
          />
        </div>
      ) : (
        <p className="muted">{searched ? '没有找到字幕' : '输入名称后搜索'}</p>
      )}
    </section>
  );
}
