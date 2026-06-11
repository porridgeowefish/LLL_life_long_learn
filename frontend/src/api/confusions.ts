// Confusion CRUD API — explain/confusions.json backed.
// Endpoints: GET/POST /api/projects/{id}/confusions,
//            PATCH/DELETE /api/projects/{id}/confusions/{confusionId}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';

export type ConfusionState = 'open' | 'asked' | 'resolved' | 'deleted';

export interface Confusion {
  id: string;
  sourceArtifactId?: string;
  paragraphId?: string;
  charStart: number;
  charEnd: number;
  quoteSnapshot: string;
  notes?: string;
  state: ConfusionState;
  createdAt: string;
}

interface ListResponse {
  confusions: Confusion[];
}

interface CreateResponse {
  confusion: Confusion;
}

/** List confusions, optionally filtered by state. */
export function useConfusions(projectSlug: string | undefined, stateFilter?: ConfusionState) {
  return useQuery({
    queryKey: qk.confusions.all(projectSlug ?? '__none__'),
    enabled: !!projectSlug,
    queryFn: async () => {
      const params = stateFilter ? `?state=${stateFilter}` : '';
      const res = await http.get<ListResponse>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/confusions${params}`,
      );
      return res.confusions;
    },
  });
}

/** Create a new confusion marker. */
export function useCreateConfusion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      confusion,
    }: {
      projectSlug: string;
      confusion: Omit<Confusion, 'id' | 'state' | 'createdAt'>;
    }) =>
      http.post<CreateResponse>(
        `/api/projects/${encodeURIComponent(projectSlug)}/confusions`,
        confusion,
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.confusions.all(vars.projectSlug) });
    },
  });
}

/** Patch a confusion (notes, state). */
export function useUpdateConfusion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      confusionId,
      patch,
    }: {
      projectSlug: string;
      confusionId: string;
      patch: Partial<Pick<Confusion, 'notes' | 'state'>>;
    }) =>
      http.patch<CreateResponse>(
        `/api/projects/${encodeURIComponent(projectSlug)}/confusions/${confusionId}`,
        patch,
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.confusions.all(vars.projectSlug) });
    },
  });
}

/** Delete (soft) a confusion. */
export function useDeleteConfusion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      confusionId,
    }: {
      projectSlug: string;
      confusionId: string;
    }) =>
      http.delete(
        `/api/projects/${encodeURIComponent(projectSlug)}/confusions/${confusionId}`,
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.confusions.all(vars.projectSlug) });
    },
  });
}
