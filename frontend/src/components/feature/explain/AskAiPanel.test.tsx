import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { AskAiPanel } from './AskAiPanel';
import { useAskAiStore } from '@/store/slices/askAi';

const askAiMocks = vi.hoisted(() => ({
  streamAskAi: vi.fn(),
  summarizeAsk: vi.fn(),
}));

vi.mock('@/api/askAi', () => ({
  streamAskAi: askAiMocks.streamAskAi,
  summarizeAsk: askAiMocks.summarizeAsk,
  useAskAiSettings: () => ({ data: { default: 'p1', searchEngine: 'google', providers: [{ id: 'p1', kind: 'openai', baseURL: 'x', apiKey: 'k', model: 'm' }] } }),
}));
vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: (src: string) => ({ html: `<div>${src}</div>`, mermaid: [] }),
}));

beforeEach(() => {
  Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true });
  Object.defineProperty(window, 'innerHeight', { value: 768, configurable: true });
  askAiMocks.streamAskAi.mockReset();
  askAiMocks.summarizeAsk.mockReset();
  useAskAiStore.setState({ open: false, messages: [], streaming: false, reviewMode: false });
});

describe('AskAiPanel', () => {
  it('renders nothing when closed', () => {
    const { container } = render(<AskAiPanel />);
    expect(container.firstChild).toBeNull();
  });

  it('centers the panel on the selection anchor', () => {
    Object.defineProperty(window, 'innerWidth', { value: 1000, configurable: true });
    Object.defineProperty(window, 'innerHeight', { value: 800, configurable: true });
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 500 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    const dialog = screen.getByRole('dialog', { name: '问 AI' });
    expect(dialog.style.left).toBe('270px');
    expect(dialog.style.top).toBe('108px');
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

  it('pins review mode to the viewport right edge', () => {
    useAskAiStore.getState().openReview({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: -9999 },
      messages: [{ id: 'm1', role: 'assistant', content: 'prior answer', createdAt: '' }],
    });
    render(<AskAiPanel />);
    const dialog = screen.getByRole('dialog', { name: '答疑回顾' });
    expect(dialog.style.right).toBe('8px');
    expect(dialog.style.left).toBe('');
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

  it('escape closes the panel', () => {
    useAskAiStore.getState().openReview({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 },
      messages: [{ id: 'm1', role: 'assistant', content: 'prior answer', createdAt: '' }],
    });
    render(<AskAiPanel />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(useAskAiStore.getState().open).toBe(false);
  });

  it('shows a visible message when the stream returns an error frame', async () => {
    askAiMocks.streamAskAi.mockImplementation(async (_slug, _cid, _body, handlers) => {
      handlers.onFrame({ type: 'error', content: 'provider quota exceeded' });
    });
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    fireEvent.change(screen.getByPlaceholderText(/问/), { target: { value: '回答' } });
    fireEvent.click(screen.getByRole('button', { name: '发送' }));

    expect(await screen.findByText(/回答失败：provider quota exceeded/)).toBeTruthy();
  });

  it('shows a setup hint when the request fails before streaming', async () => {
    askAiMocks.streamAskAi.mockRejectedValue(new Error('ask-ai not configured'));
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    fireEvent.change(screen.getByPlaceholderText(/问/), { target: { value: '回答' } });
    fireEvent.click(screen.getByRole('button', { name: '发送' }));

    expect(await screen.findByText(/模型源还没配置/)).toBeTruthy();
  });
});
