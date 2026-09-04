import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { PracticeProgress } from './PracticeProgress';

describe('PracticeProgress', () => {
  it('allows navigation after submission while answers stay locked elsewhere', () => {
    const onJump = vi.fn();
    render(
      <PracticeProgress
        tasks={[
          { id: 'q1', type: 'true-false', difficulty: 1, question: 'q1' },
          { id: 'q2', type: 'essay', difficulty: 3, question: 'q2' },
        ]}
        drafts={{
          q1: { answer: true, selfAssess: 3 },
          q2: { answer: 'answer', selfAssess: 4 },
        }}
        currentIndex={0}
        onJump={onJump}
        locked
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: '第 2 题，状态 completed' }));
    expect(onJump).toHaveBeenCalledWith(1);
  });
});
