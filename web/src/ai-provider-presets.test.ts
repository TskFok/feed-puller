import { describe, expect, it } from 'vitest';
import {
  AI_PROVIDER_PRESETS,
  applyAIProviderPreset,
  findAIProviderPreset,
  inferAIProviderId,
  isAIConfigApiKeyRequired,
  normalizeAIConfigBaseUrl
} from './ai-provider-presets';

const emptyDraft = {
  name: '',
  url: '',
  model: '',
  api_key: '',
  request_options: ''
};

describe('ai provider presets', () => {
  it('applies openai preset values', () => {
    const next = applyAIProviderPreset(emptyDraft, findAIProviderPreset('openai')!);
    expect(next.url).toBe('https://api.openai.com/v1');
    expect(next.model).toBe('gpt-4o-mini');
    expect(next.name).toBe('OpenAI');
  });

  it('applies ollama preset without overwriting custom name', () => {
    const next = applyAIProviderPreset(
      { ...emptyDraft, name: '家里 Ollama' },
      findAIProviderPreset('ollama')!
    );
    expect(next.url).toBe('http://localhost:11434/v1');
    expect(next.name).toBe('家里 Ollama');
    expect(isAIConfigApiKeyRequired('ollama')).toBe(false);
  });

  it('applies kimi preset values', () => {
    const next = applyAIProviderPreset(emptyDraft, findAIProviderPreset('kimi')!);
    expect(next.url).toBe('https://api.moonshot.cn/v1');
    expect(next.model).toBe('kimi-k2.6');
    expect(next.name).toBe('Kimi');
    expect(JSON.parse(next.request_options)).toEqual({ thinking: { type: 'disabled' } });
  });

  it('custom preset keeps current draft', () => {
    const current = {
      name: 'My API',
      url: 'https://example.com/v1',
      model: 'demo',
      api_key: 'secret',
      request_options: '{"temperature":0.6}'
    };
    const next = applyAIProviderPreset(current, findAIProviderPreset('custom')!);
    expect(next).toEqual(current);
  });

  it('infers provider from saved config', () => {
    expect(
      inferAIProviderId({
        url: 'https://api.deepseek.com/v1/',
        model: 'deepseek-chat'
      })
    ).toBe('deepseek');
    expect(
      inferAIProviderId({
        url: 'https://custom.example.com/v1',
        model: 'foo'
      })
    ).toBe('custom');
  });

  it('normalizes trailing slash in base url', () => {
    expect(normalizeAIConfigBaseUrl('https://api.openai.com/v1/')).toBe('https://api.openai.com/v1');
  });

  it('exposes all expected presets', () => {
    expect(AI_PROVIDER_PRESETS.map((item) => item.id)).toEqual([
      'openai',
      'deepseek',
      'kimi',
      'ollama',
      'custom'
    ]);
  });
});
