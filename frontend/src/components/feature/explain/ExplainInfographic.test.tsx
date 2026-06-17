import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

import type { InfographicState } from '@/api/explainInfographic';
import { ExplainInfographic } from './ExplainInfographic';

// vi.hoisted keeps the mock fns stable across vi.mock's hoisting so the test
// can drive them directly.
const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  sseHandler: null as ((data: unknown) => void) | null,
}));

// Replace the HTTP client the query/mutation call.
vi.mock('@/api/client', () => ({
  http: { get: mocks.get, post: mocks.post },
}));

// Capture the handler ExplainInfographic registers via subscribeToSSE so the
// test can fire a synthetic artifact-updated event.
vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (_eventName: unknown, handler: (data: unknown) => void) => {
    mocks.sseHandler = handler;
    return () => {
      mocks.sseHandler = null;
    };
  },
}));

function withProviders(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return <QueryClientProvider client={client}>{ui}</QueryClientProvider>;
}

const completeState: InfographicState = {
  status: 'complete',
  url: '/files/projects/p1/explain/infographic.png',
};

describe('ExplainInfographic — SSE-driven invalidation', () => {
  beforeEach(() => {
    mocks.sseHandler = null;
    mocks.get.mockReset();
    mocks.post.mockReset();
    // First GET returns complete, so the component renders the image and does
    // not auto-kick (POST). Only an SSE event should trigger a refetch.
    mocks.get.mockResolvedValue(completeState);
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('invalidates the infographic query when an artifact-updated event arrives for this project', async () => {
    render(withProviders(<ExplainInfographic projectSlug="p1" />));

    // Initial fetch settles.
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(1));

    // Fire the SSE event the backend emits on infographic completion.
    expect(mocks.sseHandler).not.toBeNull();
    await act(async () => {
      mocks.sseHandler?.({ slug: 'p1', artifact: 'explain/infographic.png' });
    });

    // Invalidation must trigger a refetch.
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
  });

  it('ignores artifact-updated events for other projects', async () => {
    render(withProviders(<ExplainInfographic projectSlug="p1" />));

    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(1));

    await act(async () => {
      mocks.sseHandler?.({ slug: 'other-proj', artifact: 'explain/infographic.png' });
    });

    // Give any would-be refetch a chance to fire, then assert it did not.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });

  it('regenerates with force=1 only after confirming in the modal', async () => {
    render(withProviders(<ExplainInfographic projectSlug="p1" />));

    // Complete state renders without auto-kicking (no POST yet).
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(1));
    expect(mocks.post).not.toHaveBeenCalled();

    // The regenerate button is available even after a successful generation.
    const regenerate = await screen.findByRole('button', { name: '重新生成' });
    await act(async () => {
      fireEvent.click(regenerate);
    });

    // Confirming the modal must POST with ?force=1 (bypasses the backend's
    // "already complete" early-return so the pipeline overwrites the PNG).
    const confirm = await screen.findByRole('button', { name: '确认重新生成' });
    await act(async () => {
      fireEvent.click(confirm);
    });

    await waitFor(() => expect(mocks.post).toHaveBeenCalledTimes(1));
    expect(mocks.post.mock.calls[0][0]).toContain('force=1');
  });
});
