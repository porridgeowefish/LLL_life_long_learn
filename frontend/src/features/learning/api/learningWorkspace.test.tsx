import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { learningWorkspaceKeys, useConversation, useRetrySource } from './learningWorkspace';

const clientMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock('@/shared/client', () => ({
  ApiError: class ApiError extends Error {},
  http: { get: clientMocks.get, post: clientMocks.post },
}));

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe('useConversation', () => {
  beforeEach(() => clientMocks.get.mockReset());

  it('loads the recent page first and prepends older messages on demand', async () => {
    clientMocks.get
      .mockResolvedValueOnce({
        conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 900,
        pageFromSeq: 501, pageThroughSeq: 900, hasMore: true, hasPrevious: true, totalMessages: 2,
        messages: [{ id: 'message-2' }], taskLinks: [{ messageId: 'message-2', taskId: 'task-2' }],
      })
      .mockResolvedValueOnce({
        conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 900,
        pageFromSeq: 1, pageThroughSeq: 500, hasMore: false, hasPrevious: false, totalMessages: 2,
        messages: [{ id: 'message-1' }], taskLinks: [{ messageId: 'message-1', taskId: 'task-1' }],
      });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('linear algebra'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(clientMocks.get).toHaveBeenCalledTimes(1);
    expect(clientMocks.get).toHaveBeenNthCalledWith(1, '/api/projects/linear%20algebra/conversation?limit=40&beforeSeq=0');
    expect(result.current.data?.messages.map((message) => message.id)).toEqual(['message-2']);
    await act(async () => { await result.current.loadPrevious(); });
    expect(clientMocks.get).toHaveBeenNthCalledWith(2, '/api/projects/linear%20algebra/conversation?limit=40&beforeSeq=501');
    await waitFor(() => expect(result.current.data?.messages.map((message) => message.id)).toEqual(['message-1', 'message-2']));
    expect(result.current.data?.taskLinks.map((link) => link.taskId)).toEqual(['task-1', 'task-2']);
    expect(result.current.data?.latestSeq).toBe(900);
    expect(result.current.data?.hasPrevious).toBe(false);
  });

  it('stops safely when a malformed page does not advance its cursor', async () => {
    clientMocks.get.mockResolvedValue({
      conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 10,
      pageFromSeq: 0, pageThroughSeq: 0, hasMore: true, hasPrevious: true, totalMessages: 0, messages: [], taskLinks: [],
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('unit'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(clientMocks.get).toHaveBeenCalledTimes(1);
  });

  it('normalizes tool-only messages with null blocks instead of crashing the teacher view', async () => {
    clientMocks.get.mockResolvedValue({
      conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 4,
      pageFromSeq: 4, pageThroughSeq: 4, hasMore: false, hasPrevious: false, totalMessages: 1,
      messages: [{ id: 'message-tool', role: 'teacher', blocks: null }], taskLinks: null,
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('unit'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.messages[0].blocks).toEqual([]);
    expect(result.current.data?.taskLinks).toEqual([]);
  });
});

it('retries a source in its project and marks the source ledger for refresh', async () => {
  clientMocks.post.mockReset();
  clientMocks.post.mockResolvedValue({ sourceId: 'failed pdf', status: 'processing' });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const sourceKey = learningWorkspaceKeys.sources('linear algebra');
  client.setQueryData(sourceKey, [{ sourceId: 'failed pdf', status: 'failed' }]);
  const { result } = renderHook(() => useRetrySource('linear algebra'), { wrapper: wrapper(client) });
  await act(async () => { await result.current.mutateAsync('failed pdf'); });
  expect(clientMocks.post).toHaveBeenCalledWith('/api/projects/linear%20algebra/sources/failed%20pdf/retry', {});
  expect(client.getQueryState(sourceKey)?.isInvalidated).toBe(true);
});
