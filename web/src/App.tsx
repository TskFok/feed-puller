import { FormEvent, useEffect, useId, useMemo, useRef, useState } from 'react';
import {
  Download,
  LogOut,
  Plus,
  RefreshCw,
  Rss,
  Settings,
  ShieldCheck,
  SquarePen,
  Trash2,
  X
} from 'lucide-react';
import { api } from './api';
import type {
  DownloadTask,
  FeedItem,
  PolledFeedItem,
  PollSchedulePreviewInput,
  Subscription,
  User
} from './types';

type Tab = 'subscriptions' | 'items' | 'downloads' | 'settings';

/** 下拉快捷项；仍可手动输入任意 IANA 标识 */
const COMMON_IANA_TIMEZONES = [
  'UTC',
  'Asia/Shanghai',
  'Asia/Hong_Kong',
  'Asia/Tokyo',
  'Asia/Singapore',
  'Europe/Berlin',
  'Europe/London',
  'America/New_York',
  'America/Los_Angeles'
] as const;

const emptySubscription: Omit<Subscription, 'id'> = {
  name: '',
  feed_url: '',
  enabled: true,
  poll_interval_minutes: 30,
  poll_cron: '',
  poll_cron_timezone: 'UTC',
  download_dir: '',
  include_keywords: '',
  exclude_keywords: '',
  use_proxy: false
};

export function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <div className="boot">正在加载 feed-puller</div>;
  }

  if (!user) {
    return <LoginView onLogin={setUser} error={error} setError={setError} />;
  }

  return <Shell user={user} setUser={setUser} />;
}

function LoginView({ onLogin, error, setError }: { onLogin: (user: User) => void; error: string; setError: (value: string) => void }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError('');
    try {
      onLogin(await api.login(email, password));
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="login-screen">
      <section className="login-panel" aria-labelledby="login-title">
        <div>
          <p className="eyebrow">RSS 自动下载</p>
          <h1 id="login-title">feed-puller</h1>
          <p className="muted">登录后管理订阅、代理和 aria2 下载任务。</p>
        </div>
        <form onSubmit={submit} className="form">
          <label>
            邮箱
            <input value={email} type="email" autoComplete="email" onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label>
            密码
            <input value={password} type="password" autoComplete="current-password" onChange={(event) => setPassword(event.target.value)} required />
          </label>
          {error && <p className="error">{error}</p>}
          <button className="primary" disabled={submitting}>
            {submitting ? '登录中' : '登录'}
          </button>
          <a className="secondary-link" href="/api/auth/feishu/start">
            使用已绑定的飞书账号登录
          </a>
        </form>
      </section>
    </main>
  );
}

function Shell({ user, setUser }: { user: User; setUser: (user: User | null) => void }) {
  const [tab, setTab] = useState<Tab>('subscriptions');

  async function logout() {
    await api.logout().catch(() => null);
    setUser(null);
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <Rss size={22} aria-hidden="true" />
          <span>feed-puller</span>
        </div>
        <nav className="nav" aria-label="主导航">
          <NavButton tab="subscriptions" active={tab} setTab={setTab} icon={<Rss size={18} />} label="订阅" />
          <NavButton tab="items" active={tab} setTab={setTab} icon={<SquarePen size={18} />} label="条目" />
          <NavButton tab="downloads" active={tab} setTab={setTab} icon={<Download size={18} />} label="下载" />
          <NavButton tab="settings" active={tab} setTab={setTab} icon={<Settings size={18} />} label="设置" />
        </nav>
        <div className="account">
          <span>{user.email}</span>
          <button className="ghost" onClick={logout}>
            <LogOut size={16} aria-hidden="true" />
            退出
          </button>
        </div>
      </aside>
      <main className="workspace">
        {tab === 'subscriptions' && <SubscriptionsView />}
        {tab === 'items' && <ItemsView />}
        {tab === 'downloads' && <DownloadsView />}
        {tab === 'settings' && <SettingsView user={user} setUser={setUser} />}
      </main>
    </div>
  );
}

function NavButton({ tab, active, setTab, icon, label }: { tab: Tab; active: Tab; setTab: (tab: Tab) => void; icon: JSX.Element; label: string }) {
  return (
    <button className={active === tab ? 'nav-button active' : 'nav-button'} onClick={() => setTab(tab)}>
      {icon}
      {label}
    </button>
  );
}

