// Health API — used by the ConnectionBadge to show online/offline state.
// Polled every 30s; also invalidated after mutations that change stats.

import { useQuery } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import { HEALTH_POLL_INTERVAL_MS } from '@/lib/constants';
import type { HealthResponse } from '@/types/api';

export function useHealth() {
  return useQuery({
    queryKey: qk.health(),
    queryFn: () => http.get<HealthResponse>('/api/health'),
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });
}
