import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';

export type InfographicStatus = 'missing' | 'pending' | 'running' | 'complete' | 'failed';

export interface InfographicState {
  status: InfographicStatus;
  url?: string;
  error?: string;
  updatedAt?: string;
}

export interface RequestInfographicArgs {
  projectSlug: string;
  /** force=true bypasses the backend's "already complete" check and regenerates. */
  force?: boolean;
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
    mutationFn: ({ projectSlug, force }: RequestInfographicArgs) => {
      const url =
        `/api/projects/${encodeURIComponent(projectSlug)}/explain/infographic` +
        (force ? '?force=1' : '');
      return http.post<InfographicState>(url);
    },
    onSuccess: (_data, { projectSlug }) => {
      qc.invalidateQueries({ queryKey: ['explain', 'infographic', projectSlug] });
    },
  });
}
