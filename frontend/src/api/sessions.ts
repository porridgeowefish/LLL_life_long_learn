// Sessions API — listing, detail, follow-up, cancel.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import type { FollowUpRequest } from '@/types/api';
import type { Session } from '@/types/domain';

interface SessionsListParams {
  recent?: boolean;
  active?: boolean;
  limit?: number;
}

export function useSessions(opts: SessionsListParams = {}) {
  return useQuery({
    queryKey: opts.active
      ? qk.sessions.active()
      : qk.sessions.recent(opts.limit),
    queryFn: () => {
      const params = new URLSearchParams();
      if (opts.recent) params.set('recent', 'true');
      if (opts.active) params.set('active', 'true');
      if (opts.limit) params.set('limit', String(opts.limit));
      const qs = params.toString();
      return http.get<{ sessions: Session[] }>(
        `/api/sessions${qs ? '?' + qs : ''}`,
      );
    },
  });
}

export function useRecentSessions(limit = 10) {
  return useSessions({ recent: true, limit });
}

export function useActiveSessions() {
  return useSessions({ active: true });
}

export function useSession(id: string | undefined) {
  return useQuery({
    queryKey: id ? qk.sessions.detail(id) : ['sessions', '__missing__'],
    enabled: !!id,
    queryFn: () =>
      http.get<{ session: Session }>(`/api/sessions/${encodeURIComponent(id!)}`),
  });
}

export function useFollowUp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: FollowUpRequest }) =>
      http.post<{ sessionId: string; runDir: string }>(
        `/api/sessions/${encodeURIComponent(id)}/follow-up`,
        payload,
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.sessions.detail(vars.id) });
      qc.invalidateQueries({ queryKey: qk.sessions.active() });
    },
  });
}

export function useCancelSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      http.post<{ ok: boolean }>(`/api/sessions/${encodeURIComponent(id)}/cancel`),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: qk.sessions.detail(id) });
      qc.invalidateQueries({ queryKey: qk.sessions.active() });
    },
  });
}