type PollScheduleDisplay = Pick<Subscription, 'enabled' | 'next_poll_at'>;

function formatLastFetchedLabel(at?: string): string {
  if (!at) return '尚未拉取';
  const label = formatTime(at);
  return label || '—';
}

function formatNextPollLabel(source: PollScheduleDisplay): string {
  if (!source.enabled) {
    return '已禁用';
  }
  const at = source.next_poll_at;
  if (!at) {
    return '—';
  }
  const when = new Date(at);
  if (Number.isNaN(when.getTime())) {
    return '—';
  }
  if (when.getTime() <= Date.now()) {
    return '已到预计时间';
  }
  return formatTime(at);
}

function schedulePayloadFromDraft(
  draft: Omit<Subscription, 'id'>,
  scheduleKind: 'interval' | 'cron',
  anchors: { last_fetched_at?: string; created_at?: string }
): PollSchedulePreviewInput {
  return {
    enabled: draft.enabled,
    poll_interval_minutes: draft.poll_interval_minutes > 0 ? draft.poll_interval_minutes : 30,
    poll_cron: scheduleKind === 'cron' ? draft.poll_cron.trim() : '',
    poll_cron_timezone: scheduleKind === 'cron' ? draft.poll_cron_timezone.trim() || 'UTC' : 'UTC',
    last_fetched_at: anchors.last_fetched_at,
    created_at: anchors.created_at
  };
}

function SubscriptionPollMeta({
  enabled,
  lastFetchedAt,
  nextPollAt,
  previewError,
  previewLoading
}: {
  enabled: boolean;
  lastFetchedAt?: string;
  nextPollAt?: string;
  previewError?: string;
  previewLoading?: boolean;
}) {
  const nextLabel = previewError
    ? previewError
    : previewLoading
      ? '推算中…'
      : formatNextPollLabel({ enabled, next_poll_at: nextPollAt });
  return (
    <div className="subscription-poll-meta" aria-live="polite">
      <p>
        上次拉取：<strong>{formatLastFetchedLabel(lastFetchedAt)}</strong>
      </p>
      <p>
        下次预计拉取：<strong>{nextLabel}</strong>
      </p>
    </div>
  );
}

function useSchedulePreview(
  draft: Omit<Subscription, 'id'>,
  scheduleKind: 'interval' | 'cron',
  anchors: { last_fetched_at?: string; created_at?: string }
) {
  const [nextPollAt, setNextPollAt] = useState<string | undefined>();
  const [previewError, setPreviewError] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setPreviewLoading(true);
    setPreviewError('');
    const payload = schedulePayloadFromDraft(draft, scheduleKind, anchors);
    const timer = setTimeout(() => {
      api
        .previewNextPoll(payload)
        .then((res) => {
          if (cancelled) return;
          if (res.error) {
            setPreviewError(res.error);
            setNextPollAt(undefined);
          } else {
            setPreviewError('');
            setNextPollAt(res.next_poll_at);
          }
        })
        .catch((err) => {
          if (cancelled) return;
          setPreviewError(messageOf(err));
          setNextPollAt(undefined);
        })
        .finally(() => {
          if (!cancelled) setPreviewLoading(false);
        });
    }, 350);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [
    draft.enabled,
    scheduleKind,
    draft.poll_interval_minutes,
    draft.poll_cron,
    draft.poll_cron_timezone,
    anchors.last_fetched_at,
    anchors.created_at
  ]);

  return { nextPollAt, previewError, previewLoading };
}

function subscriptionScheduleSummary(sub: Subscription) {
  return (
    <div className="sub-schedule-summary">
      <div>
        <span className="sub-schedule-label">上次</span> {formatLastFetchedLabel(sub.last_fetched_at)}
      </div>
      <div>
        <span className="sub-schedule-label">下次</span> {formatNextPollLabel(sub)}
      </div>
    </div>
  );
}

