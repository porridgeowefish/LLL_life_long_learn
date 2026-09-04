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

  it('renders determinate "N / M 页" text when pagesPlanned > 0', () => {
    render(<RunProgressBar active={true} activity={'running'} pagesDone={3} pagesPlanned={5} />);
    expect(screen.getByText('3 / 5 页')).toBeTruthy();
    // The activity text must NOT win when the determinate label is shown.
    expect(screen.queryByText('running')).toBeNull();
  });

  it('stays indeterminate (shows activity) when pagesPlanned is 0', () => {
    render(<RunProgressBar active={true} activity={'最近更新：explain'} pagesDone={0} pagesPlanned={0} />);
    expect(screen.getByText('最近更新：explain')).toBeTruthy();
  });
});
