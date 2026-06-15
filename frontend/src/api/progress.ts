import { useQuery } from '@tanstack/react-query';

import { http } from './client';

export interface GrowthEvent {
  id: string;
  sourceType: string;
  sourceId: string;
  attemptId: string;
  difficulty: number;
  outcome: string;
  delta: number;
  policyVersion: string;
  createdAt: string;
}

export interface ProjectProgress {
  total: number;
  bySource: Record<string, number>;
  recentEvents: GrowthEvent[];
  updatedAt: string;
}

export function useProjectProgress(projectSlug: string) {
  return useQuery({
    queryKey: ['progress', projectSlug],
    queryFn: async () => {
      const res = await http.get<{ progress: ProjectProgress }>(
        `/api/projects/${encodeURIComponent(projectSlug)}/progress`,
      );
      return res.progress;
    },
  });
}
