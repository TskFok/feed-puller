import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Download, LogOut, RefreshCw, Rss, Settings, ShieldCheck, SquarePen, Trash2 } from 'lucide-react';
import { api } from './api';
import type { DownloadTask, FeedItem, Subscription, User } from './types';

type Tab = 'subscriptions' | 'items' | 'downloads' | 'settings';

const emptySubscription: Omit<Subscription, 'id'> = {
  name: '',
  feed_url: '',
  enabled: true,
  poll_interval_minutes: 30,
  download_dir: '',
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

function SubscriptionsView() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [form, setForm] = useState(emptySubscription);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  async function load() {
    setSubscriptions(await api.subscriptions());
  }

  useEffect(() => {
    load().catch((err) => setError(messageOf(err)));
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setNotice('');
    setError('');
    try {
      if (editingID) {
        await api.updateSubscription(editingID, form);
        setNotice('订阅已更新');
      } else {
        await api.createSubscription(form);
        setNotice('订阅已创建，后台会立即拉取一次');
      }
      setForm(emptySubscription);
      setEditingID(null);
      await load();
    } catch (err) {
      setError(messageOf(err));
    }
  }

  function edit(sub: Subscription) {
    setEditingID(sub.id);
    setForm({
      name: sub.name,
      feed_url: sub.feed_url,
      enabled: sub.enabled,
      poll_interval_minutes: sub.poll_interval_minutes,
      download_dir: sub.download_dir,
      use_proxy: sub.use_proxy
    });
  }

  return (
    <section className="view">
      <Header title="订阅" description="按订阅配置拉取间隔、下载目录和是否使用代理。" />
      <form className="toolbar-form" onSubmit={submit}>
        <label>
          名称
          <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
        </label>
        <label className="wide">
          RSS 地址
          <input value={form.feed_url} onChange={(event) => setForm({ ...form, feed_url: event.target.value })} required />
        </label>
        <label>
          下载目录
          <input value={form.download_dir} onChange={(event) => setForm({ ...form, download_dir: event.target.value })} required />
        </label>
        <label>
          间隔分钟
          <input type="number" min={1} value={form.poll_interval_minutes} onChange={(event) => setForm({ ...form, poll_interval_minutes: Number(event.target.value) })} required />
        </label>
        <label className="check">
          <input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />
          启用
        </label>
        <label className="check">
          <input type="checkbox" checked={form.use_proxy} onChange={(event) => setForm({ ...form, use_proxy: event.target.checked })} />
          使用代理拉取
        </label>
        <button className="primary">{editingID ? '保存订阅' : '新增订阅'}</button>
      </form>
      <Feedback notice={notice} error={error} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>RSS</th>
              <th>目录</th>
              <th>间隔</th>
              <th>状态</th>
              <th>上次拉取</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {subscriptions.map((sub) => (
              <tr key={sub.id}>
                <td>{sub.name}</td>
                <td className="break">{sub.feed_url}</td>
                <td className="break">{sub.download_dir}</td>
                <td>{sub.poll_interval_minutes} 分钟</td>
                <td>
                  <Status value={sub.enabled ? (sub.use_proxy ? '代理拉取' : '直连拉取') : '停用'} />
                </td>
                <td>{formatTime(sub.last_fetched_at) || sub.last_error || '尚未拉取'}</td>
                <td className="actions">
                  <button className="icon-text" onClick={() => api.refreshSubscription(sub.id).then(load).catch((err) => setError(messageOf(err)))}>
                    <RefreshCw size={16} />
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
            {subscriptions.length === 0 && <EmptyRow columns={7} text="暂无订阅" />}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function ItemsView() {
  const [items, setItems] = useState<FeedItem[]>([]);
  const [error, setError] = useState('');
  useEffect(() => {
    api.items().then(setItems).catch((err) => setError(messageOf(err)));
  }, []);
  return (
    <section className="view">
      <Header title="条目" description="显示已入库 RSS 条目和是否已进入下载队列。" />
      <Feedback error={error} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>标题</th>
              <th>下载地址</th>
              <th>状态</th>
              <th>发布时间</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td className="break">{item.title || item.link || item.download_url}</td>
                <td className="break">{item.download_url || '无可下载地址'}</td>
                <td><Status value={item.download_status} /></td>
                <td>{formatTime(item.published_at) || formatTime(item.created_at)}</td>
              </tr>
            ))}
            {items.length === 0 && <EmptyRow columns={4} text="暂无条目" />}
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

