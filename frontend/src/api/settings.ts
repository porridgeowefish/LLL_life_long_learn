import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import type { AgentRuntimeID } from '@/types/domain';
import type { AgentRuntimeSettingsResponse } from '@/types/api';

export function useAgentRuntimeSettings() {
  return useQuery({
    queryKey: qk.settings.agentRuntime(),
    queryFn: () => http.get<AgentRuntimeSettingsResponse>('/api/settings/agent-runtime'),
  });
}

export function useUpdateAgentRuntime() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (selected: AgentRuntimeID) =>
      http.put<AgentRuntimeSettingsResponse>('/api/settings/agent-runtime', { selected }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.settings.agentRuntime() });
      qc.invalidateQueries({ queryKey: qk.health() });
    },
  });
}
