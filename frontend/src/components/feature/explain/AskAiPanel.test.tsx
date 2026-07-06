import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { AskAiPanel } from './AskAiPanel';
import { useAskAiStore } from '@/store/slices/askAi';

vi.mock('@/api/askAi', () => ({
  streamAskAi: vi.fn(),
  summarizeAsk: vi.fn(),
  useAskAiSettings: () => ({ data: { default: 'p1', searchEngine: 'google', providers: [{ id: 'p1', kind: 'openai', baseURL: 'x', apiKey: 'k', model: 'm' }] } }),
}));
vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: (src: string) => ({ html: `<div>${src}</div>`, mermaid: [] }),
}));

beforeEach(() => {
  useAskAiStore.setState({ open: false, messages: [], streaming: false, reviewMode: false });
});

describe('AskAiPanel', () => {
  it('renders nothing when closed', () => {
    const { container } = render(<AskAiPanel />);
    expect(container.firstChild).toBeNull();
  });

  it('active mode shows input + provider switcher + browser search', () => {
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    expect(screen.getByPlaceholderText(/问/)).toBeTruthy();
    expect(screen.getByTitle(/浏览器搜索/)).toBeTruthy();
  });

  it('review mode is read-only (no input)', () => {
    useAskAiStore.getState().openReview({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 },
      messages: [{ id: 'm1', role: 'assistant', content: 'prior answer', createdAt: '' }],
    });
    render(<AskAiPanel />);
    expect(screen.queryByPlaceholderText(/问/)).toBeNull();
    expect(screen.getByText('prior answer')).toBeTruthy();
  });

  it('close button closes the panel', () => {
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    expect(useAskAiStore.getState().open).toBe(false);
  });
});
