import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useActivity } from '@/api/activity';
import { LearningRhythm } from './LearningRhythm';

vi.mock('@/api/activity', () => ({ useActivity: vi.fn() }));

const summary = {
  rangeStart: '2026-07-06', rangeEnd: '2026-07-11', activeDays: 2,
  currentStreak: 2, longestStreak: 4, totalActions: 3, totalGrowth: 18,
  days: [
    { date: '2026-07-10', activity: 2, growth: 0, actions: 1, events: [{ id: 'r1', sourceType: 'reading', sourceId: 'p1', activityDelta: 2, delta: 0, title: '有效阅读', detail: '核心概念', createdAt: '2026-07-10T10:00:00Z', projectSlug: 'math', projectTitle: '数学' }] },
    { date: '2026-07-11', activity: 3, growth: 18, actions: 2, events: [] },
  ],
};

describe('LearningRhythm', () => {
  beforeEach(() => vi.mocked(useActivity).mockReturnValue({ data: summary, isLoading: false } as ReturnType<typeof useActivity>));

  it('shows consistency, investment and growth as separate metrics', () => {
    render(<LearningRhythm projects={[{ id: 'math', slug: 'math', title: '数学', projectType: 'system-learning', overviewAvailable: false }]} />);
    expect(screen.getByText('天 · 当前连续学习').previousElementSibling).toHaveTextContent('2');
    expect(screen.getByText('有效学习行动')).toBeTruthy();
    expect(screen.getByText('18')).toBeTruthy();
    fireEvent.click(screen.getByLabelText(/7月10日/));
    expect(screen.getByText('有效阅读')).toBeTruthy();
  });

  it('supports project and range filters', () => {
    render(<LearningRhythm projects={[{ id: 'math', slug: 'math', title: '数学', projectType: 'system-learning', overviewAvailable: false }]} />);
    fireEvent.change(screen.getByLabelText('筛选学习项目'), { target: { value: 'math' } });
    fireEvent.click(screen.getByText('近一年'));
    expect(useActivity).toHaveBeenLastCalledWith(52, 'math');
  });
});
