import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

import { UsagePage } from './UsagePage';

vi.mock('./api', () => ({
  useProjectUsage: () => ({ data: { projects: [{ projectSlug: 'parallel', title: '并发、并行与计算模型', conversationId: 'conv_1', inputTokens: 120, outputTokens: 80, noteInputTokens: 20, noteOutputTokens: 10, turns: [{ responseId: 'resp_1', inputTokens: 120, outputTokens: 80, occurredAt: '2026-09-04T00:00:00Z' }] }] }, isLoading: false }),
}));

const days = Array.from({ length: 7 }, (_, index) => ({
  date: `2026-09-0${index + 1}`,
  activity: index < 2 ? index + 1 : 0,
  growth: 0,
  actions: index < 2 ? index + 1 : 0,
  events: [],
}));

vi.mock('@/features/projects', () => ({
  OutputViewer: () => null,
  useActivity: () => ({ data: { activeDays: 2, currentStreak: 0, longestStreak: 2, totalActions: 3, totalGrowth: 0, days }, isLoading: false }),
}));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<MemoryRouter><QueryClientProvider client={client}><UsagePage /></QueryClientProvider></MemoryRouter>);
}

describe('UsagePage', () => {
  it('shows project details when a token bar is focused or selected', () => {
    renderPage();
    expect(screen.getByRole('heading', { name: 'Token 用量' })).toBeInTheDocument();
    const bar = screen.getByRole('button', {name: /并发、并行与计算模型.*230/});
    fireEvent.focus(bar);
    expect(screen.getByRole('tooltip')).toHaveTextContent('并发、并行与计算模型');
    expect(screen.getByRole('tooltip')).toHaveTextContent('笔记输入 20');
    fireEvent.click(bar);
    expect(bar).toHaveAttribute('aria-pressed', 'true');
    fireEvent.blur(bar);
    expect(screen.getByRole('link',{name:'进入项目'})).toHaveAttribute('href','/project/parallel');
  });

  it('renders the learning-investment heatmap with clickable days', () => {
    renderPage();
    expect(screen.getByRole('heading', { name: '投入热力图' })).toBeInTheDocument();
    const grid = screen.getByRole('grid', { name: '最近26周学习投入' });
    expect(grid).toBeInTheDocument();
    expect(screen.getByText(/近26周活跃 2 天 · 有效行动 3 次/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('gridcell', { name: /2026-09-01/ }));
    expect(screen.getByText(/2026-09-01 · 热度 1 · 1 次行动/)).toBeInTheDocument();
  });
});
