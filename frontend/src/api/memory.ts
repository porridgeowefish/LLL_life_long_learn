// Memory API — read/write learner-owned files under projects/<slug>/memory/.
// Backend endpoint: GET /files/projects/{id}/... and POST /files/projects/{id}/
// (restricted to memory/ paths, see backend-go/internal/server/routes_projects.go:257).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';

interface MemoryFile {
  slug: string;
  filename: string; // e.g. "project-memory.md"
}

export function useMemoryFile(slug: string | undefined, filename: string) {
  return useQuery({
    queryKey: slug ? qk.memory.file(slug, filename) : ['memory', '__missing__'],
    enabled: !!slug,
    queryFn: async () => {
      // Memory files are plain UTF-8 text — fetch as raw text, not JSON.
      return http.get<string>(
        `/files/projects/${encodeURIComponent(slug!)}/memory/${encodeURIComponent(filename)}`,
        { rawText: true },
      );
    },
  });
}

export function useSaveMemory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ slug, filename, body }: MemoryFile & { body: string }) =>
      http.post<{ ok: boolean; bytes: number }>(
        `/files/projects/${encodeURIComponent(slug)}/memory/${encodeURIComponent(filename)}`,
        body,
        { headers: { 'Content-Type': 'text/plain' } },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.memory.file(vars.slug, vars.filename) });
    },
  });
}
