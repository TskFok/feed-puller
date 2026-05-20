import { X } from 'lucide-react';
import type { ReactNode } from 'react';

export type BannerVariant = 'success' | 'error';

type BannerBaseProps = {
  children: ReactNode;
  className?: string;
};

type SuccessBannerProps = BannerBaseProps & {
  variant: 'success';
};

type ErrorBannerProps = BannerBaseProps & {
  variant: 'error';
  onDismiss: () => void;
};

export type BannerProps = SuccessBannerProps | ErrorBannerProps;

export function Banner(props: BannerProps) {
  const { variant, children, className } = props;
  const classes = ['banner', `banner-${variant}`, className].filter(Boolean).join(' ');

  if (variant === 'error') {
    const { onDismiss } = props;
    return (
      <div role="alert" className={classes}>
        <div className="banner-body">{children}</div>
        <button
          type="button"
          className="banner-dismiss"
          aria-label="关闭错误提示"
          onClick={onDismiss}
        >
          <X size={16} aria-hidden="true" />
        </button>
      </div>
    );
  }

  return (
    <div role="alert" className={classes}>
      {children}
    </div>
  );
}
