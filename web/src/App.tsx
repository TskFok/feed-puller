import { FormEvent, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import {
  CheckCircle2,
  Download,
  Loader2,
  GripVertical,
  LogOut,
  Plus,
  RefreshCw,
  Rss,
  Bot,
  Settings,
  Search,
  ShieldCheck,
  Sparkles,
  Copy,
  SquarePen,
  Trash2,
  X
} from 'lucide-react';
import { PaginationBar } from './ListPagination';
import { ToastProvider, useToast } from './Toast';
import { api } from './api';
import { pageOffset, type PageSizeOption } from './listPaging';
import { usePagination } from './usePagination';
import { useServerPagination } from './useServerPagination';
import { useFeishuQR } from './feishu-qr';
import { fetchPreviewAction, isFetchPreviewSelectionLocked, useActionLoading } from './useActionLoading';
import { ProwlarrSearchView } from './ProwlarrSearchView';
import { AnimatedModal } from './AnimatedModal';
import { FeishuLoginSetupGuide, FeishuSetupBanner, feishuSetupIncomplete } from './FeishuLoginSetupGuide';
import { ThemePicker } from './ThemePicker';
import type {
  ActiveDownload,
  AIConfig,
  CompletedDownload,
  FeedItem,
  PolledFeedItem,
  PollSchedulePreviewInput,
  Subscription,
  PaginatedResult,
  User,
  AuthOptions
} from './types';

type Tab = 'subscriptions' | 'prowlarr' | 'active' | 'completed' | 'ai-config' | 'settings';

const APP_TABS: Tab[] = ['subscriptions', 'prowlarr', 'active', 'completed', 'ai-config', 'settings'];

function tabFromHash(hash: string): Tab {
  const id = hash.replace(/^#/, '').trim();
  if (!id) {
    return 'subscriptions';
  }
  return APP_TABS.includes(id as Tab) ? (id as Tab) : 'subscriptions';
}

function hashForTab(tab: Tab): string {
  return tab === 'subscriptions' ? '' : `#${tab}`;
}

function readTabFromLocation(): Tab {
  return tabFromHash(window.location.hash);
}

function writeTabToLocation(tab: Tab) {
  const nextHash = hashForTab(tab);
  const nextUrl = `${window.location.pathname}${window.location.search}${nextHash}`;
  const currentUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (nextUrl !== currentUrl) {
    history.replaceState(null, '', nextUrl);
  }
}

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
  rss_parser: 'generic',
  ai_rename_enabled: false,
  ai_rename_season: 1,
  ai_rename_episode_offset: 0
};

const RSS_PARSER_OPTIONS = [
  { value: 'generic', label: '通用 (RSS/Atom)' },
  { value: 'mikan', label: '蜜柑计划 (Mikan)' }
] as const;

export function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  return (
    <ToastProvider>
      {loading ? (
        <div className="boot">正在加载 feed-puller</div>
      ) : !user ? (
        <LoginView onLogin={setUser} />
      ) : (
        <Shell user={user} setUser={setUser} />
      )}
    </ToastProvider>
  );
}

function LoginView({ onLogin }: { onLogin: (user: User) => void }) {
  const { showToast } = useToast();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [authOptions, setAuthOptions] = useState<AuthOptions | null>(null);
  const [mode, setMode] = useState<'password' | 'feishu'>('password');
  const [feishuGoto, setFeishuGoto] = useState<string | null>(null);

  useEffect(() => {
    api
      .authOptions()
      .then((options) => {
        setAuthOptions(options);
        if (!options.password_login_enabled && options.feishu_login_enabled) {
          setMode('feishu');
        }
      })
      .catch(() => showToast('加载登录选项失败', 'error'));
  }, [showToast]);

  useEffect(() => {
    if (mode === 'feishu') {
      api
        .getFeishuLoginUrl()
        .then((data) => setFeishuGoto(data.goto ?? null))
        .catch(() => showToast('获取飞书登录地址失败', 'error'));
    } else {
      setFeishuGoto(null);
    }
  }, [mode, showToast]);

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
    onError: (message) => showToast(message, 'error')
  });

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    try {
      onLogin(await api.login(email, password));
    } catch (err) {
      showToast(err instanceof Error ? err.message : '登录失败', 'error');
    } finally {
      setSubmitting(false);
    }
  }

  const passwordLoginEnabled = authOptions?.password_login_enabled ?? true;
  const feishuLoginEnabled = authOptions?.feishu_login_enabled ?? true;
  const showLoginTabs = passwordLoginEnabled && feishuLoginEnabled;
  const showLoginMigrationHint = authOptions != null && passwordLoginEnabled && feishuLoginEnabled;

  return (
    <main className="login-screen">
      <section className="login-panel" aria-labelledby="login-title">
        <div>
          <p className="eyebrow">RSS 自动下载</p>
          <h1 id="login-title">feed-puller</h1>
          <p className="muted">登录后管理订阅、代理和 aria2 下载任务。</p>
          {showLoginMigrationHint && (
            <p className="login-migration-hint">
              首次使用？请先用账号密码登录，在「设置 → 飞书登录迁移向导」中绑定飞书后再关闭密码登录。
            </p>
          )}
        </div>
        {authOptions == null ? (
          <p className="muted">正在加载登录选项...</p>
        ) : (
          <>
            {showLoginTabs && (
              <div className="login-tabs">
                {passwordLoginEnabled && (
                  <button type="button" className={mode === 'password' ? 'active' : ''} onClick={() => setMode('password')}>
                    账号密码登录
                  </button>
                )}
                {feishuLoginEnabled && (
                  <button type="button" className={mode === 'feishu' ? 'active' : ''} onClick={() => setMode('feishu')}>
                    飞书登录
                  </button>
                )}
              </div>
            )}
            {mode === 'password' && passwordLoginEnabled ? (
              <form onSubmit={submit} className="form">
                <label>
                  邮箱
                  <input value={email} type="email" autoComplete="email" onChange={(event) => setEmail(event.target.value)} required />
                </label>
                <label>
                  密码
                  <input value={password} type="password" autoComplete="current-password" onChange={(event) => setPassword(event.target.value)} required />
                </label>
                <button className="primary" disabled={submitting}>
                  {submitting ? '登录中' : '登录'}
                </button>
              </form>
            ) : feishuLoginEnabled ? (
              <div className="form auth-form-feishu">
                {feishuGoto == null && <p className="feishu-qr-hint">正在加载飞书扫码...</p>}
                {feishuGoto != null && (
                  <>
                    <div id="feishuLoginIframeContainer" className="feishu-iframe-host" aria-hidden />
                    <div id="feishuLoginQRContainer" className="feishu-qr-inline" />
                    <p className="feishu-qr-hint">使用飞书 App 扫码即可登录</p>
                  </>
                )}
              </div>
            ) : (
              <p className="muted">当前未启用任何登录方式，请联系管理员。</p>
            )}
          </>
        )}
        <ThemePicker variant="compact" />
      </section>
    </main>
  );
}

