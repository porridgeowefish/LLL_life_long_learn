import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ProjectTypeAdvisorDialog } from './ProjectTypeAdvisorDialog';

const mutateAsync = vi.fn();

vi.mock('@/api/projects', () => ({
  useProjectTypeAdvice: () => ({
    mutateAsync,
    reset: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

describe('ProjectTypeAdvisorDialog', () => {
  it('opens before the project form is complete and supports adopting a conversational recommendation', async () => {
    mutateAsync.mockResolvedValueOnce({
      reply: '你想先了解研究领域，建议建立学科地图。',
      recommendation: 'discipline-map',
      reason: '先建立领域方向感',
      tradeoff: '不会立即进入具体教程',
      confidence: 'high',
    });
    const onAdopt = vi.fn();
    const onOpenChange = vi.fn();
    render(
      <ProjectTypeAdvisorDialog
        open
        onOpenChange={onOpenChange}
        draft={{ current: '未接触', target: '看懂原理' }}
        onAdopt={onAdopt}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText(/我想学博弈论/), {
      target: { value: '我想先知道博弈论有哪些研究领域' },
    });
    fireEvent.click(screen.getByRole('button', { name: '发送' }));

    expect(await screen.findByText('建议选择：学科地图')).toBeTruthy();
    expect(screen.getByText('取舍：不会立即进入具体教程')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '采用建议' }));
    expect(onAdopt).toHaveBeenCalledWith('discipline-map');
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
