import { FormEvent, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  Download,
  GripVertical,
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
import { useFeishuQR } from './feishu-qr';
import type {
  FeedItem,
  PolledFeedItem,
  PollSchedulePreviewInput,
  Subscription,
  User
} from './types';

type Tab = 'subscriptions' | 'settings';

const IANA_TIMEZONE_GROUPS = [
  {
    label: '常用',
    options: [{ value: 'UTC', label: 'UTC（协调世界时）' }]
  },
  {
    label: '中国',
    options: [{ value: 'Asia/Shanghai', label: '上海 (UTC+8)' }]
  },
  {
    label: '日本',
    options: [{ value: 'Asia/Tokyo', label: '东京 (UTC+9)' }]
  },
  {
    label: '美国',
    options: [
      { value: 'America/New_York', label: '东部 · 纽约' },
      { value: 'America/Chicago', label: '中部 · 芝加哥' },
      { value: 'America/Denver', label: '山地 · 丹佛' },
      { value: 'America/Los_Angeles', label: '太平洋 · 洛杉矶' }
    ]
  }
] as const;

const KNOWN_IANA_TIMEZONES = new Set<string>(
  IANA_TIMEZONE_GROUPS.flatMap((group) => group.options.map((opt) => opt.value))
);

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
  use_proxy: false,
  rss_parser: 'generic'
};