function Shell({ user, setUser }: { user: User; setUser: (user: User | null) => void }) {
  const [tab, setTab] = useState<Tab>(() => readTabFromLocation());
  const [authOptions, setAuthOptions] = useState<AuthOptions | null>(null);
  const { showToast } = useToast();

  const selectTab = useCallback((next: Tab) => {
    setTab(next);
    writeTabToLocation(next);
  }, []);

  useEffect(() => {
    api
      .authOptions()
      .then(setAuthOptions)
      .catch(() => setAuthOptions(null));
  }, []);

  useEffect(() => {
    const onHashChange = () => setTab(readTabFromLocation());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

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
          <NavButton tab="subscriptions" active={tab} setTab={selectTab} icon={<Rss size={18} />} label="订阅" />
          <NavButton tab="prowlarr" active={tab} setTab={selectTab} icon={<Search size={18} />} label="Prowlarr 搜索" />
          <NavButton tab="active" active={tab} setTab={selectTab} icon={<Loader2 size={18} />} label="下载中" />
          <NavButton tab="completed" active={tab} setTab={selectTab} icon={<CheckCircle2 size={18} />} label="下载完成" />
          <NavButton tab="ai-config" active={tab} setTab={selectTab} icon={<Bot size={18} />} label="AI 配置" />
          <NavButton tab="settings" active={tab} setTab={selectTab} icon={<Settings size={18} />} label="设置" />
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
        {feishuSetupIncomplete(authOptions, user) && <FeishuSetupBanner onGoSettings={() => selectTab('settings')} />}
        <div key={tab} className="view-transition">
          {tab === 'subscriptions' && <SubscriptionsView onGoActive={() => selectTab('active')} />}
          {tab === 'prowlarr' && (
            <ProwlarrSearchView onGoSettings={() => selectTab('settings')} onGoActive={() => selectTab('active')} />
          )}
          {tab === 'active' && <ActiveDownloadsView />}
          {tab === 'completed' && <CompletedDownloadsView />}
          {tab === 'ai-config' && <AIConfigView />}
          {tab === 'settings' && (
            <SettingsView user={user} setUser={setUser} authOptions={authOptions} onCopyEnv={() => showToast('已复制环境变量配置')} />
          )}
        </div>
      </main>
    </div>
  );
}

function NavButton({ tab, active, setTab, icon, label }: { tab: Tab; active: Tab; setTab: (tab: Tab) => void; icon: JSX.Element; label: string }) {
  return (
    <button type="button" className={active === tab ? 'nav-button active' : 'nav-button'} onClick={() => setTab(tab)}>
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
    rss_parser: sub.rss_parser?.trim() || 'generic',
    ai_rename_enabled: sub.ai_rename_enabled ?? false,
    ai_rename_season: sub.ai_rename_season ?? 1,
    ai_rename_episode_offset: sub.ai_rename_episode_offset ?? 0
  };
}

function subscriptionToDraftForCopy(sub: Subscription): Omit<Subscription, 'id'> {
  const draft = subscriptionToDraft(sub);
  const suffix = ' (副本)';
  draft.name = draft.name.endsWith(suffix) ? draft.name : `${draft.name}${suffix}`;
  return draft;
}

type SubscriptionModalTarget =
  | { mode: 'create'; copyFromId?: number }
  | { mode: 'edit'; subscriptionId: number };

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
  const copyFrom =
    target.mode === 'create' && target.copyFromId != null
      ? subscriptions.find((s) => s.id === target.copyFromId)
      : undefined;
  const subscription =
    target.mode === 'edit' ? subscriptions.find((s) => s.id === target.subscriptionId) : undefined;
  const titleId = useId();
  const firstFieldRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft] = useState<Omit<Subscription, 'id'>>(() => {
    if (copyFrom) return subscriptionToDraftForCopy(copyFrom);
    if (isCreate || !subscription) return { ...emptySubscription };
    return subscriptionToDraft(subscription);
  });
  const [scheduleKind, setScheduleKind] = useState<'interval' | 'cron'>(() => {
    const source = copyFrom ?? (isCreate ? undefined : subscription);
    if (!source) return 'interval';
    return source.poll_cron.trim() !== '' ? 'cron' : 'interval';
  });
  const [saving, setSaving] = useState(false);
  const { showToast } = useToast();

  const previewAnchors = useMemo(
    () => ({
      last_fetched_at: subscription?.last_fetched_at,
      created_at: subscription?.created_at
    }),
    [subscription?.last_fetched_at, subscription?.created_at]
  );
  const { nextPollAt, previewError, previewLoading } = useSchedulePreview(draft, scheduleKind, previewAnchors);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
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
      showToast(messageOf(err), 'error');
    } finally {
      setSaving(false);
    }
  }

  if (!isCreate && !subscription) {
    return null;
  }
  if (isCreate && target.copyFromId != null && !copyFrom) {
    return null;
  }

  return (
    <AnimatedModal onClose={onClose} ariaLabelledBy={titleId} initialFocusRef={firstFieldRef}>
        <div className="modal-header-row">
          <div>
            <h2 id={titleId} className="modal-title">
              {isCreate ? '新增订阅' : '编辑订阅'}
            </h2>
            <p className="muted modal-subtitle">
              {isCreate
                ? copyFrom
                  ? `已填入「${copyFrom.name}」的配置，保存后将创建新订阅。`
                  : '保存后不会自动拉取；请在列表点「拉取」或按 Crontab/间隔等待调度。'
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
              <span className="modal-section-title">AI 刮削重命名</span>
            </legend>
            <p className="modal-hint muted">
              下载完成后将文件重命名为 S01E01 格式，便于媒体库刮削。需在「AI 配置」中至少添加一条可用模型。
            </p>
            <label className="check modal-check-inline">
              <input
                type="checkbox"
                checked={draft.ai_rename_enabled}
                onChange={(event) => setDraft({ ...draft, ai_rename_enabled: event.target.checked })}
              />
              启用 AI 重命名
            </label>
            {draft.ai_rename_enabled && (
              <div className="modal-keyword-grid">
                <label>
                  季度
                  <input
                    type="number"
                    min={1}
                    value={draft.ai_rename_season}
                    onChange={(event) =>
                      setDraft({ ...draft, ai_rename_season: Math.max(1, Number(event.target.value) || 1) })
                    }
                    required
                  />
                </label>
                <label>
                  集数偏移
                  <input
                    type="number"
                    value={draft.ai_rename_episode_offset}
                    onChange={(event) =>
                      setDraft({ ...draft, ai_rename_episode_offset: Number(event.target.value) || 0 })
                    }
                  />
                </label>
              </div>
            )}
            {draft.ai_rename_enabled && (
              <p className="muted modal-hint">
                例如季度 1、偏移 2：识别到「02」时将重命名为 S01E04。
              </p>
            )}
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

          <div className="modal-actions">
            <button type="button" className="ghost" disabled={saving} onClick={onClose}>
              取消
            </button>
            <button type="submit" className="primary" disabled={saving}>
              {saving ? (isCreate ? '创建中…' : '保存中…') : isCreate ? '创建订阅' : '保存更改'}
            </button>
          </div>
        </form>
    </AnimatedModal>
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
  if (status === 'completed') return '重新下载';
  return '重新下载';
}

type FetchPreviewStatusFilter = 'all' | 'pending' | 'submitted' | 'failed' | 'submitting' | 'completed' | 'no-download';

const FETCH_PREVIEW_STATUS_FILTER_OPTIONS: { value: FetchPreviewStatusFilter; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'pending', label: '未处理' },
  { value: 'submitted', label: '已处理' },
  { value: 'failed', label: '失败' },
  { value: 'submitting', label: '提交中' },
  { value: 'completed', label: '已完成' },
  { value: 'no-download', label: '无可下载' }
];

