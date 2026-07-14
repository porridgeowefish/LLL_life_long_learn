import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { IntroPage } from './IntroPage';

const assessmentData = vi.hoisted(() => ({
  schemaVersion: 1,
  baseline: '能识别单变量分布，但随机向量工具尚需补足。',
  prerequisites: [
    {
      id: 'joint-probability',
      title: '联合概率与条件分布',
      assessedLevel: 'basic',
      status: 'weak',
      summary: '联合概率描述多个随机变量同时取值；当前缺口是还不能稳定区分联合、边缘与条件分布。',
      impact: '这会影响后续理解协方差与随机向量。',
      evidence: '调查回答只覆盖了单变量正态分布。',
    },
  ],
}));

const surveyData = vi.hoisted(() => ({
  schemaVersion: 1,
  updatedAt: '2026-07-15T00:00:00Z',
  questions: [{ id: 'q1', label: '应用', prompt: '请举一个例子', answer: '' }],
}));

vi.mock('@/api/learningArtifacts', () => ({
  useIntroAssessment: () => ({ data: assessmentData }),
  useIntroSurvey: () => ({ data: surveyData }),
  useSaveIntroSurvey: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false }),
}));

vi.mock('@/components/feature/project/OutputViewer', () => ({
  OutputViewer: () => <section data-testid="intro-output">Intro 生成内容</section>,
}));

describe('IntroPage', () => {
  it('shows concise gap context without a supplement action and keeps the survey last', () => {
    render(<IntroPage projectSlug="probability" />);

    expect(screen.getByText(/联合概率描述多个随机变量同时取值/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '快速补充' })).toBeNull();

    const output = screen.getByTestId('intro-output');
    const assessment = screen.getByRole('heading', { name: '前置知识诊断' }).closest('section');
    const survey = screen.getByRole('heading', { name: '能力边界校准' }).closest('section');
    expect(assessment).not.toBeNull();
    expect(survey).not.toBeNull();
    expect(output.compareDocumentPosition(assessment!)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(assessment!.compareDocumentPosition(survey!)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });
});
