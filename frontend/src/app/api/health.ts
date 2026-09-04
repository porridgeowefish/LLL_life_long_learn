// Health API — used by the ConnectionBadge to show online/offline state.
// Polled every 30s; also invalidated after mutations that change stats.

import { useQuery } from '@tanstack/react-query';

import { http } from '@/shared/client';
import { qk } from '@/shared/queryKeys';
import { HEALTH_POLL_INTERVAL_MS } from '@/shared/lib/constants';
import type { HealthResponse } from '@/shared/types/api';

export function useHealth() {
  return useQuery({
    queryKey: qk.health(),
    queryFn: () => http.get<HealthResponse>('/api/health'),
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });
}
