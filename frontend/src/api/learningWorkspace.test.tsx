import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useConversation } from './learningWorkspace';

const clientMocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('./client', () => ({
  ApiError: class ApiError extends Error {},
  http: { get: clientMocks.get },
}));

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe('useConversation', () => {
  beforeEach(() => clientMocks.get.mockReset());

  it('reads every page and combines messages and task links in order', async () => {
    clientMocks.get
      .mockResolvedValueOnce({
        conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 900,
        pageThroughSeq: 500, hasMore: true,
        messages: [{ id: 'message-1' }], taskLinks: [{ messageId: 'message-1', taskId: 'task-1' }],
      })
      .mockResolvedValueOnce({
        conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 900,
        pageThroughSeq: 900, hasMore: false,
        messages: [{ id: 'message-2' }], taskLinks: [{ messageId: 'message-2', taskId: 'task-2' }],
      });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('linear algebra'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(clientMocks.get).toHaveBeenNthCalledWith(1, '/api/projects/linear%20algebra/conversation?limit=500&afterSeq=0');
    expect(clientMocks.get).toHaveBeenNthCalledWith(2, '/api/projects/linear%20algebra/conversation?limit=500&afterSeq=500');
    expect(result.current.data?.messages.map((message) => message.id)).toEqual(['message-1', 'message-2']);
    expect(result.current.data?.taskLinks.map((link) => link.taskId)).toEqual(['task-1', 'task-2']);
    expect(result.current.data?.latestSeq).toBe(900);
    expect(result.current.data?.hasMore).toBe(false);
  });

  it('stops safely when a malformed page does not advance its cursor', async () => {
    clientMocks.get.mockResolvedValue({
      conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 10,
      pageThroughSeq: 0, hasMore: true, messages: [], taskLinks: [],
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('unit'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(clientMocks.get).toHaveBeenCalledTimes(1);
  });

  it('normalizes tool-only messages with null blocks instead of crashing the teacher view', async () => {
    clientMocks.get.mockResolvedValue({
      conversationId: 'conversation-1', unitId: 'unit-1', latestSeq: 4,
      pageThroughSeq: 4, hasMore: false,
      messages: [{ id: 'message-tool', role: 'teacher', blocks: null }], taskLinks: null,
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useConversation('unit'), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.messages[0].blocks).toEqual([]);
    expect(result.current.data?.taskLinks).toEqual([]);
  });
});
