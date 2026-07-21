import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { TruncatedText } from './TruncatedText';

describe('TruncatedText', () => {
  it('保留完整文本作为原生提示并应用单行省略类', () => {
    const value = '一个非常长的列表文本，用于确认不会换行且可在悬停时查看全文';
    render(<TruncatedText>{value}</TruncatedText>);

    const text = screen.getByText(value);
    expect(text).toHaveAttribute('title', value);
    expect(text).toHaveClass('truncated-text');
  });

  it('合并调用方传入的样式类', () => {
    render(<TruncatedText className="muted">完整路径</TruncatedText>);

    expect(screen.getByText('完整路径')).toHaveClass('truncated-text', 'muted');
  });
});
