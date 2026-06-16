import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { QuestionCard } from './QuestionCard';

vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: (input: string) => ({ html: `<p>${input}</p>`, mermaid: [] }),
}));

describe('QuestionCard evaluation feedback', () => {
  it('shows the learner answer, AI feedback, and a concrete suggested answer', () => {
    render(
      <QuestionCard
        task={{
          id: 'q4',
          type: 'essay',
          difficulty: 4,
          question: '解释 DFA 如何识别 token。',
        }}
        index={3}
        total={5}
        draft={{ answer: '学习者的原回答', selfAssess: 3 }}
        onAnswerChange={vi.fn()}
        onAssessChange={vi.fn()}
        onCheckObjective={vi.fn()}
        readonly
        feedback={{
          taskId: 'q4',
          score: 3,
          feedback: '方向正确，但缺少接受状态。',
          suggestedAnswer: '从初始状态读取字符，并按转换函数进入接受状态。',
          evidence: '未说明接受状态。',
          passed: true,
        }}
      />,
    );

    expect(screen.getByText('学习者的原回答')).toBeInTheDocument();
    expect(screen.getByText('方向正确，但缺少接受状态。')).toBeInTheDocument();
    expect(screen.getByText('参考回答')).toBeInTheDocument();
    expect(screen.getByText('从初始状态读取字符，并按转换函数进入接受状态。')).toBeInTheDocument();
  });
});
