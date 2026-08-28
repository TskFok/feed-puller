import { FormEvent, useEffect, useState } from 'react';
import { Download, Loader2, Search } from 'lucide-react';
import { api } from './api';
import { useToast } from './Toast';
import type { OpenSubtitlesConfig, SubtitleSearchItem } from './types';

const LANGUAGE_OPTIONS = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁体中文' },
  { value: 'en', label: '英语' },
  { value: 'ja', label: '日语' },
  { value: 'ko', label: '韩语' }
] as const;

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
  const [languages, setLanguages] = useState('zh-CN');
  const [items, setItems] = useState<SubtitleSearchItem[]>([]);
  const [searching, setSearching] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState('');
  const [downloadingFileId, setDownloadingFileId] = useState<number | null>(null);

  useEffect(() => {
    api
      .openSubtitlesConfig()
      .then(setConfig)
      .catch((err) => setError(messageOf(err)));
  }, []);

  async function handleSearch(event: FormEvent) {
    event.preventDefault();
    const trimmed = query.trim();
    if (!trimmed) {
      return;
    }
    setSearching(true);
    setError('');
    try {
      const data = await api.searchSubtitles(trimmed, languages);
      setItems(data.items ?? []);
      setSearched(true);
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setSearching(false);
    }
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

  if (config && !config.configured) {
    return (
      <section className="view">
        <header className="view-header">
          <h1>字幕</h1>
          <p>通过 OpenSubtitles 搜索字幕并保存到服务器目录。</p>
        </header>
        <div className="panel">
          <p className="muted">请先在设置页填写 OpenSubtitles 用户名、密码、API Key 和下载目录。</p>
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
        <div className="horizontal-actions">
          <label className="grow">
            名称
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="例如：Inception" />
          </label>
          <label>
            语言
            <select value={languages} onChange={(event) => setLanguages(event.target.value)}>
              {LANGUAGE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="horizontal-actions">
          <button type="submit" className="primary" disabled={searching}>
            {searching ? <Loader2 size={16} className="icon-spinning" aria-hidden /> : <Search size={16} aria-hidden />}
            搜索
          </button>
        </div>
      </form>

      {error ? <p role="alert">{error}</p> : null}

      {items.length > 0 ? (
        <div className="table-wrap">
          <table>
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
            <tbody>
              {items.map((item) => (
                <tr key={item.file_id}>
                  <td>{item.release}</td>
                  <td>{item.language}</td>
                  <td>{item.file_name}</td>
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
          </table>
        </div>
      ) : (
        <p className="muted">{searched ? '没有找到字幕' : '输入名称后搜索'}</p>
      )}
    </section>
  );
}
