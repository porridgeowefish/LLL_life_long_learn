// Generic project file read/write — used by MarkdownEditor and any component
// that needs to load/save files under a project's directory tree.
// Backend: GET/POST /files/projects/{slug}/{dir}/{filename}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';

/** Read a raw text file from a project directory. */
export function useProjectFile(slug: string | undefined, relPath: string) {
  return useQuery({
    queryKey: slug ? qk.files.raw(slug, relPath) : ['files', '__missing__'],
    enabled: !!slug && !!relPath,
    queryFn: () =>
      http.get<string>(
        `/files/projects/${encodeURIComponent(slug!)}/${relPath}`,
        { rawText: true },
      ),
  });
}

/** Write a raw text file to a whitelisted project directory. */
export function useFileWrite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ slug, relPath, body }: { slug: string; relPath: string; body: string }) =>
      http.post<{ ok: boolean; bytes: number }>(
        `/files/projects/${encodeURIComponent(slug)}/${relPath}`,
        body,
        { headers: { 'Content-Type': 'text/plain' } },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.files.raw(vars.slug, vars.relPath) });
    },
  });
}
