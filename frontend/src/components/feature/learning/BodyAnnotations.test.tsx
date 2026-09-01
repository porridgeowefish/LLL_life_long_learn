import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BodyAnnotations } from './BodyAnnotations';

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  remove: vi.fn(),
  openActive: vi.fn(),
  applyHighlights: vi.fn(),
}));

vi.mock('@/api/learningWorkspace', () => ({
  useBodyAnnotations: () => ({ data: [], isLoading: false }),
  useCreateBodyAnnotation: () => ({ mutateAsync: mocks.create, isPending: false }),
  useDeleteBodyAnnotation: () => ({ mutate: mocks.remove }),
}));

vi.mock('@/api/askAi', () => ({
  useAskAiSettings: () => ({ data: { default: 'teacher-provider' } }),
}));

vi.mock('@/store/slices/askAi', () => ({
  useAskAiStore: () => ({ openActive: mocks.openActive, openReview: vi.fn() }),
}));

vi.mock('@/components/feature/explain/AskAiPanel', () => ({ AskAiPanel: () => null }));

vi.mock('@/components/primitive/MarkdownView', async () => {
  const React = await import('react');
  return {
    MarkdownView: React.forwardRef<HTMLDivElement, { source: string }>(function FakeMarkdown({ source }, ref) {
      return <div ref={ref} data-testid="body-markdown">{source}</div>;
    }),
  };
});

vi.mock('@/lib/summaryHighlights', () => ({
  applyHighlights: mocks.applyHighlights,
  selectionToTextAnchor: () => ({ text: '极限', start: 2, end: 4 }),
}));

vi.mock('@/lib/selectionGeometry', () => ({
  selectionEndpointRect: () => ({ top: 10, bottom: 30, left: 40, right: 80, width: 40, height: 20 }),
}));

describe('BodyAnnotations', () => {
  beforeEach(() => {
    mocks.create.mockReset();
    mocks.remove.mockReset();
    mocks.openActive.mockReset();
    mocks.applyHighlights.mockReset();
    Object.defineProperty(window, 'getSelection', {
      configurable: true,
      value: () => ({
        isCollapsed: false,
        rangeCount: 1,
        getRangeAt: () => ({}),
        removeAllRanges: vi.fn(),
      }),
    });
  });

  it('turns a text selection into a versioned annotation before opening Ask AI', async () => {
    mocks.create.mockResolvedValue({
      annotation: {
        annotationId: 'annotation-1', assetVersionId: 'body-v7', quoteSnapshot: '极限',
        anchors: { start: 2, end: 4 }, ask: { messages: [] },
      },
    });
    render(<BodyAnnotations slug="calculus" content="函数极限定义" assetVersionId="body-v7" />);

    fireEvent.pointerUp(screen.getByTestId('body-markdown'), { clientX: 60, clientY: 30 });
    expect(screen.getByRole('toolbar', { name: '选中文本操作' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '问 AI' }));

    await waitFor(() => expect(mocks.create).toHaveBeenCalledWith({
      assetVersionId: 'body-v7',
      quoteSnapshot: '极限',
      anchors: { start: 2, end: 4, prefix: '函数', suffix: '定义' },
    }));
    expect(mocks.openActive).toHaveBeenCalledWith(expect.objectContaining({
      projectSlug: 'calculus',
      confusionId: 'annotation-1',
      quote: '极限',
      providerId: 'teacher-provider',
      initialInput: '请解释「极限」',
      anchor: expect.objectContaining({ left: 40, bottom: 30 }),
    }));
  });
});
