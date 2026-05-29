import { AlertCircle, CheckCircle2, X } from 'lucide-react';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode
} from 'react';
import { createPortal } from 'react-dom';

export const TOAST_DISMISS_MS = 4000;

export type ToastVariant = 'success' | 'error';

export type ToastAction = {
  label: string;
  onClick: () => void;
};

export type ToastOptions = {
  action?: ToastAction;
};

type ToastItem = {
  id: string;
  message: string;
  variant: ToastVariant;
  action?: ToastAction;
};

type ToastContextValue = {
  showToast: (message: string, variant?: ToastVariant, options?: ToastOptions) => void;
  dismissToast: (id: string) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

function nextToastId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const dismissToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
  }, []);

  const showToast = useCallback(
    (message: string, variant: ToastVariant = 'success', options?: ToastOptions) => {
      if (!message) return;
      const id = nextToastId();
      setToasts((prev) => [...prev, { id, message, variant, action: options?.action }]);
      const timer = setTimeout(() => dismissToast(id), TOAST_DISMISS_MS);
      timersRef.current.set(id, timer);
    },
    [dismissToast]
  );

  useEffect(
    () => () => {
      for (const timer of timersRef.current.values()) {
        clearTimeout(timer);
      }
      timersRef.current.clear();
    },
    []
  );

  return (
    <ToastContext.Provider value={{ showToast, dismissToast }}>
      {children}
      {createPortal(<ToastViewport toasts={toasts} onDismiss={dismissToast} />, document.body)}
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return ctx;
}

function ToastViewport({
  toasts,
  onDismiss
}: {
  toasts: ToastItem[];
  onDismiss: (id: string) => void;
}) {
  if (toasts.length === 0) return null;

  return (
    <div className="toast-viewport" aria-live="polite" aria-relevant="additions">
      {toasts.map((toast) => (
        <ToastCard key={toast.id} toast={toast} onDismiss={() => onDismiss(toast.id)} />
      ))}
    </div>
  );
}

function ToastCard({ toast, onDismiss }: { toast: ToastItem; onDismiss: () => void }) {
  const Icon = toast.variant === 'success' ? CheckCircle2 : AlertCircle;
  const label = toast.variant === 'success' ? '操作成功' : '操作失败';

  function handleActionClick() {
    toast.action?.onClick();
    onDismiss();
  }

  return (
    <div role="status" className={`toast toast-${toast.variant}`}>
      <Icon className="toast-icon" size={18} aria-hidden="true" />
      <div className="toast-body">
        <span className="toast-label">{label}</span>
        <p className="toast-message">{toast.message}</p>
        {toast.action && (
          <button type="button" className="toast-action" onClick={handleActionClick}>
            {toast.action.label}
          </button>
        )}
      </div>
      <button type="button" className="toast-dismiss" aria-label="关闭提示" onClick={onDismiss}>
        <X size={16} aria-hidden="true" />
      </button>
    </div>
  );
}
