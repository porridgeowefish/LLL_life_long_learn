import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';

export interface LearnerPreferences {
  path: string;
  content: string;
  maxBytes: number;
}

export function usePreferences() {
  return useQuery({
    queryKey: qk.preferences.file(),
    queryFn: () => http.get<LearnerPreferences>('/api/preferences'),
  });
}

export function useSavePreferences() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (content: string) => http.put<{ ok: boolean; path: string; bytes: number }>(
      '/api/preferences',
      content,
      { headers: { 'Content-Type': 'text/markdown; charset=utf-8' } },
    ),
    onSuccess: (_result, content) => {
      client.setQueryData<LearnerPreferences>(qk.preferences.file(), (current) => current ? { ...current, content } : current);
      void client.invalidateQueries({ queryKey: qk.preferences.file() });
    },
  });
}
