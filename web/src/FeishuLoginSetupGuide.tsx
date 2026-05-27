import { CheckCircle2, Circle, Copy } from 'lucide-react';
import { useState } from 'react';
import type { AuthOptions, User } from './types';

const PASSWORD_LOGIN_DISABLE_ENV = 'PASSWORD_LOGIN_ENABLED=false';
export const FEISHU_BANNER_DISMISS_KEY = 'feed-puller-feishu-banner-dismissed';

export function isFeishuBannerDismissed(): boolean {
  if (typeof localStorage === 'undefined') {
    return false;
  }
  return localStorage.getItem(FEISHU_BANNER_DISMISS_KEY) === '1';
}

export function dismissFeishuBanner(): void {
  localStorage.setItem(FEISHU_BANNER_DISMISS_KEY, '1');
}

type FeishuLoginSetupGuideProps = {
  user: User;
  authOptions: AuthOptions;
  onBind: () => void;
  onCopyEnv?: (text: string) => void;
};

export function feishuSetupIncomplete(authOptions: AuthOptions | null, user: User): boolean {
  return (
    authOptions != null &&
    authOptions.password_login_enabled &&
    authOptions.feishu_login_enabled &&
    !user.feishu_bound
  );
}

export function FeishuLoginSetupGuide({ user, authOptions, onBind, onCopyEnv }: FeishuLoginSetupGuideProps) {
  if (!authOptions.feishu_login_enabled) {
    return null;
  }

  const step1Done = true;
  const step2Done = user.feishu_bound;
  const step3Ready = step2Done && authOptions.password_login_enabled;
  const migrationComplete = step2Done && !authOptions.password_login_enabled;

  if (migrationComplete) {
    return (
      <section className="feishu-setup-guide feishu-setup-guide--success" aria-labelledby="feishu-setup-title">
        <h2 id="feishu-setup-title">飞书登录迁移</h2>
        <p className="feishu-setup-success">已完成迁移，当前仅支持飞书扫码登录。</p>
      </section>
    );
  }

  if (!authOptions.password_login_enabled && !user.feishu_bound) {
    return (
      <section className="feishu-setup-guide feishu-setup-guide--warning" aria-labelledby="feishu-setup-title">
        <h2 id="feishu-setup-title">飞书登录迁移</h2>
        <p className="feishu-setup-warning">
          账号密码登录已关闭，但当前账号尚未绑定飞书。请使用仍具备密码登录权限的环境完成绑定，或临时将{' '}
          <code>PASSWORD_LOGIN_ENABLED</code> 设为 <code>true</code> 后重启服务。
        </p>
      </section>
    );
  }

  async function copyEnvLine() {
    try {
      await navigator.clipboard.writeText(PASSWORD_LOGIN_DISABLE_ENV);
      onCopyEnv?.(PASSWORD_LOGIN_DISABLE_ENV);
    } catch {
      onCopyEnv?.('');
    }
  }

  return (
    <section className="feishu-setup-guide" aria-labelledby="feishu-setup-title">
      <h2 id="feishu-setup-title">飞书登录迁移向导</h2>
      <p className="muted feishu-setup-intro">按以下步骤操作，可在绑定飞书后安全关闭账号密码登录，避免误配置导致无法登录。</p>
      <ol className="feishu-setup-steps">
        <li className={step1Done ? 'feishu-setup-step feishu-setup-step--done' : 'feishu-setup-step'}>
          <span className="feishu-setup-step-icon" aria-hidden="true">
            {step1Done ? <CheckCircle2 size={20} /> : <Circle size={20} />}
          </span>
          <div className="feishu-setup-step-body">
            <strong>使用账号密码登录</strong>
            <p className="muted">首次部署时使用环境变量中的管理员账号登录（你已完成此步）。</p>
          </div>
        </li>
        <li className={step2Done ? 'feishu-setup-step feishu-setup-step--done' : 'feishu-setup-step feishu-setup-step--current'}>
          <span className="feishu-setup-step-icon" aria-hidden="true">
            {step2Done ? <CheckCircle2 size={20} /> : <Circle size={20} />}
          </span>
          <div className="feishu-setup-step-body">
            <strong>绑定飞书账号</strong>
            <p className="muted">
              在下方「飞书备用登录」中扫码绑定。绑定成功后，可用飞书扫码登录同一管理员账号。
            </p>
            {!step2Done && (
              <button type="button" className="primary-link" onClick={onBind}>
                立即绑定飞书
              </button>
            )}
            {step2Done && <p className="feishu-setup-step-status">已绑定：{user.feishu_name || user.feishu_open_id || '飞书账号'}</p>}
          </div>
        </li>
        <li className={step3Ready ? 'feishu-setup-step feishu-setup-step--current' : 'feishu-setup-step'}>
          <span className="feishu-setup-step-icon" aria-hidden="true">
            {!authOptions.password_login_enabled ? <CheckCircle2 size={20} /> : <Circle size={20} />}
          </span>
          <div className="feishu-setup-step-body">
            <strong>关闭账号密码登录</strong>
            <p className="muted">确认飞书可正常登录后，在 <code>.env</code> 或 Docker 环境中设置并重启服务：</p>
            <div className="feishu-setup-env">
              <code>{PASSWORD_LOGIN_DISABLE_ENV}</code>
              <button type="button" className="ghost icon-text" onClick={copyEnvLine} disabled={!step3Ready}>
                <Copy size={16} aria-hidden="true" />
                复制
              </button>
            </div>
            {!step3Ready && <p className="feishu-setup-step-hint muted">请先完成飞书绑定后再关闭密码登录。</p>}
            {step3Ready && (
              <p className="feishu-setup-step-hint muted">修改环境变量后需重启 feed-puller 才会生效；重启前请再次确认飞书扫码可登录。</p>
            )}
          </div>
        </li>
      </ol>
    </section>
  );
}

type FeishuSetupBannerProps = {
  onGoSettings: () => void;
};

export function FeishuSetupBanner({ onGoSettings }: FeishuSetupBannerProps) {
  const [dismissed, setDismissed] = useState(() => isFeishuBannerDismissed());

  if (dismissed) {
    return null;
  }

  function handleDismiss() {
    dismissFeishuBanner();
    setDismissed(true);
  }

  return (
    <div className="feishu-setup-banner" role="status">
      <p>
        <strong>建议完成飞书登录迁移：</strong>
        绑定飞书后可关闭账号密码登录，降低账号泄露风险。
      </p>
      <div className="feishu-setup-banner-actions">
        <button type="button" className="primary-link" onClick={onGoSettings}>
          前往设置
        </button>
        <button type="button" className="ghost feishu-setup-banner-dismiss" onClick={handleDismiss}>
          不再提示
        </button>
      </div>
    </div>
  );
}
