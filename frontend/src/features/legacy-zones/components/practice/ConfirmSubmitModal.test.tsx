import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ConfirmSubmitModal } from './ConfirmSubmitModal';

describe('ConfirmSubmitModal', () => {
  it('blocks a final submission while any question is empty', () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmSubmitModal
        open
        onOpenChange={vi.fn()}
        tasks={[
          { id: 'q1', question: 'one', type: 'short-answer', difficulty: 2 },
          { id: 'q2', question: 'two', type: 'essay', difficulty: 4 },
        ]}
        drafts={{
          q1: { answer: 'answer', selfAssess: 3 },
          q2: { answer: '', selfAssess: 0 },
        }}
        onConfirm={onConfirm}
      />,
    );

    const submit = screen.getByRole('button', { name: '请先完成全部题目' });
    expect(submit).toBeDisabled();
    fireEvent.click(submit);
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
