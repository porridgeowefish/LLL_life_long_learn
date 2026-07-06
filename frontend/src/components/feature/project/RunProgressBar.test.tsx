import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { RunProgressBar } from './RunProgressBar';

describe('RunProgressBar', () => {
  it('renders nothing when inactive', () => {
    const { container } = render(<RunProgressBar active={false} activity={null} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders the activity text when active', () => {
    render(<RunProgressBar active={true} activity={'最近更新：explain'} />);
    expect(screen.getByText('最近更新：explain')).toBeTruthy();
  });

  it('falls back to the running label when activity is null', () => {
    render(<RunProgressBar active={true} activity={null} />);
    expect(screen.getByText('运行中…')).toBeTruthy();
  });

  it('calls onDismiss when the dismiss button is clicked', () => {
    const onDismiss = vi.fn();
    render(<RunProgressBar active={true} activity={'x'} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole('button', { name: '收起进度' }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
