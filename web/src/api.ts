import type {
  DownloadTask,
  FeedItem,
  PolledFeedItem,
  BatchDownloadResult,
  BatchStatusResult,
  PollSchedulePreviewInput,
  PollSchedulePreviewResult,
  Subscription,
  User
} from './types';

type RequestOptions = RequestInit & { json?: unknown };

function asArray<T>(data: unknown): T[] {
  return Array.isArray(data) ? data : [];
}

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
  subscriptions: async () => asArray<Subscription>(await request<Subscription[]>('/api/subscriptions')),
  createSubscription: (payload: Omit<Subscription, 'id'>) =>
    request<Subscription>('/api/subscriptions', { method: 'POST', json: payload }),
  updateSubscription: (id: number, payload: Omit<Subscription, 'id'>) =>
    request<Subscription>(`/api/subscriptions/${id}`, { method: 'PUT', json: payload }),
  deleteSubscription: (id: number) => request<{ ok: boolean }>(`/api/subscriptions/${id}`, { method: 'DELETE' }),
  reorderSubscriptions: (subscriptionIds: number[]) =>
    request<{ ok: boolean }>('/api/subscriptions/reorder', { method: 'PUT', json: { subscription_ids: subscriptionIds } }),
  refreshSubscription: (id: number) =>
    request<{ items: PolledFeedItem[] }>(`/api/subscriptions/${id}/refresh`, { method: 'POST' }),
  previewNextPoll: (payload: PollSchedulePreviewInput) =>
    request<PollSchedulePreviewResult>('/api/subscriptions/preview-next-poll', { method: 'POST', json: payload }),
  downloadFeedItem: (id: number) => request<FeedItem>(`/api/items/${id}/download`, { method: 'POST' }),
  batchDownloadFeedItems: (itemIds: number[]) =>
    request<BatchDownloadResult>('/api/items/batch-download', { method: 'POST', json: { item_ids: itemIds } }),
  batchUpdateFeedItemStatus: (itemIds: number[], downloadStatus: 'pending' | 'submitted') =>
    request<BatchStatusResult>('/api/items/batch-status', {
      method: 'POST',
      json: { item_ids: itemIds, download_status: downloadStatus }
    }),
  items: async (subscriptionId?: number) =>
    asArray<FeedItem>(
      await request<FeedItem[]>(subscriptionId ? `/api/items?subscription_id=${subscriptionId}` : '/api/items')
    ),
  downloads: async () => asArray<DownloadTask>(await request<DownloadTask[]>('/api/downloads')),
  proxy: () => request<{ proxy_url: string }>('/api/settings/proxy'),
  saveProxy: (proxy_url: string) =>
    request<{ proxy_url: string }>('/api/settings/proxy', { method: 'PUT', json: { proxy_url } }),
  feishuBinding: () => request<{ bound: boolean; feishu_name?: string; feishu_open_id?: string }>('/api/settings/feishu-binding'),
  getFeishuLoginUrl: () => request<{ url: string; goto: string }>('/api/auth/feishu/login-url'),
  getFeishuBindUrl: () => request<{ url: string; goto?: string }>('/api/settings/feishu-bind-url'),
  unbindFeishu: () => request<{ ok: boolean }>('/api/settings/feishu-binding', { method: 'DELETE' })
};