function subscriptionToDraft(sub: Subscription): Omit<Subscription, 'id'> {
  return {
    name: sub.name,
    feed_url: sub.feed_url,
    enabled: sub.enabled,
    poll_interval_minutes: sub.poll_interval_minutes,
    poll_cron: sub.poll_cron ?? '',
    poll_cron_timezone: sub.poll_cron_timezone?.trim() || 'UTC',
    download_dir: sub.download_dir,
    include_keywords: sub.include_keywords ?? '',
    exclude_keywords: sub.exclude_keywords ?? '',
    use_proxy: sub.use_proxy
  };
}

type SubscriptionModalTarget = { mode: 'create' } | { mode: 'edit'; subscriptionId: number };

function SubscriptionModal({
  target,
  subscriptions,
  onClose,
  onSuccess
}: {
  target: SubscriptionModalTarget;
  subscriptions: Subscription[];
  onClose: () => void;
  onSuccess: (saved: Subscription) => void | Promise<void>;
}) {
  const isCreate = target.mode === 'create';
  const subscription =
    target.mode === 'edit' ? subscriptions.find((s) => s.id === target.subscriptionId) : undefined;
  const titleId = useId();
  const ianaTzListId = useId();
  const firstFieldRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft] = useState<Omit<Subscription, 'id'>>(() =>
    isCreate || !subscription ? { ...emptySubscription } : subscriptionToDraft(subscription)
  );
  const [scheduleKind, setScheduleKind] = useState<'interval' | 'cron'>(() =>
    isCreate || !subscription ? 'interval' : subscription.poll_cron.trim() !== '' ? 'cron' : 'interval'
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const previewAnchors = useMemo(
    () => ({
      last_fetched_at: subscription?.last_fetched_at,
      created_at: subscription?.created_at
    }),
    [subscription?.last_fetched_at, subscription?.created_at]
  );
  const { nextPollAt, previewError, previewLoading } = useSchedulePreview(draft, scheduleKind, previewAnchors);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  useEffect(() => {
    firstFieldRef.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError('');
    try {
      const payload: Omit<Subscription, 'id'> =
        scheduleKind === 'cron'
          ? {
              ...draft,
              poll_interval_minutes: draft.poll_interval_minutes || 30,
              poll_cron_timezone: draft.poll_cron_timezone.trim() || 'UTC'
            }
          : { ...draft, poll_cron: '', poll_cron_timezone: 'UTC' };
      const saved =
        target.mode === 'create'
          ? await api.createSubscription(payload)
          : await api.updateSubscription(target.subscriptionId, payload);
      await onSuccess(saved);
      onClose();
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setSaving(false);
    }
  }

  if (!isCreate && !subscription) {
    return null;
  }

  return (
    <div className="modal-overlay" role="presentation" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="modal-panel"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="modal-header-row">
          <div>
            <h2 id={titleId} className="modal-title">
              {isCreate ? '新增订阅' : '编辑订阅'}
            </h2>
            <p className="muted modal-subtitle">
              {isCreate
                ? '保存后不会自动拉取；请在列表点「拉取」或按 Crontab/间隔等待调度。'
                : `#${subscription!.id} · ${subscription!.feed_url}`}
            </p>
          </div>
          <button
            type="button"
            className="modal-close ghost"
            aria-label={isCreate ? '关闭新建订阅' : '关闭编辑订阅'}
            onClick={onClose}
          >
            <X size={20} aria-hidden="true" />
          </button>
        </div>
        <form className="subscription-edit-form" onSubmit={submit}>
          <fieldset className="modal-fieldset">
            <legend>
              <span className="modal-section-title">基本信息</span>
            </legend>
            <label className="modal-full">
              订阅名称
              <input
                ref={firstFieldRef}
                value={draft.name}
                onChange={(event) => setDraft({ ...draft, name: event.target.value })}
                required
              />
            </label>
            <label className="modal-full">
              订阅地址
              <input
                value={draft.feed_url}
                onChange={(event) => setDraft({ ...draft, feed_url: event.target.value })}
                required
                spellCheck={false}
              />
            </label>
          </fieldset>

          <fieldset className="modal-fieldset modal-fieldset-schedule">
            <legend>
              <span className="modal-section-title">抓取与选项</span>
            </legend>
            <div className="schedule-mode-row" role="radiogroup" aria-label="拉取调度方式">
              <label className="check modal-check-inline">
                <input
                  type="radio"
                  name="schedule-kind"
                  checked={scheduleKind === 'interval'}
                  onChange={() => {
                    setScheduleKind('interval');
                    setDraft((d) => ({ ...d, poll_cron: '', poll_cron_timezone: 'UTC' }));
                  }}
                />
                固定间隔
              </label>
              <label className="check modal-check-inline">
                <input
                  type="radio"
                  name="schedule-kind"
                  checked={scheduleKind === 'cron'}
                  onChange={() => {
                    setScheduleKind('cron');
                  }}
                />
                Crontab
              </label>
            </div>
            {scheduleKind === 'interval' ? (
              <label>
                间隔（分钟）
                <input
                  type="number"
                  min={1}
                  value={draft.poll_interval_minutes}
                  onChange={(event) =>
                    setDraft({ ...draft, poll_interval_minutes: Number(event.target.value) })
                  }
                  required
                />
              </label>
            ) : (
              <>
                <label className="modal-full">
                  表达式（分 时 日 月 周）
                  <input
                    aria-label="Crontab 表达式"
                    value={draft.poll_cron}
                    onChange={(event) => setDraft({ ...draft, poll_cron: event.target.value })}
                    placeholder="例如每六小时一次：0 */6 * * *"
                    spellCheck={false}
                    required
                  />
                </label>
                <label className="modal-full">
                  Crontab 时区（IANA）
                  <input
                    aria-label="Crontab 时区（IANA）"
                    list={ianaTzListId}
                    value={draft.poll_cron_timezone}
                    onChange={(event) => setDraft({ ...draft, poll_cron_timezone: event.target.value })}
                    placeholder="Asia/Shanghai（空则视为 UTC）"
                    spellCheck={false}
                  />
                  <datalist id={ianaTzListId}>
                    {COMMON_IANA_TIMEZONES.map((zone) => (
                      <option key={zone} value={zone} />
                    ))}
                  </datalist>
                </label>
              </>
            )}
            {scheduleKind === 'cron' && (
              <p className="muted modal-schedule-help">
                Crontab 在五字段（分 时 日 月 周）下按所选 IANA 时区解释字段含义；未填时区在提交时会按 UTC
                处理。表达式内不要使用 TZ=/CRON_TZ=。支持 @hourly 等别名。调度器每分钟检查一次。
              </p>
            )}
            <SubscriptionPollMeta
              enabled={draft.enabled}
              lastFetchedAt={subscription?.last_fetched_at}
              nextPollAt={nextPollAt}
              previewError={previewError}
              previewLoading={previewLoading}
            />
            <label className="check modal-check-inline">
              <input
                type="checkbox"
                checked={draft.enabled}
                onChange={(event) => setDraft({ ...draft, enabled: event.target.checked })}
              />
              启用此订阅
            </label>
            <label className="check modal-check-inline">
              <input
                type="checkbox"
                checked={draft.use_proxy}
                onChange={(event) => setDraft({ ...draft, use_proxy: event.target.checked })}
              />
              使用代理服务器拉取
            </label>
          </fieldset>

          <fieldset className="modal-fieldset">
            <legend>
              <span className="modal-section-title">保存路径</span>
            </legend>
            <label className="modal-full">
              下载目录
              <input
                value={draft.download_dir}
                onChange={(event) => setDraft({ ...draft, download_dir: event.target.value })}
                required
                spellCheck={false}
              />
            </label>
          </fieldset>

          <fieldset className="modal-fieldset">
            <legend>
              <span className="modal-section-title">条目过滤（正则）</span>
            </legend>
            <p className="modal-hint muted">每行一条正则；匹配标题、链接与下载地址。包含留空表示不过滤；排除命中任一则丢弃该条目。</p>
            <div className="modal-keyword-grid">
              <label>
                包含关键字
                <textarea
                  aria-label="包含关键字"
                  value={draft.include_keywords}
                  rows={5}
                  onChange={(event) =>
                    setDraft({ ...draft, include_keywords: event.target.value })
                  }
                  placeholder={'例如：\\.mp4$\n1080p'}
                />
              </label>
              <label>
                排除关键字
                <textarea
                  aria-label="排除关键字"
                  value={draft.exclude_keywords}
                  rows={5}
                  onChange={(event) =>
                    setDraft({ ...draft, exclude_keywords: event.target.value })
                  }
                  placeholder="例如：预告|Preview"
                />
              </label>
            </div>
          </fieldset>

          {error && <p className="error modal-error">{error}</p>}
          <div className="modal-actions">
            <button type="button" className="ghost" disabled={saving} onClick={onClose}>
              取消
            </button>
            <button type="submit" className="primary" disabled={saving}>
              {saving ? (isCreate ? '创建中…' : '保存中…') : isCreate ? '创建订阅' : '保存更改'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function formatBytes(n: number | null | undefined): string {
  if (n == null || n < 0) return '未知';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let u = 0;
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024;
    u++;
  }
  const digits = u === 0 ? 0 : v >= 10 ? 1 : 1;
  return `${v.toFixed(digits)} ${units[u]}`;
}

type FeedItemDownloadRow = { download_url?: string; download_status: string };

function canDownloadFeedItem(row: FeedItemDownloadRow): boolean {
  const url = row.download_url?.trim();
  if (!url) return false;
  return row.download_status !== 'submitting';
}

function feedItemDownloadButtonLabel(status: string): string {
  if (status === 'pending') return '下载';
  if (status === 'failed') return '重试';
  return '重新下载';
}

function fetchPreviewStatus(row: PolledFeedItem): string {
  const url = row.download_url?.trim();
  if (!url) return '无可下载';
  if (row.download_status === 'pending') return '未处理';
  if (row.download_status === 'failed') return '失败';
  if (row.download_status === 'submitting') return '提交中';
  return '已处理';
}

function FetchPreviewModal({
  subscriptionName,
  initialItems,
  onClose
}: {
  subscriptionName: string;
  initialItems: PolledFeedItem[];
  onClose: () => void;
}) {
  const titleId = useId();
  const selectAllId = useId();
  const [rows, setRows] = useState<PolledFeedItem[]>(initialItems);
  const [selected, setSelected] = useState<Set<number>>(() => new Set());
  const [rowLoading, setRowLoading] = useState<number | null>(null);
  const [batchLoading, setBatchLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  const downloadableRows = useMemo(() => rows.filter((row) => canDownloadFeedItem(row)), [rows]);
  const downloadableIds = useMemo(() => downloadableRows.map((row) => row.id), [downloadableRows]);
  const selectedDownloadableCount = useMemo(
    () => downloadableIds.filter((id) => selected.has(id)).length,
    [downloadableIds, selected]
  );
  const allDownloadableSelected =
    downloadableIds.length > 0 && selectedDownloadableCount === downloadableIds.length;

  function applyUpdatedItems(updates: FeedItem[]) {
    if (updates.length === 0) return;
    const byId = new Map(updates.map((item) => [item.id, item]));
    setRows((prev) =>
      prev.map((row) => {
        const updated = byId.get(row.id);
        return updated ? { ...row, ...updated, content_length: row.content_length } : row;
      })
    );
  }

  function toggleRowSelected(id: number, checked: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }

  function toggleSelectAllDownloadable() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allDownloadableSelected) {
        for (const id of downloadableIds) next.delete(id);
      } else {
        for (const id of downloadableIds) next.add(id);
      }
      return next;
    });
  }

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  async function downloadRow(row: PolledFeedItem) {
    setError('');
    setNotice('');
    setRowLoading(row.id);
    try {
      const updated = await api.downloadFeedItem(row.id);
      applyUpdatedItems([updated]);
      setSelected((prev) => {
        const next = new Set(prev);
        next.delete(row.id);
        return next;
      });
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setRowLoading(null);
    }
  }

  async function downloadSelected() {
    const ids = downloadableIds.filter((id) => selected.has(id));
    if (ids.length === 0) return;
    setError('');
    setNotice('');
    setBatchLoading(true);
    try {
      const result = await api.batchDownloadFeedItems(ids);
      applyUpdatedItems(result.items);
      setSelected((prev) => {
        const next = new Set(prev);
        for (const item of result.items) next.delete(item.id);
        return next;
      });
      const ok = result.items.length;
      const failed = result.failures?.length ?? 0;
      if (ok > 0) {
        setNotice(`已提交 ${ok} 条下载任务`);
      }
      if (failed > 0) {
        const detail = result.failures!.map((f) => `#${f.item_id}: ${f.error}`).join('；');
        setError(`有 ${failed} 条未成功：${detail}`);
      }
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setBatchLoading(false);
    }
  }

  return (
    <div className="modal-overlay" role="presentation" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="modal-panel fetch-preview-modal"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="modal-header-row">
          <div>
            <h2 id={titleId} className="modal-title">
              拉取结果 · {subscriptionName}
            </h2>
            <p className="muted modal-subtitle">
              勾选条目后可批量下载；单条亦可点击操作列。已处理条目可重新下载，无可下载地址的条目无法勾选。
            </p>
          </div>
          <button type="button" className="modal-close ghost" aria-label="关闭拉取结果" onClick={onClose}>
            <X size={20} aria-hidden="true" />
          </button>
        </div>
        {notice && <p className="notice modal-notice">{notice}</p>}
        {error && <p className="error modal-error">{error}</p>}
        {downloadableIds.length > 0 && (
          <div className="fetch-preview-toolbar">
            <label className="check fetch-preview-select-all">
              <input
                id={selectAllId}
                type="checkbox"
                checked={allDownloadableSelected}
                disabled={batchLoading || rowLoading !== null}
                onChange={toggleSelectAllDownloadable}
              />
              全选可下载（{downloadableIds.length}）
            </label>
            <button
              type="button"
              className="primary"
              disabled={batchLoading || rowLoading !== null || selectedDownloadableCount === 0}
              onClick={downloadSelected}
            >
              <Download size={16} aria-hidden="true" />
              {batchLoading ? '提交中…' : `批量下载（${selectedDownloadableCount}）`}
            </button>
          </div>
        )}
        <div className="table-wrap fetch-preview-table-wrap">
          <table>
            <thead>
              <tr>
                <th className="fetch-preview-col-check" scope="col">
                  <span className="sr-only">选择</span>
                </th>
                <th>名称</th>
                <th>文件大小</th>
                <th>状态</th>
                <th>时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td className="fetch-preview-col-check">
                    {canDownloadFeedItem(row) ? (
                      <input
                        type="checkbox"
                        aria-label={`选择 ${row.title || row.link || '条目'}`}
                        checked={selected.has(row.id)}
                        disabled={batchLoading || rowLoading !== null}
                        onChange={(event) => toggleRowSelected(row.id, event.target.checked)}
                      />
                    ) : (
                      <span className="muted" aria-hidden="true">
                        —
                      </span>
                    )}
                  </td>
                  <td className="break">{row.title || row.link || row.download_url || '（无标题）'}</td>
                  <td>{formatBytes(row.content_length ?? undefined)}</td>
                  <td>{fetchPreviewStatus(row)}</td>
                  <td>{formatTime(row.published_at) || formatTime(row.created_at)}</td>
                  <td className="actions">
                    {canDownloadFeedItem(row) ? (
                      <button
                        type="button"
                        className="icon-text"
                        disabled={batchLoading || rowLoading === row.id}
                        onClick={() => downloadRow(row)}
                      >
                        <Download size={16} aria-hidden="true" />
                        {rowLoading === row.id ? '提交中…' : feedItemDownloadButtonLabel(row.download_status)}
                      </button>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty">
                    本次拉取没有新条目（可能已全部存在或被过滤）
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function SubscriptionsView() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [subscriptionModal, setSubscriptionModal] = useState<SubscriptionModalTarget | null>(null);
  const [fetchPreview, setFetchPreview] = useState<{ name: string; items: PolledFeedItem[] } | null>(null);
  const [fetchLoadingId, setFetchLoadingId] = useState<number | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  async function load() {
    setSubscriptions(await api.subscriptions());
  }

  useEffect(() => {
    load().catch((err) => setError(messageOf(err)));
  }, []);

  function edit(sub: Subscription) {
    setSubscriptionModal({ mode: 'edit', subscriptionId: sub.id });
  }

  function upsertSubscription(saved: Subscription) {
    setSubscriptions((prev) => {
      const idx = prev.findIndex((s) => s.id === saved.id);
      if (idx < 0) {
        return [saved, ...prev];
      }
      const next = [...prev];
      next[idx] = saved;
      return next;
    });
  }

  async function pullSubscription(sub: Subscription) {
    setFetchLoadingId(sub.id);
    setError('');
    try {
      const { items } = await api.refreshSubscription(sub.id);
      setFetchPreview({ name: sub.name, items });
      setNotice('拉取完成');
      await load();
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setFetchLoadingId(null);
    }
  }

  return (
    <section className="view">
      <Header title="订阅" description="列表展示拉取调度摘要；点「拉取」预览条目，点「编辑」配置地址与过滤规则。" />
      {fetchPreview && (
        <FetchPreviewModal
          subscriptionName={fetchPreview.name}
          initialItems={fetchPreview.items}
          onClose={() => setFetchPreview(null)}
        />
      )}
      {subscriptionModal && (
        <SubscriptionModal
          key={subscriptionModal.mode === 'edit' ? subscriptionModal.subscriptionId : 'create'}
          target={subscriptionModal}
          subscriptions={subscriptions}
          onClose={() => setSubscriptionModal(null)}
          onSuccess={async (saved) => {
            upsertSubscription(saved);
            if (subscriptionModal.mode === 'create') {
              setNotice('订阅已创建，请使用「拉取」或等待定时调度');
            } else {
              setNotice('订阅已更新');
            }
          }}
        />
      )}
      <div className="subscriptions-toolbar">
        <button type="button" className="primary" onClick={() => setSubscriptionModal({ mode: 'create' })}>
          <Plus size={18} aria-hidden="true" />
          新增订阅
        </button>
      </div>
      <Feedback notice={notice} error={error} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>订阅名称</th>
              <th>拉取调度</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {subscriptions.map((sub) => (
              <tr key={sub.id}>
                <td>{sub.name}</td>
                <td className="sub-schedule-cell">{subscriptionScheduleSummary(sub)}</td>
                <td className="actions">
                  <button
                    type="button"
                    className="icon-text"
                    disabled={fetchLoadingId === sub.id}
                    onClick={() => pullSubscription(sub)}
                  >
                    <RefreshCw size={16} className={fetchLoadingId === sub.id ? 'icon-spinning' : undefined} aria-hidden="true" />
                    拉取
                  </button>
                  <button className="icon-text" onClick={() => edit(sub)}>
                    <SquarePen size={16} />
                    编辑
                  </button>
                  <button className="danger" onClick={() => api.deleteSubscription(sub.id).then(load).catch((err) => setError(messageOf(err)))}>
                    <Trash2 size={16} />
                    删除
                  </button>
                </td>
              </tr>
            ))}
            {subscriptions.length === 0 && <EmptyRow columns={3} text="暂无订阅" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function ItemsView() {
  const [items, setItems] = useState<FeedItem[]>([]);
  const [rowLoading, setRowLoading] = useState<number | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    api.items().then(setItems).catch((err) => setError(messageOf(err)));
  }, []);

  async function downloadItem(item: FeedItem) {
    setError('');
    setNotice('');
    setRowLoading(item.id);
    try {
      const updated = await api.downloadFeedItem(item.id);
      setItems((prev) => prev.map((row) => (row.id === updated.id ? updated : row)));
      setNotice('下载任务已提交');
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setRowLoading(null);
    }
  }

  return (
    <section className="view">
      <Header title="条目" description="已入库 RSS 条目；可手动下载或重新下载（已提交 aria2 的条目亦可再次提交）。" />
      <Feedback notice={notice} error={error} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>标题</th>
              <th>下载地址</th>
              <th>状态</th>
              <th>发布时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td className="break">{item.title || item.link || item.download_url}</td>
                <td className="break">{item.download_url || '无可下载地址'}</td>
                <td><Status value={item.download_status} /></td>
                <td>{formatTime(item.published_at) || formatTime(item.created_at)}</td>
                <td className="actions">
                  {canDownloadFeedItem(item) ? (
                    <button
                      type="button"
                      className="icon-text"
                      disabled={rowLoading === item.id}
                      onClick={() => downloadItem(item)}
                    >
                      <Download size={16} aria-hidden="true" />
                      {rowLoading === item.id ? '提交中…' : feedItemDownloadButtonLabel(item.download_status)}
                    </button>
                  ) : (
                    <span className="muted">—</span>
                  )}
                </td>
              </tr>
            ))}
            {items.length === 0 && <EmptyRow columns={5} text="暂无条目" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function DownloadsView() {
  const [downloads, setDownloads] = useState<DownloadTask[]>([]);
  const [error, setError] = useState('');
  useEffect(() => {
    api.downloads().then(setDownloads).catch((err) => setError(messageOf(err)));
  }, []);
  return (
    <section className="view">
      <Header title="下载" description="展示提交给外部 aria2 的任务结果。" />
      <Feedback error={error} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>任务</th>
              <th>目录</th>
              <th>状态</th>
              <th>aria2 GID</th>
              <th>错误</th>
            </tr>
          </thead>
          <tbody>
            {downloads.map((task) => (
              <tr key={task.id}>
                <td className="break">{task.url}</td>
                <td className="break">{task.dir}</td>
                <td><Status value={task.status} /></td>
                <td>{task.aria2_gid || '-'}</td>
                <td className="break">{task.error || '-'}</td>
              </tr>
            ))}
            {downloads.length === 0 && <EmptyRow columns={5} text="暂无下载任务" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function SettingsView({ user, setUser }: { user: User; setUser: (user: User | null) => void }) {
  const [proxyURL, setProxyURL] = useState('');
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');
  const feishuLabel = useMemo(() => (user.feishu_bound ? user.feishu_name || user.feishu_open_id || '已绑定' : '未绑定'), [user]);

  useEffect(() => {
    api.proxy().then((data) => setProxyURL(data.proxy_url)).catch((err) => setError(messageOf(err)));
  }, []);

  async function saveProxy(event: FormEvent) {
    event.preventDefault();
    setNotice('');
    setError('');
    try {
      const saved = await api.saveProxy(proxyURL);
      setProxyURL(saved.proxy_url);
      setNotice('代理设置已保存');
    } catch (err) {
      setError(messageOf(err));
    }
  }

  async function unbind() {
    setNotice('');
    setError('');
    try {
      await api.unbindFeishu();
      const fresh = await api.me();
      setUser(fresh);
      setNotice('飞书账号已解绑');
    } catch (err) {
      setError(messageOf(err));
    }
  }

  return (
    <section className="view">
      <Header title="设置" description="代理只用于拉取 RSS 内容，不参与 aria2 RPC 或实际下载。" />
      <div className="settings-grid">
        <form className="settings-panel" onSubmit={saveProxy}>
          <h2>全局代理</h2>
          <label>
            HTTP/HTTPS 代理地址
            <input value={proxyURL} onChange={(event) => setProxyURL(event.target.value)} placeholder="http://user:pass@127.0.0.1:7890" />
          </label>
          <button className="primary">保存代理</button>
        </form>
        <div className="settings-panel">
          <h2>飞书备用登录</h2>
          <p className="muted">当前状态：{feishuLabel}</p>
          <div className="horizontal-actions">
            <a className="primary-link" href="/api/auth/feishu/start">
              <ShieldCheck size={16} />
              绑定飞书
            </a>
            {user.feishu_bound && (
              <button className="ghost" onClick={unbind}>
                解绑
              </button>
            )}
          </div>
        </div>
      </div>
      <Feedback notice={notice} error={error} />
    </section>
  );
}

function Header({ title, description }: { title: string; description: string }) {
  return (
    <header className="view-header">
      <h1>{title}</h1>
      <p>{description}</p>
    </header>
  );
}

function Feedback({ notice, error }: { notice?: string; error?: string }) {
  return (
    <>
      {notice && <p className="notice">{notice}</p>}
      {error && <p className="error">{error}</p>}
    </>
  );
}

function EmptyRow({ columns, text }: { columns: number; text: string }) {
  return (
    <tr>
      <td colSpan={columns} className="empty">{text}</td>
    </tr>
  );
}

function Status({ value }: { value: string }) {
  return <span className={`status status-${value}`}>{value}</span>;
}

function formatTime(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('zh-CN', { hour12: false });
}

function messageOf(err: unknown) {
  return err instanceof Error ? err.message : '请求失败';
}

