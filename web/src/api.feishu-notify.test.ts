import { describe, expect, it, vi, beforeEach } from 'vitest';
import { api } from './api';

const sampleConfig = {
  feishu_notify_type: 'webhook' as const,
  feishu_bot_webhook: 'https://hook.test',
  feishu_receive_open_id: '',
  feishu_receive_targets: '',
  feishu_complete_title: '[完成]',
  feishu_fail_title: '[失败]',
  feishu_prowlarr_complete_title: '[Prowlarr 完成]',
  feishu_prowlarr_fail_title: '[Prowlarr 失败]',
  feishu_prowlarr_complete_body: '**类型**: {{media_type}}\n**标题**: {{title}}\n**路径**: {{path}}',
  feishu_prowlarr_fail_body: '**类型**: {{media_type}}\n**标题**: {{title}}\n**错误**: {{error}}',
  feishu_include_subscription: true,
  feishu_include_title: true,
  feishu_include_path: true,
  feishu_notify_on_fail: true,
  feishu_use_interactive_card: true,
  feishu_batch_window_seconds: 30,
  configured: true
};

describe('api feishu notify', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path === '/api/settings/feishu-notify' && (!init?.method || init.method === 'GET')) {
          return new Response(JSON.stringify(sampleConfig), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/feishu-notify' && init?.method === 'PUT') {
          const body = JSON.parse(String(init.body));
          expect(body.feishu_notify_type).toBe('api');
          expect(body.feishu_receive_targets).toContain('chat_id:oc_test');
          return new Response(
            JSON.stringify({
              ...sampleConfig,
              feishu_notify_type: 'api',
              feishu_receive_open_id: 'ou_test',
              feishu_receive_targets: 'chat_id:oc_test',
              feishu_batch_window_seconds: 0
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/settings/feishu-notify/test' && init?.method === 'POST') {
          return new Response(JSON.stringify({ message: '测试消息已发送' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({ error: 'unexpected' }), { status: 404 });
      })
    );
  });

  it('loads feishu notify config', async () => {
    const cfg = await api.feishuNotifyConfig();
    expect(cfg.feishu_notify_type).toBe('webhook');
    expect(cfg.feishu_use_interactive_card).toBe(true);
  });

  it('saves feishu notify config', async () => {
    const cfg = await api.saveFeishuNotifyConfig({
      ...sampleConfig,
      feishu_notify_type: 'api',
      feishu_receive_open_id: 'ou_test',
      feishu_receive_targets: 'chat_id:oc_test',
      feishu_batch_window_seconds: 0
    });
    expect(cfg.feishu_receive_open_id).toBe('ou_test');
    expect(cfg.feishu_batch_window_seconds).toBe(0);
  });

  it('sends test notification', async () => {
    const result = await api.testFeishuNotify();
    expect(result.message).toBe('测试消息已发送');
  });
});
