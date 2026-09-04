import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BodyAnnotations } from './BodyAnnotations';

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  remove: vi.fn(),
  openActive: vi.fn(),
  applyHighlights: vi.fn(),
}));

vi.mock('@/features/learning/api/learningWorkspace', () => ({
  useBodyAnnotations: () => ({ data: [], isLoading: false }),
  useCreateBodyAnnotation: () => ({ mutateAsync: mocks.create, isPending: false }),
  useDeleteBodyAnnotation: () => ({ mutate: mocks.remove }),
}));

vi.mock('@/features/settings', () => ({
  useAskAiSettings: () => ({ data: { default: 'teacher-provider' } }),
}));

vi.mock('@/shared/store/slices/askAi', () => ({
  useAskAiStore: () => ({ openActive: mocks.openActive, openReview: vi.fn() }),
}));

vi.mock('@/features/learning/components/AskAiPanel', () => ({ AskAiPanel: () => null }));

vi.mock('@/shared/primitive/MarkdownView', async () => {
  const React = await import('react');
  return {
    MarkdownView: React.forwardRef<HTMLDivElement, { source: string }>(function FakeMarkdown({ source }, ref) {
      const lines = source.split('\n');
      return <div ref={ref} data-testid="body-markdown">{lines.some((line) => /^##\s+/.test(line))
        ? lines.filter(Boolean).map((line, index) => /^##\s+/.test(line)
          ? <h2 key={index}>{line.replace(/^##\s+/, '')}</h2>
          : /^#\s+/.test(line) ? <h1 key={index}>{line.replace(/^#\s+/, '')}</h1> : <p key={index}>{line}</p>)
        : source}</div>;
    }),
  };
});

vi.mock('@/shared/lib/summaryHighlights', () => ({
  applyHighlights: mocks.applyHighlights,
  selectionToTextAnchor: () => ({ text: '极限', start: 2, end: 4 }),
}));

vi.mock('@/shared/lib/selectionGeometry', () => ({
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

  it('shows only the requested level-two section while preserving the full body DOM', async () => {
    const props = {
      slug: 'kubernetes',
      content: '# 正文\n\n导语\n\n## 第一节\n\n第一节内容\n\n## 第二节\n\n第二节内容',
      assetVersionId: 'body-v8',
      pageIndex: 0,
    } as Parameters<typeof BodyAnnotations>[0] & { pageIndex: number };
    const view = render(<BodyAnnotations {...props} />);

    await waitFor(() => expect(screen.getByText('第一节')).toBeVisible());
    expect(screen.getByText('正文')).toBeVisible();
    expect(screen.getByText('第二节')).not.toBeVisible();

    view.rerender(<BodyAnnotations {...props} pageIndex={1} />);
    await waitFor(() => expect(screen.getByText('第二节')).toBeVisible());
    expect(screen.getByText('正文')).not.toBeVisible();
    expect(screen.getByText('第一节')).not.toBeVisible();
  });
});
