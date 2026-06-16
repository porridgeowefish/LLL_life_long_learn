import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';

export type InfographicStatus = 'missing' | 'pending' | 'running' | 'complete' | 'failed';

export interface InfographicState {
  status: InfographicStatus;
  url?: string;
  error?: string;
}

export function useExplainInfographic(projectSlug: string | undefined) {
  return useQuery({
    queryKey: ['explain', 'infographic', projectSlug],
    enabled: !!projectSlug,
    retry: false,
    refetchInterval: (query) => {
      const data = query.state.data as InfographicState | undefined;
      return data?.status === 'pending' || data?.status === 'running' ? 3000 : false;
    },
    queryFn: async () => {
      const res = await http.get<InfographicState>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/explain/infographic`,
      );
      return res;
    },
  });
}

export function useRequestExplainInfographic() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (projectSlug: string) =>
      http.post<InfographicState>(
        `/api/projects/${encodeURIComponent(projectSlug)}/explain/infographic`,
      ),
    onSuccess: (_data, projectSlug) => {
      qc.invalidateQueries({ queryKey: ['explain', 'infographic', projectSlug] });
    },
  });
}
