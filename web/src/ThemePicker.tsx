import { Monitor, Moon, Sun } from 'lucide-react';
import { useTheme, type ThemePreference } from './theme';

type ThemePickerProps = {
  variant?: 'panel' | 'compact';
};

const THEME_OPTIONS: { value: ThemePreference; label: string; icon: typeof Moon }[] = [
  { value: 'dark', label: 'Y2K 暗色', icon: Moon },
  { value: 'light', label: 'Bubblegum 浅色', icon: Sun },
  { value: 'system', label: '跟随系统', icon: Monitor }
];

export function ThemePicker({ variant = 'panel' }: ThemePickerProps) {
  const { preference, setPreference } = useTheme();
  const compact = variant === 'compact';

  return (
    <div className={compact ? 'login-theme-picker' : undefined}>
      {compact && <p className="login-theme-picker-label muted">外观</p>}
      {!compact && (
        <>
          <h2>外观</h2>
          <p className="muted">Y2K 暗色、Bubblegum 浅色，或跟随系统偏好。选择后会记住你的设置。</p>
        </>
      )}
      <div
        className={compact ? 'theme-toggle-group theme-toggle-group--compact' : 'theme-toggle-group'}
        role="group"
        aria-label="主题选择"
      >
        {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
          <button
            key={value}
            type="button"
            className={preference === value ? 'ghost theme-toggle-btn active' : 'ghost theme-toggle-btn'}
            onClick={() => setPreference(value)}
            aria-label={compact ? label : undefined}
            title={compact ? label : undefined}
          >
            <Icon size={16} aria-hidden />
            {!compact && label}
          </button>
        ))}
      </div>
    </div>
  );
}
