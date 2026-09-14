import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useActivityRefresh } from './useActivityRefresh';

let handler: ((data: unknown) => void) | undefined;
vi.mock('@/app/hooks/useSSE', () => ({
  subscribeToSSE: (_name: string, callback: (data: unknown) => void) => {
    handler = callback;
    return () => { handler = undefined; };
  },
}));

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe('useActivityRefresh', () => {
  beforeEach(() => { handler = undefined; });

  it('invalidates every activity summary after the shared activity SSE event', () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useActivityRefresh(), { wrapper: wrapper(client) });

    act(() => handler?.({ projectSlug: 'python' }));

    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['activity'] });
  });
});
