import { DEFAULT_PAGE_SIZE } from './listPaging';
import type {
  ActiveDownload,
  AIConfig,
  AIConfigTestResult,
  CompletedDownload,
  RenameDownloadResult,
  DownloadTask,
  FeedItem,
  PaginatedResult,
  PolledFeedItem,
  BatchDownloadResult,
  BatchStatusResult,
  PollSchedulePreviewInput,
  PollSchedulePreviewResult,
  Subscription,
  User
} from './types';
import type { PageSizeOption } from './listPaging';

type RequestOptions = RequestInit & { json?: unknown };

function asArray<T>(data: unknown): T[] {
  return Array.isArray(data) ? data : [];
}

function pageQuery(page: number, pageSize: number) {
  return `page=${page}&page_size=${pageSize}`;
}

function normalizePaginated<T>(data: unknown, page: number, pageSize: number): PaginatedResult<T> {
  if (data == null) {
    return { items: [], total: 0, page, page_size: pageSize };
  }
  if (Array.isArray(data)) {
    return { items: data as T[], total: data.length, page: 1, page_size: pageSize };
  }
  const body = data as PaginatedResult<T>;
  return {
    items: asArray<T>(body.items),
    total: body.total ?? 0,
    page: body.page ?? page,
    page_size: body.page_size ?? pageSize
  };
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
  subscriptions: async (page = 1, pageSize: PageSizeOption = DEFAULT_PAGE_SIZE): Promise<PaginatedResult<Subscription>> =>
    normalizePaginated<Subscription>(
      await request<PaginatedResult<Subscription>>(`/api/subscriptions?${pageQuery(page, pageSize)}`),
      page,
      pageSize
    ),
  subscriptionIds: () => request<{ ids: number[] }>('/api/subscriptions/ids'),
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
  items: async (subscriptionId?: number, page = 1, pageSize: PageSizeOption = DEFAULT_PAGE_SIZE) => {
    const base = subscriptionId ? `/api/items?subscription_id=${subscriptionId}&` : '/api/items?';
    return normalizePaginated<FeedItem>(
      await request<PaginatedResult<FeedItem>>(`${base}${pageQuery(page, pageSize)}`),
      page,
      pageSize
    );
  },
  downloads: async () => asArray<DownloadTask>(await request<DownloadTask[]>('/api/downloads')),
  activeDownloads: async (page = 1, pageSize: PageSizeOption = DEFAULT_PAGE_SIZE): Promise<PaginatedResult<ActiveDownload>> =>
    normalizePaginated<ActiveDownload>(
      await request<PaginatedResult<ActiveDownload>>(`/api/downloads/active?${pageQuery(page, pageSize)}`),
      page,
      pageSize
    ),
  completedDownloads: async (page = 1, pageSize: PageSizeOption = DEFAULT_PAGE_SIZE): Promise<PaginatedResult<CompletedDownload>> =>
    normalizePaginated<CompletedDownload>(
      await request<PaginatedResult<CompletedDownload>>(`/api/downloads/completed?${pageQuery(page, pageSize)}`),
      page,
      pageSize
    ),
  retryCompletedDownloadRename: (taskId: number) =>
    request<RenameDownloadResult>(`/api/downloads/${taskId}/rename`, { method: 'POST' }),
  proxy: () => request<{ proxy_url: string }>('/api/settings/proxy'),
  saveProxy: (proxy_url: string) =>
    request<{ proxy_url: string }>('/api/settings/proxy', { method: 'PUT', json: { proxy_url } }),
  feishuBinding: () => request<{ bound: boolean; feishu_name?: string; feishu_open_id?: string }>('/api/settings/feishu-binding'),
  getFeishuLoginUrl: () => request<{ url: string; goto: string }>('/api/auth/feishu/login-url'),
  getFeishuBindUrl: () => request<{ url: string; goto?: string }>('/api/settings/feishu-bind-url'),
  unbindFeishu: () => request<{ ok: boolean }>('/api/settings/feishu-binding', { method: 'DELETE' }),
  aiConfigs: async (page = 1, pageSize: PageSizeOption = DEFAULT_PAGE_SIZE): Promise<PaginatedResult<AIConfig>> =>
    normalizePaginated<AIConfig>(
      await request<PaginatedResult<AIConfig>>(`/api/ai-configs?${pageQuery(page, pageSize)}`),
      page,
      pageSize
    ),
  createAIConfig: (payload: Omit<AIConfig, 'id' | 'created_at' | 'updated_at'>) =>
    request<AIConfig>('/api/ai-configs', { method: 'POST', json: payload }),
  updateAIConfig: (id: number, payload: Omit<AIConfig, 'id' | 'created_at' | 'updated_at'>) =>
    request<AIConfig>(`/api/ai-configs/${id}`, { method: 'PUT', json: payload }),
  deleteAIConfig: (id: number) => request<{ ok: boolean }>(`/api/ai-configs/${id}`, { method: 'DELETE' }),
  testAIConfig: (id: number) => request<AIConfigTestResult>(`/api/ai-configs/${id}/test`, { method: 'POST' })
};