const RSS_PARSER_OPTIONS = [
  { value: 'generic', label: '通用 (RSS/Atom)' },
  { value: 'mikan', label: '蜜柑计划 (Mikan)' }
] as const;

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
  const [mode, setMode] = useState<'password' | 'feishu'>('password');
  const [feishuGoto, setFeishuGoto] = useState<string | null>(null);

  useEffect(() => {
    if (mode === 'feishu') {
      setError('');
      api
        .getFeishuLoginUrl()
        .then((data) => setFeishuGoto(data.goto ?? null))
        .catch(() => setError('获取飞书登录地址失败'));
    } else {
      setFeishuGoto(null);
    }
  }, [mode, setError]);

  const handleFeishuLoginSuccess = useCallback(
    (user: unknown) => {
      onLogin(user as User);
    },
    [onLogin]
  );

  useFeishuQR({
    authUrl: feishuGoto,
    mode: 'login',
    qrContainerId: 'feishuLoginQRContainer',
    iframeContainerId: 'feishuLoginIframeContainer',
    onLoginSuccess: handleFeishuLoginSuccess,
    onError: setError
  });

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
        <div className="login-tabs">
          <button type="button" className={mode === 'password' ? 'active' : ''} onClick={() => setMode('password')}>
            账号密码登录
          </button>
          <button type="button" className={mode === 'feishu' ? 'active' : ''} onClick={() => setMode('feishu')}>
            飞书登录
          </button>
        </div>
        {mode === 'password' ? (
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
          </form>
        ) : (
          <div className="form auth-form-feishu">
            {error && <p className="error">{error}</p>}
            {feishuGoto == null && !error && <p className="feishu-qr-hint">正在加载飞书扫码...</p>}
            {feishuGoto != null && (
              <>
                <div id="feishuLoginIframeContainer" className="feishu-iframe-host" aria-hidden />
                <div id="feishuLoginQRContainer" className="feishu-qr-inline" />
                <p className="feishu-qr-hint">使用飞书 App 扫码即可登录</p>
              </>
            )}
          </div>
        )}
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
    use_proxy: sub.use_proxy,
    rss_parser: sub.rss_parser?.trim() || 'generic'
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
            <label className="modal-full">
              RSS 解析器
              <select
                className="form-select"
                value={draft.rss_parser}
                onChange={(event) => setDraft({ ...draft, rss_parser: event.target.value })}
              >
                {RSS_PARSER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
            {draft.rss_parser === 'mikan' && (
              <p className="muted modal-hint">
                蜜柑解析器会将 enclosure 中的 .torrent 链接转换为 magnet 后再提交 aria2，与 NaStool 等工具行为更接近。
              </p>
            )}
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
                  <select
                    className="form-select"
                    aria-label="Crontab 时区（IANA）"
                    value={draft.poll_cron_timezone}
                    onChange={(event) =>
                      setDraft({ ...draft, poll_cron_timezone: event.target.value })
                    }
                  >
                    {!KNOWN_IANA_TIMEZONES.has(draft.poll_cron_timezone.trim() || 'UTC') && (
                      <option value={draft.poll_cron_timezone.trim() || 'UTC'}>
                        {draft.poll_cron_timezone.trim() || 'UTC'}（已保存）
                      </option>
                    )}
                    {IANA_TIMEZONE_GROUPS.map((group) => (
                      <optgroup key={group.label} label={group.label}>
                        {group.options.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </optgroup>
                    ))}
                  </select>
                </label>
              </>
            )}
            {scheduleKind === 'cron' && (
              <p className="muted modal-schedule-help">
                Crontab 在五字段（分 时 日 月 周）下按所选 IANA 时区解释字段含义。表达式内不要使用
                TZ=/CRON_TZ=。支持 @hourly 等别名。调度器每分钟检查一次。
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

function canSelectFeedItem(row: FeedItemDownloadRow): boolean {
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
  const [statusLoading, setStatusLoading] = useState(false);
  const [notice, setNotice] = useTransientNotice();
  const [error, setError] = useState('');

  const selectableRows = useMemo(() => rows.filter((row) => canSelectFeedItem(row)), [rows]);
  const selectableIds = useMemo(() => selectableRows.map((row) => row.id), [selectableRows]);
  const downloadableRows = useMemo(() => rows.filter((row) => canDownloadFeedItem(row)), [rows]);
  const downloadableIds = useMemo(() => downloadableRows.map((row) => row.id), [downloadableRows]);
  const selectedCount = useMemo(() => selectableIds.filter((id) => selected.has(id)).length, [selectableIds, selected]);
  const selectedDownloadableCount = useMemo(
    () => downloadableIds.filter((id) => selected.has(id)).length,
    [downloadableIds, selected]
  );
  const allSelectableSelected = selectableIds.length > 0 && selectedCount === selectableIds.length;
  const batchBusy = batchLoading || statusLoading;

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

  function toggleSelectAll() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allSelectableSelected) {
        for (const id of selectableIds) next.delete(id);
      } else {
        for (const id of selectableIds) next.add(id);
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

  async function updateSelectedStatus(downloadStatus: 'pending' | 'submitted') {
    const ids = selectableIds.filter((id) => selected.has(id));
    if (ids.length === 0) return;
    setError('');
    setNotice('');
    setStatusLoading(true);
    try {
      const result = await api.batchUpdateFeedItemStatus(ids, downloadStatus);
      applyUpdatedItems(result.items);
      const label = downloadStatus === 'pending' ? '未处理' : '已处理';
      setNotice(`已将 ${result.items.length} 条标记为${label}`);
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setStatusLoading(false);
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
              勾选条目后可批量下载或批量修改状态；单条亦可点击操作列。已处理条目可重新下载，提交中的条目无法勾选。
            </p>
          </div>
          <button type="button" className="modal-close ghost" aria-label="关闭拉取结果" onClick={onClose}>
            <X size={20} aria-hidden="true" />
          </button>
        </div>
        {notice && <p className="notice modal-notice">{notice}</p>}
        {error && <p className="error modal-error">{error}</p>}
        {selectableIds.length > 0 && (
          <div className="fetch-preview-toolbar">
            <label className="check fetch-preview-select-all">
              <input
                id={selectAllId}
                type="checkbox"
                checked={allSelectableSelected}
                disabled={batchBusy || rowLoading !== null}
                onChange={toggleSelectAll}
              />
              全选（{selectableIds.length}）
            </label>
            <div className="fetch-preview-toolbar-actions">
              <button
                type="button"
                className="ghost"
                disabled={batchBusy || rowLoading !== null || selectedCount === 0}
                onClick={() => updateSelectedStatus('submitted')}
              >
                {statusLoading ? '更新中…' : `标记已处理（${selectedCount}）`}
              </button>
              <button
                type="button"
                className="ghost"
                disabled={batchBusy || rowLoading !== null || selectedCount === 0}
                onClick={() => updateSelectedStatus('pending')}
              >
                {statusLoading ? '更新中…' : `标记未处理（${selectedCount}）`}
              </button>
              {downloadableIds.length > 0 && (
                <button
                  type="button"
                  className="primary"
                  disabled={batchBusy || rowLoading !== null || selectedDownloadableCount === 0}
                  onClick={downloadSelected}
                >
                  <Download size={16} aria-hidden="true" />
                  {batchLoading ? '提交中…' : `批量下载（${selectedDownloadableCount}）`}
                </button>
              )}
            </div>
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
                    {canSelectFeedItem(row) ? (
                      <input
                        type="checkbox"
                        aria-label={`选择 ${row.title || row.link || '条目'}`}
                        checked={selected.has(row.id)}
                        disabled={batchBusy || rowLoading !== null}
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
                        disabled={batchBusy || rowLoading === row.id}
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

function moveListItem<T>(list: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) {
    return list;
  }
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

function SubscriptionsView() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [subscriptionModal, setSubscriptionModal] = useState<SubscriptionModalTarget | null>(null);
  const [fetchPreview, setFetchPreview] = useState<{ name: string; items: PolledFeedItem[] } | null>(null);
  const [fetchLoadingId, setFetchLoadingId] = useState<number | null>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [reorderSaving, setReorderSaving] = useState(false);
  const [notice, setNotice] = useTransientNotice();
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

  async function commitReorder(from: number, to: number) {
    if (from === to) return;
    const previous = subscriptions;
    const next = moveListItem(subscriptions, from, to);
    setSubscriptions(next);
    setReorderSaving(true);
    setError('');
    try {
      await api.reorderSubscriptions(next.map((sub) => sub.id));
      setNotice('订阅顺序已保存');
    } catch (err) {
      setSubscriptions(previous);
      setError(messageOf(err));
    } finally {
      setReorderSaving(false);
    }
  }

  const rowBusy = fetchLoadingId !== null || reorderSaving;

  return (
    <section className="view">
      <Header title="订阅" description="拖动左侧手柄可调整顺序；点「拉取」预览条目，点「编辑」配置地址与过滤规则。" />
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
              <th className="sub-drag-col" scope="col">
                <span className="sr-only">排序</span>
              </th>
              <th>订阅名称</th>
              <th>拉取调度</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {subscriptions.map((sub, index) => (
              <tr
                key={sub.id}
                className={dragOverIndex === index && dragIndex !== null && dragIndex !== index ? 'sub-row-drag-over' : undefined}
                onDragOver={(event) => {
                  event.preventDefault();
                  if (dragIndex !== null && dragIndex !== index) {
                    setDragOverIndex(index);
                  }
                }}
                onDragLeave={() => {
                  if (dragOverIndex === index) setDragOverIndex(null);
                }}
                onDrop={(event) => {
                  event.preventDefault();
                  if (dragIndex !== null) {
                    void commitReorder(dragIndex, index);
                  }
                  setDragIndex(null);
                  setDragOverIndex(null);
                }}
              >
                <td className="sub-drag-cell">
                  <button
                    type="button"
                    className="sub-drag-handle"
                    draggable={!rowBusy}
                    aria-label={`拖动调整 ${sub.name} 的顺序`}
                    disabled={rowBusy}
                    onDragStart={(event) => {
                      setDragIndex(index);
                      event.dataTransfer.effectAllowed = 'move';
                    }}
                    onDragEnd={() => {
                      setDragIndex(null);
                      setDragOverIndex(null);
                    }}
                  >
                    <GripVertical size={16} aria-hidden="true" />
                  </button>
                </td>
                <td>{sub.name}</td>
                <td className="sub-schedule-cell">{subscriptionScheduleSummary(sub)}</td>
                <td className="actions">
                  <button
                    type="button"
                    className="icon-text"
                    disabled={rowBusy || fetchLoadingId === sub.id}
                    onClick={() => pullSubscription(sub)}
                  >
                    <RefreshCw size={16} className={fetchLoadingId === sub.id ? 'icon-spinning' : undefined} aria-hidden="true" />
                    拉取
                  </button>
                  <button className="icon-text" disabled={rowBusy} onClick={() => edit(sub)}>
                    <SquarePen size={16} />
                    编辑
                  </button>
                  <button
                    className="danger"
                    disabled={rowBusy}
                    onClick={() => api.deleteSubscription(sub.id).then(load).catch((err) => setError(messageOf(err)))}
                  >
                    <Trash2 size={16} />
                    删除
                  </button>
                </td>
              </tr>
            ))}
            {subscriptions.length === 0 && <EmptyRow columns={4} text="暂无订阅" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}


function SettingsView({ user, setUser }: { user: User; setUser: (user: User | null) => void }) {
  const [proxyURL, setProxyURL] = useState('');
  const [notice, setNotice] = useTransientNotice();
  const [error, setError] = useState('');
  const [bindFeishuAuthUrl, setBindFeishuAuthUrl] = useState<string | null>(null);
  const [bindModalOpen, setBindModalOpen] = useState(false);
  const feishuLabel = useMemo(() => (user.feishu_bound ? user.feishu_name || user.feishu_open_id || '已绑定' : '未绑定'), [user]);

  useEffect(() => {
    api.proxy().then((data) => setProxyURL(data.proxy_url)).catch((err) => setError(messageOf(err)));
  }, []);

  const closeBindModal = useCallback(() => {
    setBindModalOpen(false);
    setBindFeishuAuthUrl(null);
  }, []);

  const handleBindSuccess = useCallback(async () => {
    closeBindModal();
    try {
      const fresh = await api.me();
      setUser(fresh);
      setNotice('飞书账号绑定成功');
    } catch (err) {
      setError(messageOf(err));
    }
  }, [closeBindModal, setUser]);

  useFeishuQR({
    authUrl: bindModalOpen ? bindFeishuAuthUrl : null,
    mode: 'bind',
    qrContainerId: 'feishuBindQRContainer',
    iframeContainerId: 'feishuBindIframeContainer',
    onBindSuccess: handleBindSuccess,
    onError: (message) => {
      closeBindModal();
      setError(message);
    }
  });

  useEffect(() => {
    if (!bindModalOpen) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  }, [bindModalOpen]);

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

  async function startBind() {
    setNotice('');
    setError('');
    try {
      const data = await api.getFeishuBindUrl();
      setBindFeishuAuthUrl(data.goto ?? null);
      setBindModalOpen(true);
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

  const bindFeishuModal =
    bindModalOpen && bindFeishuAuthUrl ? (
      <div className="bind-feishu-overlay" onClick={closeBindModal} role="presentation">
        <div className="bind-feishu-modal" onClick={(event) => event.stopPropagation()} role="dialog" aria-labelledby="bind-feishu-title">
          <h3 id="bind-feishu-title" className="bind-feishu-title">
            <span className="bind-feishu-icon" aria-hidden>
              品
            </span>
            绑定飞书
          </h3>
          <p className="bind-feishu-desc">使用飞书 App 扫码，可将飞书账号绑定到当前用户</p>
          <div id="feishuBindIframeContainer" className="feishu-iframe-host" aria-hidden />
          <div id="feishuBindQRContainer" className="feishu-qr-inline bind-feishu-qr-sdk" />
          <button type="button" className="bind-feishu-close" onClick={closeBindModal}>
            关闭
          </button>
        </div>
      </div>
    ) : null;

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
            <button type="button" className="primary-link" onClick={startBind}>
              <ShieldCheck size={16} />
              绑定飞书
            </button>
            {user.feishu_bound && (
              <button className="ghost" onClick={unbind}>
                解绑
              </button>
            )}
          </div>
        </div>
      </div>
      {bindFeishuModal != null ? createPortal(bindFeishuModal, document.body) : null}
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

const NOTICE_DISMISS_MS = 4000;

function useTransientNotice() {
  const [notice, setNoticeState] = useState('');
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearNoticeTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const setNotice = useCallback(
    (message: string) => {
      clearNoticeTimer();
      setNoticeState(message);
      if (message) {
        timerRef.current = setTimeout(() => {
          setNoticeState('');
          timerRef.current = null;
        }, NOTICE_DISMISS_MS);
      }
    },
    [clearNoticeTimer]
  );

  useEffect(() => () => clearNoticeTimer(), [clearNoticeTimer]);

  return [notice, setNotice] as const;
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

