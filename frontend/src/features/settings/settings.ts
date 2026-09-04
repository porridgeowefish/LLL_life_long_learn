import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from '@/shared/client';
import { qk } from '@/shared/queryKeys';
import type { AgentRuntimeID } from '@/shared/types/domain';
import type { AgentRuntimeSettingsResponse } from '@/shared/types/api';

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
