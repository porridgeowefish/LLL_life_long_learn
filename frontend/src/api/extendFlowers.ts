import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  createEmptyFlower,
  normalizeFlower,
  parseFlowerJson,
  serializeFlower,
  type KnowledgeFlower,
} from '@/components/feature/extend/flowerData';
import { type ProjectMeta } from '@/types/domain';

import { ApiError, http } from './client';
import { qk } from './queryKeys';

export const FLOWER_REL_PATH = 'extend/flower.json';

async function readFlower(projectSlug: string, projectTitle: string): Promise<KnowledgeFlower> {
  try {
    const raw = await http.get<string>(
      `/files/projects/${encodeURIComponent(projectSlug)}/${FLOWER_REL_PATH}`,
      { rawText: true },
    );
    return parseFlowerJson(raw, projectSlug, projectTitle);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return createEmptyFlower(projectSlug, projectTitle);
    }
    throw error;
  }
}

export function useExtendFlower(projectSlug: string | undefined, projectTitle: string) {
  return useQuery({
    queryKey: projectSlug ? qk.files.raw(projectSlug, FLOWER_REL_PATH) : ['files', '__missing__', FLOWER_REL_PATH],
    enabled: !!projectSlug,
    queryFn: () => readFlower(projectSlug!, projectTitle),
  });
}

export function useProjectFlowers(projects: ReadonlyArray<ProjectMeta>) {
  return useQueries({
    queries: projects.map((project) => ({
      queryKey: qk.files.raw(project.slug, FLOWER_REL_PATH),
      queryFn: () => readFlower(project.slug, project.title),
      staleTime: 10_000,
    })),
  });
}

export function useSaveExtendFlower() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ flower }: { flower: KnowledgeFlower }) => {
      const normalized = normalizeFlower(flower, flower.projectSlug, flower.title);
      return http.post<{ ok: boolean; bytes: number }>(
        `/files/projects/${encodeURIComponent(flower.projectSlug)}/${FLOWER_REL_PATH}`,
        serializeFlower({
          ...normalized,
          updatedAt: new Date().toISOString(),
        }),
        { headers: { 'Content-Type': 'text/plain' } },
      );
    },
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: qk.files.raw(vars.flower.projectSlug, FLOWER_REL_PATH) });
    },
  });
}
