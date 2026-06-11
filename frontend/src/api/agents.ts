// Agents API — registry listing + invocation. Backend endpoint at
// POST /api/agents/:id/invoke returns 201 + { session, runDir } before
// Claude exits (asynchronous). The SSE stream delivers progress.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import type { AgentsListResponse, InvokeAgentRequest } from '@/types/api';
import type { Session } from '@/types/domain';

interface InvokeAgentResponse {
  session: Session;
  runDir: string;
}

export function useAgents() {
  return useQuery({
    queryKey: qk.agents.all(),
    queryFn: () => http.get<AgentsListResponse>('/api/agents'),
  });
}

export function useInvokeAgent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ agentId, payload }: { agentId: string; payload: InvokeAgentRequest }) =>
      http.post<InvokeAgentResponse>(
        `/api/agents/${encodeURIComponent(agentId)}/invoke`,
        payload,
      ),
    onSuccess: () => {
      // Optimistic: the new session is "active"; refresh active list.
      qc.invalidateQueries({ queryKey: qk.sessions.active() });
      qc.invalidateQueries({ queryKey: qk.sessions.recent() });
    },
  });
}
