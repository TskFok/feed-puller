import { describe, expect, it } from 'vitest';
import { getFocusableElements, handleFocusTrapKeyDown } from './focusTrap';

describe('focusTrap', () => {
  it('getFocusableElements 返回可聚焦元素', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <input aria-label="a" />
      <button type="button">b</button>
      <button type="button" disabled>b2</button>
    `;
    const focusable = getFocusableElements(container);
    expect(focusable).toHaveLength(2);
  });

  it('Tab 在末项时循环到首项', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <input aria-label="a" />
      <button type="button" id="last">b</button>
    `;
    document.body.appendChild(container);
    const last = container.querySelector('#last') as HTMLButtonElement;
    last.focus();

    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true });
    let prevented = false;
    event.preventDefault = () => {
      prevented = true;
    };
    handleFocusTrapKeyDown(event, container);

    expect(prevented).toBe(true);
    expect(document.activeElement).toBe(container.querySelector('input'));
    container.remove();
  });
});
