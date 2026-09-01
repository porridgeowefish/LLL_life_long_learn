import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { OutputViewer } from './OutputViewer';

vi.mock('@tanstack/react-query', () => ({
  useQuery: () => ({ data: 'bottom text', isLoading: false, error: null }),
}));

vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: () => ({ html: '<p>bottom text</p>', mermaid: [] }),
}));

vi.mock('@/api/confusions', () => ({
  useCreateConfusion: () => ({ mutate: vi.fn() }),
}));

describe('OutputViewer selection toolbar', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1000 });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 760 });
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue(
      new DOMRect(0, 0, 240, 40),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('opens above a bottom-edge selection even when pointerup occurs outside the article', async () => {
    render(<OutputViewer slug="demo" zone="Explain" />);
    const paragraph = screen.getByText('bottom text');
    const range = document.createRange();
    range.selectNodeContents(paragraph);
    Object.defineProperty(range, 'getBoundingClientRect', {
      value: () => new DOMRect(300, 700, 320, 24),
    });
    Object.defineProperty(range, 'getClientRects', {
      value: () => [new DOMRect(300, 700, 320, 24)],
    });

    vi.spyOn(window, 'getSelection').mockReturnValue({
      isCollapsed: false,
      toString: () => 'bottom text',
      getRangeAt: () => range,
      removeAllRanges: vi.fn(),
    } as unknown as Selection);

    fireEvent.pointerUp(window, { clientX: 500, clientY: 712 });

    const toolbar = await screen.findByRole('toolbar', { name: '选中文本操作' });
    await waitFor(() => expect(Number.parseFloat(toolbar.style.top)).toBeLessThan(700));
    expect(toolbar.style.width).toBe('max-content');
    expect(toolbar.parentElement).toBe(document.body);
  });

  it('anchors a cross-line selection to the pointer release line', async () => {
    render(<OutputViewer slug="demo" zone="Explain" />);
    const paragraph = screen.getByText('bottom text');
    const range = document.createRange();
    range.selectNodeContents(paragraph);
    Object.defineProperty(range, 'getClientRects', {
      value: () => [
        new DOMRect(700, 300, 200, 24),
        new DOMRect(80, 340, 320, 24),
      ],
    });

    vi.spyOn(window, 'getSelection').mockReturnValue({
      isCollapsed: false,
      toString: () => 'bottom text',
      getRangeAt: () => range,
      removeAllRanges: vi.fn(),
    } as unknown as Selection);

    fireEvent.pointerUp(window, { clientX: 200, clientY: 352 });

    const toolbar = await screen.findByRole('toolbar', { name: '选中文本操作' });
    await waitFor(() => expect(Number.parseFloat(toolbar.style.top)).toBeGreaterThan(340));
    expect(Number.parseFloat(toolbar.style.left)).toBeLessThan(300);
  });

  it('shows the toolbar when a bottom-edge drag leaves the window before pointerup', async () => {
    render(<OutputViewer slug="demo" zone="Explain" />);
    const paragraph = screen.getByText('bottom text');
    const range = document.createRange();
    range.selectNodeContents(paragraph);
    Object.defineProperty(range, 'getClientRects', {
      value: () => [new DOMRect(300, 710, 320, 24)],
    });

    vi.spyOn(window, 'getSelection').mockReturnValue({
      isCollapsed: false,
      toString: () => 'bottom text',
      getRangeAt: () => range,
      removeAllRanges: vi.fn(),
    } as unknown as Selection);

    fireEvent.pointerDown(paragraph);
    fireEvent.mouseOut(window, { relatedTarget: null });

    const toolbar = await screen.findByRole('toolbar', { name: '选中文本操作' });
    await waitFor(() => expect(Number.parseFloat(toolbar.style.top)).toBeLessThan(710));
  });
});
