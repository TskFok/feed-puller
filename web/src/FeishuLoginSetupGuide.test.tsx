import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { FeishuLoginSetupGuide, feishuSetupIncomplete } from './FeishuLoginSetupGuide';
import type { AuthOptions, User } from './types';

const baseUser: User = {
  id: 1,
  email: 'admin@example.com',
  feishu_bound: false
};

const boundUser: User = {
  ...baseUser,
  feishu_bound: true,
  feishu_name: 'Alice'
};

const bothEnabled: AuthOptions = {
  password_login_enabled: true,
  feishu_login_enabled: true
};

describe('feishuSetupIncomplete', () => {
  it('未完成绑定时返回 true', () => {
    expect(feishuSetupIncomplete(bothEnabled, baseUser)).toBe(true);
  });

  it('已绑定或密码登录已关闭时返回 false', () => {
    expect(feishuSetupIncomplete(bothEnabled, boundUser)).toBe(false);
    expect(
      feishuSetupIncomplete({ password_login_enabled: false, feishu_login_enabled: true }, baseUser)
    ).toBe(false);
  });
});

describe('FeishuLoginSetupGuide', () => {
  it('未绑定飞书时显示三步向导与绑定按钮', () => {
    render(<FeishuLoginSetupGuide user={baseUser} authOptions={bothEnabled} onBind={vi.fn()} />);

    expect(screen.getByRole('heading', { name: '飞书登录迁移向导' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '立即绑定飞书' })).toBeInTheDocument();
    expect(screen.getByText('请先完成飞书绑定后再关闭密码登录。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '复制' })).toBeDisabled();
  });

  it('已绑定且仍启用密码登录时启用第三步复制', async () => {
    const onCopyEnv = vi.fn();
    vi.stubGlobal('navigator', {
      ...navigator,
      clipboard: { writeText: vi.fn(async () => undefined) }
    });

    render(
      <FeishuLoginSetupGuide user={boundUser} authOptions={bothEnabled} onBind={vi.fn()} onCopyEnv={onCopyEnv} />
    );

    expect(screen.getByText('已绑定：Alice')).toBeInTheDocument();
    const copyButton = screen.getByRole('button', { name: '复制' });
    expect(copyButton).not.toBeDisabled();
    fireEvent.click(copyButton);
    await waitFor(() => expect(onCopyEnv).toHaveBeenCalledWith('PASSWORD_LOGIN_ENABLED=false'));
  });

  it('密码登录已关闭且已绑定时显示成功状态', () => {
    render(
      <FeishuLoginSetupGuide
        user={boundUser}
        authOptions={{ password_login_enabled: false, feishu_login_enabled: true }}
        onBind={vi.fn()}
      />
    );

    expect(screen.getByText('已完成迁移，当前仅支持飞书扫码登录。')).toBeInTheDocument();
  });
});
