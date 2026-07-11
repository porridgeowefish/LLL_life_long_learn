import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';

export interface LearningEvent {
  id: string;
  sourceType: string;
  sourceId: string;
  activityDelta?: number;
  delta: number;
  title?: string;
  detail?: string;
  outcome?: string;
  createdAt: string;
  projectSlug: string;
  projectTitle: string;
}

export interface ActivityDay {
  date: string;
  activity: number;
  growth: number;
  actions: number;
  events: LearningEvent[];
}

export interface ActivitySummary {
  rangeStart: string;
  rangeEnd: string;
  activeDays: number;
  currentStreak: number;
  longestStreak: number;
  totalActions: number;
  totalGrowth: number;
  days: ActivityDay[];
}

export function useActivity(weeks: 26 | 52, project = '') {
  return useQuery({
    queryKey: ['activity', weeks, project || 'all'],
    queryFn: async () => {
      const query = new URLSearchParams({ weeks: String(weeks) });
      if (project) query.set('project', project);
      const res = await http.get<{ summary: ActivitySummary }>(`/api/activity?${query}`);
      return res.summary;
    },
    refetchOnMount: 'always',
  });
}

export function useRecordReading(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (event: { id: string; sourceId: string; title: string; detail: string; activityDelta: number }) =>
      http.post(`/api/projects/${encodeURIComponent(projectSlug)}/activity`, event),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['activity'] }),
  });
}