function fetchPreviewStatusKey(row: PolledFeedItem): Exclude<FetchPreviewStatusFilter, 'all'> {
  const url = row.download_url?.trim();
  if (!url) return 'no-download';
  if (row.download_status === 'pending') return 'pending';
  if (row.download_status === 'failed') return 'failed';
  if (row.download_status === 'submitting') return 'submitting';
  if (row.download_status === 'completed') return 'completed';
  return 'submitted';
}

function fetchPreviewStatus(row: PolledFeedItem): string {
  const key = fetchPreviewStatusKey(row);
  return FETCH_PREVIEW_STATUS_FILTER_OPTIONS.find((option) => option.value === key)?.label ?? '已处理';
}

function matchesFetchPreviewStatusFilter(row: PolledFeedItem, filter: FetchPreviewStatusFilter): boolean {
  return filter === 'all' || fetchPreviewStatusKey(row) === filter;
}

function FetchPreviewModal({
  subscriptionName,
  initialItems,
  onClose,
  onGoActive
}: {
  subscriptionName: string;
  initialItems: PolledFeedItem[];
  onClose: () => void;
  onGoActive?: () => void;
}) {
  const titleId = useId();
  const selectAllId = useId();
  const statusFilterId = useId();
  const [rows, setRows] = useState<PolledFeedItem[]>(initialItems);
  const [statusFilter, setStatusFilter] = useState<FetchPreviewStatusFilter>('all');
  const [selected, setSelected] = useState<Set<number>>(() => new Set());
  const action = useActionLoading();
  const { showToast } = useToast();
  const selectionLocked = isFetchPreviewSelectionLocked(action.active);

  function showDownloadSubmittedToast(message: string) {
    showToast(message, 'success', onGoActive ? { action: { label: '查看进度', onClick: onGoActive } } : undefined);
  }

  const filteredRows = useMemo(
    () => rows.filter((row) => matchesFetchPreviewStatusFilter(row, statusFilter)),
    [rows, statusFilter]
  );
  const pagination = usePagination(filteredRows.length, [statusFilter]);
  const pagedFilteredRows = useMemo(
    () => pagination.slice(filteredRows),
    [filteredRows, pagination.slice]
  );
  const selectableRows = useMemo(() => filteredRows.filter((row) => canSelectFeedItem(row)), [filteredRows]);
  const selectableIds = useMemo(() => selectableRows.map((row) => row.id), [selectableRows]);
  const downloadableRows = useMemo(() => filteredRows.filter((row) => canDownloadFeedItem(row)), [filteredRows]);
  const downloadableIds = useMemo(() => downloadableRows.map((row) => row.id), [downloadableRows]);
  const selectedCount = useMemo(() => selectableIds.filter((id) => selected.has(id)).length, [selectableIds, selected]);
  const selectedDownloadableCount = useMemo(
    () => downloadableIds.filter((id) => selected.has(id)).length,
    [downloadableIds, selected]
  );
  const allSelectableSelected = selectableIds.length > 0 && selectedCount === selectableIds.length;

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

  async function downloadRow(row: PolledFeedItem) {
    try {
      await action.run(fetchPreviewAction.downloadRow(row.id), async () => {
        const updated = await api.downloadFeedItem(row.id);
        applyUpdatedItems([updated]);
        setSelected((prev) => {
          const next = new Set(prev);
          next.delete(row.id);
          return next;
        });
        showDownloadSubmittedToast('已提交下载');
      });
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function downloadSelected() {
    const ids = downloadableIds.filter((id) => selected.has(id));
    if (ids.length === 0) return;
    try {
      await action.run(fetchPreviewAction.batchDownload, async () => {
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
          showDownloadSubmittedToast(`已提交 ${ok} 条下载任务`);
        }
        if (failed > 0) {
          const detail = result.failures!.map((f) => `#${f.item_id}: ${f.error}`).join('；');
          showToast(`有 ${failed} 条未成功：${detail}`, 'error');
        }
      });
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function updateSelectedStatus(downloadStatus: 'pending' | 'submitted') {
    const ids = selectableIds.filter((id) => selected.has(id));
    if (ids.length === 0) return;
    const actionKey =
      downloadStatus === 'pending' ? fetchPreviewAction.statusPending : fetchPreviewAction.statusSubmitted;
    try {
      await action.run(actionKey, async () => {
        const result = await api.batchUpdateFeedItemStatus(ids, downloadStatus);
        applyUpdatedItems(result.items);
        const label = downloadStatus === 'pending' ? '未处理' : '已处理';
        showToast(`已将 ${result.items.length} 条标记为${label}`);
      });
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  return (
    <AnimatedModal onClose={onClose} ariaLabelledBy={titleId} panelClassName="fetch-preview-modal">
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
        {rows.length > 0 && (
          <div className="fetch-preview-filters">
            <label className="fetch-preview-status-filter" htmlFor={statusFilterId}>
              <span>状态筛选</span>
              <select
                id={statusFilterId}
                className="form-select"
                value={statusFilter}
                disabled={selectionLocked}
                onChange={(event) => setStatusFilter(event.target.value as FetchPreviewStatusFilter)}
              >
                {FETCH_PREVIEW_STATUS_FILTER_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <span className="muted fetch-preview-filter-count">
              显示 {filteredRows.length} / {rows.length} 条
            </span>
          </div>
        )}
        {selectableIds.length > 0 && (
          <div className="fetch-preview-toolbar">
            <label className="check fetch-preview-select-all">
              <input
                id={selectAllId}
                type="checkbox"
                checked={allSelectableSelected}
                disabled={selectionLocked}
                onChange={toggleSelectAll}
              />
              全选（{selectableIds.length}）
            </label>
            <div className="fetch-preview-toolbar-actions">
              <button
                type="button"
                className="ghost"
                disabled={action.isActive(fetchPreviewAction.statusSubmitted) || selectedCount === 0}
                onClick={() => updateSelectedStatus('submitted')}
              >
                {action.isActive(fetchPreviewAction.statusSubmitted)
                  ? '更新中…'
                  : `标记已处理（${selectedCount}）`}
              </button>
              <button
                type="button"
                className="ghost"
                disabled={action.isActive(fetchPreviewAction.statusPending) || selectedCount === 0}
                onClick={() => updateSelectedStatus('pending')}
              >
                {action.isActive(fetchPreviewAction.statusPending)
                  ? '更新中…'
                  : `标记未处理（${selectedCount}）`}
              </button>
              {downloadableIds.length > 0 && (
                <button
                  type="button"
                  className="primary"
                  disabled={action.isActive(fetchPreviewAction.batchDownload) || selectedDownloadableCount === 0}
                  onClick={downloadSelected}
                >
                  <Download size={16} aria-hidden="true" />
                  {action.isActive(fetchPreviewAction.batchDownload)
                    ? '提交中…'
                    : `批量下载（${selectedDownloadableCount}）`}
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
              {pagedFilteredRows.map((row) => (
                <tr key={row.id}>
                  <td className="fetch-preview-col-check">
                    {canSelectFeedItem(row) ? (
                      <input
                        type="checkbox"
                        aria-label={`选择 ${row.title || row.link || '条目'}`}
                        checked={selected.has(row.id)}
                        disabled={
                          selectionLocked || action.isActive(fetchPreviewAction.downloadRow(row.id))
                        }
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
                        disabled={
                          action.isActive(fetchPreviewAction.batchDownload) ||
                          action.isActive(fetchPreviewAction.downloadRow(row.id))
                        }
                        onClick={() => downloadRow(row)}
                      >
                        <Download size={16} aria-hidden="true" />
                        {action.isActive(fetchPreviewAction.downloadRow(row.id))
                          ? '提交中…'
                          : feedItemDownloadButtonLabel(row.download_status)}
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
              {rows.length > 0 && filteredRows.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty">
                    没有符合当前状态筛选的条目
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <PaginationBar
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalItems={pagination.totalItems}
          rangeStart={pagination.rangeStart}
          rangeEnd={pagination.rangeEnd}
          onPageChange={pagination.setPage}
          onPageSizeChange={pagination.setPageSize}
        />
    </AnimatedModal>
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

function formatSpeed(bps: number): string {
  return `${formatBytes(bps)}/s`;
}

function aria2StatusLabel(status: string): string {
  switch (status) {
    case 'active':
      return '下载中';
    case 'waiting':
      return '等待中';
    case 'paused':
      return '已暂停';
    case 'complete':
      return '已完成';
    case 'error':
      return '失败';
    default:
      return status || '未知';
  }
}

function ActiveDownloadsView() {
  const { showToast } = useToast();
  const loadErrorToastedRef = useRef(false);
  const listEmptyRef = useRef(true);
  const loadActive = useCallback(
    (page: number, pageSize: PageSizeOption): Promise<PaginatedResult<ActiveDownload>> =>
      api.activeDownloads(page, pageSize),
    []
  );
  const pagination = useServerPagination<ActiveDownload>(loadActive, {
    onError: (err) => {
      if (listEmptyRef.current && !loadErrorToastedRef.current) {
        showToast(messageOf(err), 'error');
        loadErrorToastedRef.current = true;
      }
    }
  });
  const { items: rows, loading, reload } = pagination;
  listEmptyRef.current = rows.length === 0;

  useEffect(() => {
    const timer = window.setInterval(() => {
      void reload();
    }, 5000);
    return () => window.clearInterval(timer);
  }, [reload]);

  return (
    <section className="view">
      <Header title="下载中" description="展示已提交 aria2 的任务及实时进度，每 5 秒刷新。" />
      {loading && rows.length === 0 ? (
        <p className="muted">正在加载…</p>
      ) : (
        <>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>订阅</th>
                <th>标题</th>
                <th>进度</th>
                <th>速度</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>{row.subscription_name}</td>
                  <td className="break">{row.title || row.url || '（无标题）'}</td>
                  <td>
                    <DownloadProgressCell row={row} />
                  </td>
                  <td className="muted">{row.status_error ? '—' : formatSpeed(row.download_speed)}</td>
                  <td className="muted">
                    {row.status_error ? <span className="inline-error">{row.status_error}</span> : aria2StatusLabel(row.aria2_status)}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && !loading && <EmptyRow columns={5} text="当前没有进行中的下载" />}
            </tbody>
          </table>
        </div>
        <PaginationBar
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalItems={pagination.total}
          rangeStart={pagination.rangeStart}
          rangeEnd={pagination.rangeEnd}
          onPageChange={pagination.setPage}
          onPageSizeChange={pagination.setPageSize}
        />
        </>
      )}
    </section>
  );
}

function activeDownloadProgressPercent(row: ActiveDownload): number | null {
  if (row.status_error) {
    return null;
  }
  if (row.progress_percent != null && Number.isFinite(row.progress_percent)) {
    return Math.min(100, Math.max(0, row.progress_percent));
  }
  if (row.total_length > 0) {
    return Math.min(100, Math.max(0, (row.completed_length / row.total_length) * 100));
  }
  return null;
}

function DownloadProgressCell({ row }: { row: ActiveDownload }) {
  if (row.status_error) {
    return <span className="muted">—</span>;
  }
  const percent = activeDownloadProgressPercent(row);
  const hasTotal = row.total_length > 0;
  const label = hasTotal
    ? `${formatBytes(row.completed_length)} / ${formatBytes(row.total_length)}`
    : formatBytes(row.completed_length);
  const width = percent ?? 0;

  return (
    <div className="download-progress">
      <div className="download-progress-head">
        {percent != null ? (
          <span className="download-progress-percent">{percent.toFixed(1)}%</span>
        ) : (
          <span className="download-progress-percent muted">{aria2StatusLabel(row.aria2_status)}</span>
        )}
        <span className="download-progress-size muted">{label}</span>
      </div>
      <div
        className="download-progress-bar"
        role="progressbar"
        aria-valuenow={width}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={percent != null ? `下载进度 ${percent.toFixed(1)}%` : '下载进度未知'}
      >
        <div className="download-progress-fill" style={{ width: percent != null ? `${width}%` : '0%' }} />
      </div>
    </div>
  );
}

function CompletedDownloadsView() {
  const { showToast } = useToast();
  const [renameBusyId, setRenameBusyId] = useState<number | null>(null);
  const [renameHint, setRenameHint] = useState('');
  const loadErrorToastedRef = useRef(false);
  const listEmptyRef = useRef(true);
  const loadCompleted = useCallback(
    (page: number, pageSize: PageSizeOption): Promise<PaginatedResult<CompletedDownload>> =>
      api.completedDownloads(page, pageSize),
    []
  );
  const pagination = useServerPagination<CompletedDownload>(loadCompleted, {
    onError: (err) => {
      if (listEmptyRef.current && !loadErrorToastedRef.current) {
        showToast(messageOf(err), 'error');
        loadErrorToastedRef.current = true;
      }
    }
  });
  const { items: rows, loading, reload } = pagination;
  listEmptyRef.current = rows.length === 0;

  useEffect(() => {
    const timer = window.setInterval(() => {
      void reload();
    }, 30000);
    return () => window.clearInterval(timer);
  }, [reload]);

  async function retryRename(row: CompletedDownload) {
    setRenameBusyId(row.id);
    setRenameHint('');
    try {
      const result = await api.retryCompletedDownloadRename(row.id);
      setRenameHint(result.message || (result.skipped ? '无需重命名' : '重命名成功'));
      void reload();
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setRenameBusyId(null);
    }
  }

  return (
    <section className="view">
      <Header
        title="下载完成"
        description="aria2 任务完成后会自动出现在此列表；可对已启用 AI 重命名的订阅手动重试刮削重命名。"
      />
      {renameHint && <p className="muted">{renameHint}</p>}
      {loading && rows.length === 0 ? (
        <p className="muted">正在加载…</p>
      ) : (
        <>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>订阅</th>
                <th>标题</th>
                <th>保存目录</th>
                <th>文件路径</th>
                <th>完成时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>{row.subscription_name}</td>
                  <td className="break">{row.title || row.url || '（无标题）'}</td>
                  <td className="break muted">{row.dir}</td>
                  <td className="break muted">{row.final_path?.trim() || '—'}</td>
                  <td>{formatTime(row.completed_at) || '—'}</td>
                  <td>
                    {row.ai_rename_enabled ? (
                      <button
                        type="button"
                        className="ghost"
                        disabled={renameBusyId === row.id}
                        onClick={() => void retryRename(row)}
                      >
                        {renameBusyId === row.id ? '重命名中…' : '重命名'}
                      </button>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && !loading && <EmptyRow columns={6} text="暂无已完成的下载" />}
            </tbody>
          </table>
        </div>
        <PaginationBar
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalItems={pagination.total}
          rangeStart={pagination.rangeStart}
          rangeEnd={pagination.rangeEnd}
          onPageChange={pagination.setPage}
          onPageSizeChange={pagination.setPageSize}
        />
        </>
      )}
    </section>
  );
}

function SubscriptionsView({ onGoActive }: { onGoActive?: () => void }) {
  const [subscriptionModal, setSubscriptionModal] = useState<SubscriptionModalTarget | null>(null);
  const [fetchPreview, setFetchPreview] = useState<{ name: string; items: PolledFeedItem[] } | null>(null);
  const [fetchLoadingId, setFetchLoadingId] = useState<number | null>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [reorderSaving, setReorderSaving] = useState(false);
  const { showToast } = useToast();
  const loadSubscriptions = useCallback(
    (page: number, pageSize: PageSizeOption): Promise<PaginatedResult<Subscription>> =>
      api.subscriptions(page, pageSize),
    []
  );
  const pagination = useServerPagination<Subscription>(loadSubscriptions, {
    onError: (err) => showToast(messageOf(err), 'error')
  });
  const { items: subscriptions, loading, reload } = pagination;

  function edit(sub: Subscription) {
    setSubscriptionModal({ mode: 'edit', subscriptionId: sub.id });
  }

  function copySubscription(sub: Subscription) {
    setSubscriptionModal({ mode: 'create', copyFromId: sub.id });
  }

  async function pullSubscription(sub: Subscription) {
    setFetchLoadingId(sub.id);
    try {
      const { items } = await api.refreshSubscription(sub.id);
      setFetchPreview({ name: sub.name, items });
      showToast('拉取完成');
      await reload();
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setFetchLoadingId(null);
    }
  }

  async function commitReorder(from: number, to: number) {
    if (from === to) return;
    setReorderSaving(true);
    try {
      const { ids } = await api.subscriptionIds();
      const next = moveListItem(ids, from, to);
      await api.reorderSubscriptions(next);
      showToast('订阅顺序已保存');
      await reload();
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setReorderSaving(false);
    }
  }

  const rowBusy = fetchLoadingId !== null || reorderSaving;
  const subscriptionPageOffset = pageOffset(pagination.page, pagination.pageSize);

  return (
    <section className="view">
      <Header title="订阅" description="拖动左侧手柄可调整顺序；点「拉取」预览条目，点「编辑」配置地址与过滤规则，点「复制」基于已有订阅新建。" />
      {fetchPreview && (
        <FetchPreviewModal
          subscriptionName={fetchPreview.name}
          initialItems={fetchPreview.items}
          onClose={() => setFetchPreview(null)}
          onGoActive={() => {
            setFetchPreview(null);
            onGoActive?.();
          }}
        />
      )}
      {subscriptionModal && (
        <SubscriptionModal
          key={
            subscriptionModal.mode === 'edit'
              ? subscriptionModal.subscriptionId
              : subscriptionModal.copyFromId != null
                ? `copy-${subscriptionModal.copyFromId}`
                : 'create'
          }
          target={subscriptionModal}
          subscriptions={subscriptions}
          onClose={() => setSubscriptionModal(null)}
          onSuccess={async () => {
            await reload();
            if (subscriptionModal.mode === 'create') {
              showToast('订阅已创建，请使用「拉取」或等待定时调度');
            } else {
              showToast('订阅已更新');
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
            {subscriptions.map((sub, pageIndex) => {
              const index = subscriptionPageOffset + pageIndex;
              return (
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
                    type="button"
                    className="icon-text"
                    disabled={rowBusy}
                    aria-label={`复制 ${sub.name}`}
                    onClick={() => copySubscription(sub)}
                  >
                    <Copy size={16} aria-hidden="true" />
                    复制
                  </button>
                  <button
                    className="danger"
                    disabled={rowBusy}
                    onClick={() =>
                      api
                        .deleteSubscription(sub.id)
                        .then(() => reload())
                        .catch((err) => showToast(messageOf(err), 'error'))
                    }
                  >
                    <Trash2 size={16} />
                    删除
                  </button>
                </td>
              </tr>
            );
            })}
            {subscriptions.length === 0 && !loading && <EmptyRow columns={4} text="暂无订阅" />}
          </tbody>
        </table>
      </div>
      <PaginationBar
        page={pagination.page}
        pageSize={pagination.pageSize}
        totalPages={pagination.totalPages}
        totalItems={pagination.total}
        rangeStart={pagination.rangeStart}
        rangeEnd={pagination.rangeEnd}
        onPageChange={pagination.setPage}
        onPageSizeChange={pagination.setPageSize}
      />
    </section>
  );
}


const emptyAIConfig: Omit<AIConfig, 'id'> = {
  name: '',
  url: '',
  model: '',
  api_key: ''
};

type AIConfigModalTarget = { mode: 'create' } | { mode: 'edit'; configId: number };

function AIConfigModal({
  target,
  configs,
  onClose,
  onSuccess
}: {
  target: AIConfigModalTarget;
  configs: AIConfig[];
  onClose: () => void;
  onSuccess: (saved: AIConfig) => void | Promise<void>;
}) {
  const isCreate = target.mode === 'create';
  const existing = target.mode === 'edit' ? configs.find((c) => c.id === target.configId) : undefined;
  const titleId = useId();
  const firstFieldRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft] = useState<Omit<AIConfig, 'id'>>(() => {
    if (isCreate || !existing) return { ...emptyAIConfig };
    return {
      name: existing.name,
      url: existing.url,
      model: existing.model,
      api_key: existing.api_key
    };
  });
  const [saving, setSaving] = useState(false);
  const { showToast } = useToast();

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    try {
      const saved =
        target.mode === 'create'
          ? await api.createAIConfig(draft)
          : await api.updateAIConfig(target.configId, draft);
      await onSuccess(saved);
      onClose();
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setSaving(false);
    }
  }

  if (!isCreate && !existing) {
    return null;
  }

  return (
    <AnimatedModal onClose={onClose} ariaLabelledBy={titleId} initialFocusRef={firstFieldRef}>
        <div className="modal-header-row">
          <div>
            <h2 id={titleId} className="modal-title">
              {isCreate ? '新增 AI 配置' : '编辑 AI 配置'}
            </h2>
            <p className="muted modal-subtitle">填写 OpenAI 兼容接口的地址、模型名称与 API Key。</p>
          </div>
          <button type="button" className="modal-close ghost" aria-label="关闭" onClick={onClose}>
            <X size={20} aria-hidden="true" />
          </button>
        </div>
        <form className="subscription-edit-form" onSubmit={submit}>
          <label className="modal-full">
            模型名称
            <input
              ref={firstFieldRef}
              value={draft.name}
              onChange={(event) => setDraft({ ...draft, name: event.target.value })}
              placeholder="例如：DeepSeek 主账号"
              required
            />
          </label>
          <label className="modal-full">
            API 地址
            <input
              value={draft.url}
              onChange={(event) => setDraft({ ...draft, url: event.target.value })}
              placeholder="https://api.openai.com/v1"
              required
              spellCheck={false}
            />
          </label>
          <label className="modal-full">
            模型
            <input
              value={draft.model}
              onChange={(event) => setDraft({ ...draft, model: event.target.value })}
              placeholder="gpt-4o-mini"
              required
              spellCheck={false}
            />
          </label>
          <label className="modal-full">
            API Key
            <input
              value={draft.api_key}
              onChange={(event) => setDraft({ ...draft, api_key: event.target.value })}
              type="password"
              autoComplete="off"
              required
            />
          </label>
          <div className="modal-actions">
            <button type="button" className="ghost" onClick={onClose}>
              取消
            </button>
            <button className="primary" disabled={saving}>
              {saving ? '保存中' : '保存'}
            </button>
          </div>
        </form>
    </AnimatedModal>
  );
}

function AIConfigView() {
  const [modal, setModal] = useState<AIConfigModalTarget | null>(null);
  const { showToast } = useToast();
  const [testingId, setTestingId] = useState<number | null>(null);
  const loadAIConfigs = useCallback(
    (page: number, pageSize: PageSizeOption): Promise<PaginatedResult<AIConfig>> =>
      api.aiConfigs(page, pageSize),
    []
  );
  const pagination = useServerPagination<AIConfig>(loadAIConfigs, {
    onError: (err) => showToast(messageOf(err), 'error')
  });
  const { items: configs, loading, reload } = pagination;

  async function testConfig(cfg: AIConfig) {
    setTestingId(cfg.id);
    try {
      const result = await api.testAIConfig(cfg.id);
      if (result.ok) {
        showToast(result.message || `「${cfg.name}」API 连通正常`);
      } else {
        showToast(result.error || 'API 连通检查失败', 'error');
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setTestingId(null);
    }
  }

  const rowBusy = testingId !== null;

  return (
    <section className="view">
      <Header title="AI 配置" description="管理 OpenAI 兼容的模型接入；列表中可检查 API 是否通畅。" />
      {modal && (
        <AIConfigModal
          key={modal.mode === 'edit' ? modal.configId : 'create'}
          target={modal}
          configs={configs}
          onClose={() => setModal(null)}
          onSuccess={async () => {
            await reload();
            showToast(modal.mode === 'create' ? 'AI 配置已创建' : 'AI 配置已更新');
          }}
        />
      )}
      <div className="subscriptions-toolbar">
        <button type="button" className="primary" onClick={() => setModal({ mode: 'create' })}>
          <Plus size={18} aria-hidden="true" />
          新增配置
        </button>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>模型名称</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {configs.map((cfg) => (
              <tr key={cfg.id}>
                <td>{cfg.name}</td>
                <td className="actions">
                  <button
                    type="button"
                    className="icon-text"
                    disabled={rowBusy || testingId === cfg.id}
                    onClick={() => testConfig(cfg)}
                  >
                    <ShieldCheck size={16} className={testingId === cfg.id ? 'icon-spinning' : undefined} aria-hidden="true" />
                    检查连通
                  </button>
                  <button type="button" className="icon-text" disabled={rowBusy} onClick={() => setModal({ mode: 'edit', configId: cfg.id })}>
                    <SquarePen size={16} />
                    编辑
                  </button>
                  <button
                    className="danger"
                    disabled={rowBusy}
                    onClick={() =>
                      api
                        .deleteAIConfig(cfg.id)
                        .then(() => {
                          void reload();
                          showToast('AI 配置已删除');
                        })
                        .catch((err) => showToast(messageOf(err), 'error'))
                    }
                  >
                    <Trash2 size={16} />
                    删除
                  </button>
                </td>
              </tr>
            ))}
            {configs.length === 0 && !loading && <EmptyRow columns={2} text="暂无 AI 配置" />}
          </tbody>
        </table>
      </div>
      <PaginationBar
        page={pagination.page}
        pageSize={pagination.pageSize}
        totalPages={pagination.totalPages}
        totalItems={pagination.total}
        rangeStart={pagination.rangeStart}
        rangeEnd={pagination.rangeEnd}
        onPageChange={pagination.setPage}
        onPageSizeChange={pagination.setPageSize}
      />
    </section>
  );
}

function AppearancePanel() {
  return (
    <div className="settings-panel">
      <ThemePicker variant="panel" />
    </div>
  );
}

function SettingsView({
  user,
  setUser,
  authOptions,
  onCopyEnv
}: {
  user: User;
  setUser: (user: User | null) => void;
  authOptions: AuthOptions | null;
  onCopyEnv: () => void;
}) {
  const [proxyURL, setProxyURL] = useState('');
  const [prowlarrURL, setProwlarrURL] = useState('');
  const [prowlarrAPIKey, setProwlarrAPIKey] = useState('');
  const [prowlarrDownloadDir, setProwlarrDownloadDir] = useState('');
  const [prowlarrTVDownloadDir, setProwlarrTVDownloadDir] = useState('');
  const [prowlarrMovieRename, setProwlarrMovieRename] = useState(false);
  const [prowlarrTMDBKey, setProwlarrTMDBKey] = useState('');
  const [prowlarrIndexerIDs, setProwlarrIndexerIDs] = useState<number[]>([]);
  const [prowlarrTesting, setProwlarrTesting] = useState(false);
  const { showToast } = useToast();
  const [bindFeishuAuthUrl, setBindFeishuAuthUrl] = useState<string | null>(null);
  const [bindModalOpen, setBindModalOpen] = useState(false);
  const feishuLabel = useMemo(() => (user.feishu_bound ? user.feishu_name || user.feishu_open_id || '已绑定' : '未绑定'), [user]);

  useEffect(() => {
    api.proxy().then((data) => setProxyURL(data.proxy_url)).catch((err) => showToast(messageOf(err), 'error'));
    api.prowlarrConfig()
      .then((data) => {
        setProwlarrURL(data.url);
        setProwlarrAPIKey(data.api_key);
        setProwlarrDownloadDir(data.download_dir);
        setProwlarrTVDownloadDir(data.tv_download_dir);
        setProwlarrMovieRename(data.movie_rename_enabled);
        setProwlarrTMDBKey(data.tmdb_api_key);
        setProwlarrIndexerIDs(data.indexer_ids ?? []);
      })
      .catch((err) => showToast(messageOf(err), 'error'));
  }, [showToast]);

  const closeBindModal = useCallback(() => {
    setBindModalOpen(false);
    setBindFeishuAuthUrl(null);
  }, []);

  const handleBindSuccess = useCallback(async () => {
    closeBindModal();
    try {
      const fresh = await api.me();
      setUser(fresh);
      showToast('飞书账号绑定成功');
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }, [closeBindModal, setUser, showToast]);

  useFeishuQR({
    authUrl: bindModalOpen ? bindFeishuAuthUrl : null,
    mode: 'bind',
    qrContainerId: 'feishuBindQRContainer',
    iframeContainerId: 'feishuBindIframeContainer',
    onBindSuccess: handleBindSuccess,
    onError: (message) => {
      closeBindModal();
      showToast(message, 'error');
    }
  });

  async function saveProxy(event: FormEvent) {
    event.preventDefault();
    try {
      const saved = await api.saveProxy(proxyURL);
      setProxyURL(saved.proxy_url);
      showToast('代理设置已保存');
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function saveProwlarr(event: FormEvent) {
    event.preventDefault();
    try {
      const saved = await api.saveProwlarrConfig({
        url: prowlarrURL,
        api_key: prowlarrAPIKey,
        download_dir: prowlarrDownloadDir,
        tv_download_dir: prowlarrTVDownloadDir,
        movie_rename_enabled: prowlarrMovieRename,
        tmdb_api_key: prowlarrTMDBKey,
        indexer_ids: prowlarrIndexerIDs
      });
      setProwlarrURL(saved.url);
      setProwlarrAPIKey(saved.api_key);
      setProwlarrDownloadDir(saved.download_dir);
      setProwlarrTVDownloadDir(saved.tv_download_dir);
      setProwlarrMovieRename(saved.movie_rename_enabled);
      setProwlarrTMDBKey(saved.tmdb_api_key);
      setProwlarrIndexerIDs(saved.indexer_ids ?? []);
      showToast('Prowlarr 设置已保存');
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function testProwlarrConnection() {
    setProwlarrTesting(true);
    try {
      const result = await api.testProwlarr({ url: prowlarrURL, api_key: prowlarrAPIKey });
      if (result.ok) {
        showToast(result.message || 'Prowlarr 连通正常');
      } else {
        showToast(result.error || '连接失败', 'error');
      }
    } catch (err) {
      showToast(messageOf(err), 'error');
    } finally {
      setProwlarrTesting(false);
    }
  }

  async function startBind() {
    try {
      const data = await api.getFeishuBindUrl();
      setBindFeishuAuthUrl(data.goto ?? null);
      setBindModalOpen(true);
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  async function unbind() {
    try {
      await api.unbindFeishu();
      const fresh = await api.me();
      setUser(fresh);
      showToast('飞书账号已解绑');
    } catch (err) {
      showToast(messageOf(err), 'error');
    }
  }

  const bindFeishuModal =
    bindModalOpen && bindFeishuAuthUrl ? (
      <AnimatedModal onClose={closeBindModal} ariaLabelledBy="bind-feishu-title" panelClassName="bind-feishu-modal">
        <h3 id="bind-feishu-title" className="bind-feishu-title">
          <span className="bind-feishu-icon" aria-hidden>
            <Sparkles size={16} />
          </span>
          绑定飞书
        </h3>
        <p className="bind-feishu-desc">使用飞书 App 扫码，可将飞书账号绑定到当前用户</p>
        <div id="feishuBindIframeContainer" className="feishu-iframe-host" aria-hidden />
        <div id="feishuBindQRContainer" className="feishu-qr-inline bind-feishu-qr-sdk" />
        <button type="button" className="bind-feishu-close" onClick={closeBindModal}>
          关闭
        </button>
      </AnimatedModal>
    ) : null;

  return (
    <section className="view">
      <Header title="设置" description="代理只用于拉取 RSS 内容，不参与 aria2 RPC 或实际下载。" />
      <div className="settings-grid">
        {authOptions != null && (
          <FeishuLoginSetupGuide user={user} authOptions={authOptions} onBind={startBind} onCopyEnv={() => onCopyEnv()} />
        )}
        <AppearancePanel />
        <form className="settings-panel" onSubmit={saveProxy}>
          <h2>全局代理</h2>
          <label>
            HTTP/HTTPS 代理地址
            <input value={proxyURL} onChange={(event) => setProxyURL(event.target.value)} placeholder="http://user:pass@127.0.0.1:7890" />
          </label>
          <button className="primary">保存代理</button>
        </form>
        <form className="settings-panel" onSubmit={saveProwlarr}>
          <h2>Prowlarr 搜索</h2>
          <label>
            Prowlarr 地址
            <input value={prowlarrURL} onChange={(event) => setProwlarrURL(event.target.value)} placeholder="http://127.0.0.1:9696" />
          </label>
          <label>
            API Key
            <input value={prowlarrAPIKey} onChange={(event) => setProwlarrAPIKey(event.target.value)} placeholder="在 Prowlarr 设置中获取" />
          </label>
          <label>
            电影保存目录
            <input
              value={prowlarrDownloadDir}
              onChange={(event) => setProwlarrDownloadDir(event.target.value)}
              placeholder="/data/movies"
            />
          </label>
          <label>
            剧集保存目录
            <input
              value={prowlarrTVDownloadDir}
              onChange={(event) => setProwlarrTVDownloadDir(event.target.value)}
              placeholder="留空则与电影目录相同"
            />
          </label>
          <label>
            TMDB API Key（用于电影/剧集重命名）
            <input value={prowlarrTMDBKey} onChange={(event) => setProwlarrTMDBKey(event.target.value)} placeholder="可选，建议填写" />
          </label>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={prowlarrMovieRename}
              onChange={(event) => setProwlarrMovieRename(event.target.checked)}
            />
            电影下载完成后重命名为「标题 (年份).ext」
          </label>
          <div className="horizontal-actions">
            <button type="button" className="ghost" disabled={prowlarrTesting} onClick={testProwlarrConnection}>
              {prowlarrTesting ? '测试中…' : '测试连接'}
            </button>
            <button className="primary">保存 Prowlarr</button>
          </div>
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
      {bindFeishuModal}
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

