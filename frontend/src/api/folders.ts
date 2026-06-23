// Folder layout API — TanStack Query hooks for the workspace-global project
// folder store. Mirrors the confusions/projects hook shape.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import type { Folder, FolderLayout } from '@/lib/folders';

// Normalize: tolerate a missing/null folders array or null slugOrder so the UI
// never crashes on a hand-edited or partial folders.json.
function normalize(data: unknown): FolderLayout {
  const d = (data ?? {}) as Partial<FolderLayout>;
  const folders = (d.folders ?? []).map((f) => ({
    id: f.id,
    name: f.name,
    slugOrder: f.slugOrder ?? [],
  })) as Folder[];
  return { folders };
}

export function useFolders() {
  return useQuery({
    queryKey: qk.folders.all(),
    queryFn: async () => normalize(await http.get<FolderLayout>('/api/folders')),
  });
}

/** Replace the whole folder layout (client computes the new state, server sanitizes). */
export function useSaveFolders() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (layout: FolderLayout) => http.put<FolderLayout>('/api/folders', layout),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.folders.all() }),
  });
}
