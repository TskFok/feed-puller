import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from './api';

describe('api ai config models', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path === '/api/ai-configs/models' && init?.method === 'POST') {
          return new Response(JSON.stringify({ models: ['gpt-4o-mini', 'gpt-4o'] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/ai-configs/2/models' && init?.method === 'POST') {
          return new Response(JSON.stringify({ models: ['deepseek-chat'] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({ error: 'unexpected' }), { status: 404 });
      })
    );
  });

  it('loads models from draft payload', async () => {
    const result = await api.fetchAIConfigModels({
      url: 'https://api.openai.com/v1',
      api_key: 'sk-test'
    });
    expect(result.models).toEqual(['gpt-4o-mini', 'gpt-4o']);
  });

  it('loads models by saved config id', async () => {
    const result = await api.fetchAIConfigModelsById(2);
    expect(result.models).toEqual(['deepseek-chat']);
  });
});
