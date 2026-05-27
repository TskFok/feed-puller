import { useEffect, useRef, type ReactNode, type RefObject } from 'react';
import { createPortal } from 'react-dom';
import { getFocusableElements, handleFocusTrapKeyDown } from './focusTrap';

type AnimatedModalProps = {
  onClose: () => void;
  ariaLabelledBy: string;
  panelClassName?: string;
  initialFocusRef?: RefObject<HTMLElement>;
  children: ReactNode;
};

export function AnimatedModal({ onClose, ariaLabelledBy, panelClassName, initialFocusRef, children }: AnimatedModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const raf = requestAnimationFrame(() => {
      if (initialFocusRef?.current) {
        initialFocusRef.current.focus();
        return;
      }
      const dialog = dialogRef.current;
      if (!dialog) {
        return;
      }
      const [first] = getFocusableElements(dialog);
      first?.focus();
    });

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose();
        return;
      }
      const dialog = dialogRef.current;
      if (dialog) {
        handleFocusTrapKeyDown(event, dialog);
      }
    }

    document.addEventListener('keydown', onKeyDown);
    return () => {
      cancelAnimationFrame(raf);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', onKeyDown);
      previousFocusRef.current?.focus({ preventScroll: true });
    };
  }, [initialFocusRef, onClose]);

  const panelClass = panelClassName ? `modal-panel ${panelClassName}` : 'modal-panel';

  return createPortal(
    <div className="modal-overlay" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={ariaLabelledBy}
        className={panelClass}
        onMouseDown={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body
  );
}
