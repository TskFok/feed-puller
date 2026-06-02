import type { AIConfig } from './types';

export type AIConfigDraft = Omit<AIConfig, 'id' | 'created_at' | 'updated_at'>;

export type AIProviderPreset = {
  id: string;
  label: string;
  description: string;
  baseUrl: string;
  model: string;
  requestOptions: string;
  apiKeyRequired: boolean;
  apiKeyPlaceholder: string;
};

export const AI_PROVIDER_PRESETS: ReadonlyArray<AIProviderPreset> = [
  {
    id: 'openai',
    label: 'OpenAI',
    description: '官方 OpenAI API',
    baseUrl: 'https://api.openai.com/v1',
    model: 'gpt-4o-mini',
    requestOptions: '',
    apiKeyRequired: true,
    apiKeyPlaceholder: 'sk-...'
  },
  {
    id: 'deepseek',
    label: 'DeepSeek',
    description: 'DeepSeek OpenAI 兼容接口',
    baseUrl: 'https://api.deepseek.com/v1',
    model: 'deepseek-chat',
    requestOptions: '',
    apiKeyRequired: true,
    apiKeyPlaceholder: 'sk-...'
  },
  {
    id: 'kimi',
    label: 'Kimi',
    description: 'Kimi 开放平台 OpenAI 兼容接口，使用 platform.kimi.com 申请的 API Key',
    baseUrl: 'https://api.moonshot.cn/v1',
    model: 'kimi-k2.6',
    requestOptions: '{\n  "thinking": {\n    "type": "disabled"\n  }\n}',
    apiKeyRequired: true,
    apiKeyPlaceholder: '在 Kimi 开放平台申请的 API Key'
  },
  {
    id: 'ollama',
    label: 'Ollama',
    description: '本地 Ollama 服务，通常无需 API Key',
    baseUrl: 'http://localhost:11434/v1',
    model: 'llama3.2',
    requestOptions: '',
    apiKeyRequired: false,
    apiKeyPlaceholder: '可选，本地一般留空'
  },
  {
    id: 'custom',
    label: '自定义',
    description: '手动填写地址与模型',
    baseUrl: '',
    model: '',
    requestOptions: '',
    apiKeyRequired: true,
    apiKeyPlaceholder: 'sk-...'
  }
];

export function findAIProviderPreset(providerId: string | null | undefined) {
  return AI_PROVIDER_PRESETS.find((item) => item.id === providerId) ?? null;
}

export function normalizeAIConfigBaseUrl(url: string) {
  return url.trim().replace(/\/+$/, '');
}

export function inferAIProviderId(draft: Pick<AIConfigDraft, 'url' | 'model'>): string {
  const url = normalizeAIConfigBaseUrl(draft.url);
  const model = draft.model.trim();

  for (const preset of AI_PROVIDER_PRESETS) {
    if (preset.id === 'custom') {
      continue;
    }
    if (normalizeAIConfigBaseUrl(preset.baseUrl) === url && preset.model === model) {
      return preset.id;
    }
  }

  for (const preset of AI_PROVIDER_PRESETS) {
    if (preset.id === 'custom') {
      continue;
    }
    if (normalizeAIConfigBaseUrl(preset.baseUrl) === url) {
      return preset.id;
    }
  }

  if (url || model) {
    return 'custom';
  }

  return 'openai';
}

function resolveNameAfterPreset(currentName: string, preset: AIProviderPreset) {
  const trimmed = currentName.trim();
  if (!trimmed) {
    return preset.label;
  }
  if (AI_PROVIDER_PRESETS.some((item) => item.label === trimmed)) {
    return preset.label;
  }
  return currentName;
}

export function applyAIProviderPreset(current: AIConfigDraft, preset: AIProviderPreset): AIConfigDraft {
  if (preset.id === 'custom') {
    return current;
  }

  return {
    ...current,
    name: resolveNameAfterPreset(current.name, preset),
    url: preset.baseUrl,
    model: preset.model,
    request_options: preset.requestOptions
  };
}

export function isAIConfigApiKeyRequired(providerId: string | null | undefined) {
  return findAIProviderPreset(providerId)?.apiKeyRequired ?? true;
}
