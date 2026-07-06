import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

import { useArtifactRefresh } from './useArtifactRefresh';

// Capture SSE handlers by event name.
let handlers: Record<string, (data: unknown) => void> = {};
vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (name: string, h: (data: unknown) => void) => {
    handlers[name] = h;
    return () => {
      delete handlers[name];
    };
  },
}));

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe('useArtifactRefresh', () => {
  beforeEach(() => {
    handlers = {};
  });

  it('invalidates files + practice on artifact-updated for the slug', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'practice' }));

    const keys = spy.mock.calls.map((c) => (c[0] as { queryKey: unknown }).queryKey);
    expect(keys).toContainEqual(['files', 'myproj']);
    expect(keys).toContainEqual(['practice', 'tasks', 'myproj']);
  });

  it('ignores artifact events for other projects', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['artifact-updated']({ projectSlug: 'other', zone: 'explain' }));
    expect(spy).not.toHaveBeenCalled();
  });

  it('invalidates confusions on confusion-updated', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['confusion-updated']({ projectSlug: 'myproj', id: 'c1' }));
    expect(spy).toHaveBeenCalledWith({ queryKey: ['confusions', 'myproj'] });
  });
});
