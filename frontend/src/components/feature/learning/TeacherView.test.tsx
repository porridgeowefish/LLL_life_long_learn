import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { streamTeacherTurn } from '@/api/learningWorkspace';
import { TeacherView } from './TeacherView';

vi.mock('@/api/learningWorkspace', () => ({
  useConversation: () => ({ data: { messages: [], taskLinks: [] } }),
  useAssistantTasks: () => ({ data: [] }),
  useAssets: () => ({ data: [{ key: 'body', title: '正文', editRevision: 2 }] }),
  useSources: () => ({ data: [{ sourceId: 'source_notes', displayName: '闭包讲义', status: 'ready' }] }),
  useUploadSource: () => ({ mutateAsync: vi.fn(), isPending: false }),
  streamTeacherTurn: vi.fn(async () => undefined),
  resumeTeacherTurn: vi.fn(async () => false),
  stopTeacherResponse: vi.fn(async () => undefined),
}));

vi.mock('@/api/askAi', () => ({
  useAskAiSettings: () => ({ data: { default: 'p1', providers: [{ id: 'p1', name: '主教师', kind: 'openai', baseURL: 'https://example.test/v1', apiKey: '••••', model: 'gpt-new' }], bindings: {} } }),
}));

beforeEach(() => { vi.clearAllMocks(); localStorage.clear(); });

describe('TeacherView source references', () => {
  it('sends only learner-selected source ids with the teacher turn', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><TeacherView slug="closures" title="闭包" /></QueryClientProvider>);

    expect(await screen.findByRole('option', { name: '主教师 · gpt-new' })).toBeTruthy();

    fireEvent.click(screen.getByText('资料', { selector: 'summary' }));
    fireEvent.click(screen.getByRole('checkbox', { name: /闭包讲义/ }));
    fireEvent.change(screen.getByPlaceholderText('和 闭包 的教师继续讨论…'), { target: { value: '请结合资料解释' } });
    const transcript = screen.getByRole('log', { name: '教师对话' });
    Object.defineProperty(transcript, 'scrollHeight', { configurable: true, value: 1200 });
    transcript.scrollTop = 0;
    fireEvent.click(screen.getByRole('button', { name: '发送' }));

    await waitFor(() => expect(streamTeacherTurn).toHaveBeenCalled());
    await waitFor(() => expect(transcript.scrollTop).toBe(1200));
    expect(vi.mocked(streamTeacherTurn).mock.calls[0][1]).toMatchObject({
      content: '请结合资料解释',
      attachmentRefs: ['source_notes'],
      providerId: 'p1',
    });
  });

  it('shows an immediate thinking state and renders disclosed reasoning while streaming', async () => {
    let finish!: () => void;
    vi.mocked(streamTeacherTurn).mockImplementationOnce(async (_slug, _input, _signal, onFrame) => {
      onFrame({ type: 'message-started', data: { teacherMessageId: 'msg-teacher' } });
      onFrame({ type: 'reasoning-summary-delta', data: { blockId: 'reasoning', delta: '正在检查问题边界' } });
      await new Promise<void>((resolve) => { finish = resolve; });
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><TeacherView slug="closures" title="闭包" /></QueryClientProvider>);

    fireEvent.change(screen.getByPlaceholderText('和 闭包 的教师继续讨论…'), { target: { value: '为什么？' } });
    fireEvent.click(screen.getByRole('button', { name: '发送' }));

    expect(await screen.findByRole('status')).toHaveTextContent('教师正在生成…');
    const reasoningSummary = await screen.findByText('思考过程');
    const reasoning = reasoningSummary.closest('details');
    expect(reasoning).not.toHaveAttribute('open');
    expect(await screen.findByText('正在检查问题边界')).toBeTruthy();
    fireEvent.click(reasoningSummary);
    expect(reasoning).toHaveAttribute('open');
    await act(async () => { finish(); });
  });

  it('coalesces burst deltas and scrolls at most once per rendered frame', async () => {
    let emit!: Parameters<typeof streamTeacherTurn>[3];
    let finish!: () => void;
    vi.mocked(streamTeacherTurn).mockImplementationOnce(async (_slug, _input, _signal, onFrame) => {
      emit = onFrame;
      await new Promise<void>((resolve) => { finish = resolve; });
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><TeacherView slug="closures" title="闭包" /></QueryClientProvider>);

    const transcript = screen.getByRole('log', { name: '教师对话' });
    let scrollWrites = 0;
    let scrollTop = 0;
    Object.defineProperty(transcript, 'scrollHeight', { configurable: true, value: 1600 });
    Object.defineProperty(transcript, 'scrollTop', {
      configurable: true,
      get: () => scrollTop,
      set: (value: number) => { scrollTop = value; scrollWrites += 1; },
    });
    fireEvent.change(screen.getByPlaceholderText('和 闭包 的教师继续讨论…'), { target: { value: '连续输出' } });
    fireEvent.click(screen.getByRole('button', { name: '发送' }));
    await waitFor(() => expect(streamTeacherTurn).toHaveBeenCalled());
    await new Promise((resolve) => setTimeout(resolve, 50));
    scrollWrites = 0;

    const expected = '流'.repeat(120);
    act(() => {
      for (let index = 0; index < 120; index += 1) {
        emit({ type: 'text-delta', data: { blockId: 'text', delta: '流' } });
      }
    });

    expect(await screen.findByText(expected)).toBeTruthy();
    await new Promise((resolve) => requestAnimationFrame(() => resolve(undefined)));
    expect(scrollWrites).toBeLessThanOrEqual(2);
    await act(async () => { finish(); });
  });
});
