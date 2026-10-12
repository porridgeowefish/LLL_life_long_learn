import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from '@/shared/client';

export interface PracticeItemMark { correct: boolean; expected: string[] }
export interface PracticeResult { versionId: string; answers: Record<string, string[]>; items: Record<string, PracticeItemMark>; score: number; total: number; submittedAt: string }

const key = (slug: string) => ['learning-workspace', slug, 'assets', 'practice', 'results'] as const;
const base = (slug: string) => `/api/projects/${encodeURIComponent(slug)}/assets/practice/results`;

export function usePracticeResults(slug: string) {
  return useQuery({ queryKey: key(slug), queryFn: async () => (await http.get<{ results: Record<string, PracticeResult> }>(base(slug))).results ?? {}, enabled: Boolean(slug) });
}

export function useSavePracticeResult(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (result: Omit<PracticeResult, 'submittedAt'>) => http.put<PracticeResult>(`${base(slug)}/${encodeURIComponent(result.versionId)}`, result),
    onSuccess: (saved, sent) => client.setQueryData<Record<string, PracticeResult>>(key(slug), (current) => ({ ...(current ?? {}), [sent.versionId]: { ...sent, ...saved } })),
  });
}

export function useRetakePractice(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (versionId: string) => http.delete(`${base(slug)}/${encodeURIComponent(versionId)}`),
    onSuccess: (_data, versionId) => client.setQueryData<Record<string, PracticeResult>>(key(slug), (current) => {
      const next = { ...(current ?? {}) };
      delete next[versionId];
      return next;
    }),
  });
}
