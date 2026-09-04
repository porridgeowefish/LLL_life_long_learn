import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { UsagePage } from './UsagePage';

vi.mock('./api', () => ({
  useTeacherUsage: () => ({ data: { total: 1, conversations: [{ projectSlug: 'parallel', title: '并发、并行与计算模型', conversationId: 'conv_1', inputTokens: 120, outputTokens: 80, turns: [{ responseId: 'resp_1', inputTokens: 120, outputTokens: 80, occurredAt: '2026-09-04T00:00:00Z' }] }] }, isLoading: false }),
}));

describe('UsagePage', () => {
  it('shows teacher usage grouped by durable conversation and its turns', () => {
    render(<UsagePage />);
    expect(screen.getByRole('heading', { name: '教师 Token 用量' })).toBeInTheDocument();
    expect(screen.getByText('并发、并行与计算模型')).toBeInTheDocument();
    expect(screen.getAllByText(/输入 120/)).toHaveLength(2);
    expect(screen.getAllByText(/输出 80/)).toHaveLength(2);
  });
});
