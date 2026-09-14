import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { subscribeToSSE } from '@/app/hooks/useSSE';
import { SSE_EVENTS } from '@/shared/lib/constants';

// AppShell owns this subscription. All activity summaries share the same
// durable progress stream, so one broad invalidation refreshes home and usage.
export function useActivityRefresh(): void {
  const queryClient = useQueryClient();

  useEffect(() => subscribeToSSE(SSE_EVENTS.learningActivityUpdated, () => {
    void queryClient.invalidateQueries({ queryKey: ['activity'] });
  }), [queryClient]);
}
