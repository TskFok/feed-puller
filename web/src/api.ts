import type { DownloadTask, FeedItem, Subscription, User } from './types';

type RequestOptions = RequestInit & { json?: unknown };

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  let body = options.body;
  if (options.json !== undefined) {
    headers.set('Content-Type', 'application/json');
    body = JSON.stringify(options.json);
  }
  const response = await fetch(path, {
    ...options,
    body,
    headers,
    credentials: 'include'
  });
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(data?.error || `请求失败：${response.status}`);
  }
  return data as T;
}

export const api = {
  me: () => request<User>('/api/auth/me'),
  login: (email: string, password: string) =>
    request<User>('/api/auth/login', { method: 'POST', json: { email, password } }),
  logout: () => request<{ ok: boolean }>('/api/auth/logout', { method: 'POST' }),
  subscriptions: () => request<Subscription[]>('/api/subscriptions'),
  createSubscription: (payload: Omit<Subscription, 'id'>) =>
    request<Subscription>('/api/subscriptions', { method: 'POST', json: payload }),
  updateSubscription: (id: number, payload: Omit<Subscription, 'id'>) =>
    request<Subscription>(`/api/subscriptions/${id}`, { method: 'PUT', json: payload }),
  deleteSubscription: (id: number) => request<{ ok: boolean }>(`/api/subscriptions/${id}`, { method: 'DELETE' }),
  refreshSubscription: (id: number) => request<{ ok: boolean }>(`/api/subscriptions/${id}/refresh`, { method: 'POST' }),
  items: (subscriptionId?: number) =>
    request<FeedItem[]>(subscriptionId ? `/api/items?subscription_id=${subscriptionId}` : '/api/items'),
  downloads: () => request<DownloadTask[]>('/api/downloads'),
  proxy: () => request<{ proxy_url: string }>('/api/settings/proxy'),
  saveProxy: (proxy_url: string) =>
    request<{ proxy_url: string }>('/api/settings/proxy', { method: 'PUT', json: { proxy_url } }),
  feishuBinding: () => request<{ bound: boolean; feishu_name?: string; feishu_open_id?: string }>('/api/settings/feishu-binding'),
  unbindFeishu: () => request<{ ok: boolean }>('/api/settings/feishu-binding', { method: 'DELETE' })
};

