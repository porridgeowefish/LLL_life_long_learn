import { render, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { mountMermaidBlocks } from '@/lib/mermaidRenderer';
import { MarkdownView } from './MarkdownView';

vi.mock('@/lib/mermaidRenderer', () => ({ mountMermaidBlocks: vi.fn(() => undefined) }));

describe('MarkdownView streaming performance boundary', () => {
  it('defers Mermaid until the streamed message is complete', async () => {
    const source = '```mermaid\nflowchart TD\nA["开始"] --> B["结束"]\n```';
    const view = render(<MarkdownView source={source} streaming />);

    await waitFor(() => expect(view.getByText('图表将在回答完成后渲染')).toBeTruthy());
    expect(mountMermaidBlocks).not.toHaveBeenCalled();

    view.rerender(<MarkdownView source={source} streaming={false} />);
    await waitFor(() => expect(mountMermaidBlocks).toHaveBeenCalledTimes(1));
  });
});
