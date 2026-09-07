// Projects API — TanStack Query hooks wrapping the backend endpoints.
// Provides the project API shape with typed responses.
// and proper cache invalidation.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from '@/shared/client';
import { qk } from '@/shared/queryKeys';
import type {
  CreateProjectRequest,
  DeleteProjectResponse,
  DisciplineOverviewGenerationResponse,
  DisciplineOverviewResponse,
  DisciplineTopicsResponse,
  DisciplineLearningPlanResponse,
  ProjectTypeAdviceRequest,
  ProjectTypeAdviceResponse,
  ProjectResponse,
  ProjectsListResponse,
  ZoneResponse,
} from '@/shared/types/api';
import type { ZoneName } from '@/shared/types/domain';

export function useProjects() {
  return useQuery({
    queryKey: qk.projects.all(),
    queryFn: () => http.get<ProjectsListResponse>('/api/projects'),
  });
}

export function useProject(slug: string | undefined) {
  return useQuery({
    queryKey: slug ? qk.projects.detail(slug) : ['projects', '__missing__'],
    enabled: !!slug,
    queryFn: () => http.get<ProjectResponse>(`/api/projects/${encodeURIComponent(slug!)}`),
  });
}

export function useProjectTree(slug: string | undefined) {
  return useQuery({
    queryKey: slug ? qk.projects.tree(slug) : ['projects', '__missing__', 'tree'],
    enabled: !!slug,
    queryFn: () =>
      http.get<ProjectsListResponse>(`/api/projects/${encodeURIComponent(slug!)}/tree`),
  });
}

export function useZone(slug: string | undefined, zone: ZoneName | undefined) {
  return useQuery({
    queryKey: slug && zone ? qk.projects.zone(slug, zone) : ['projects', '__missing__', 'zones'],
    enabled: !!slug && !!zone,
    queryFn: () =>
      http.get<ZoneResponse>(
        `/api/projects/${encodeURIComponent(slug!)}/zones/${zone!}`,
      ),
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateProjectRequest) =>
      http.post<ProjectResponse>('/api/projects', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.projects.all() });
      qc.invalidateQueries({ queryKey: qk.folders.all() });
      qc.invalidateQueries({ queryKey: qk.health() });
    },
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slug: string) =>
      http.delete<DeleteProjectResponse>(`/api/projects/${encodeURIComponent(slug)}`),
    onSuccess: (_response, slug) => {
      qc.removeQueries({ predicate: (query) => query.queryKey.includes(slug) });
      qc.invalidateQueries({ queryKey: qk.projects.all() });
      qc.invalidateQueries({ queryKey: qk.folders.all() });
      qc.invalidateQueries({ queryKey: qk.sessions.recent() });
      qc.invalidateQueries({ queryKey: qk.sessions.active() });
      qc.invalidateQueries({ queryKey: qk.health() });
    },
  });
}

export function useProjectTypeAdvice() {
  return useMutation({
    mutationFn: (payload: ProjectTypeAdviceRequest) =>
      http.post<ProjectTypeAdviceResponse>('/api/project-type-advice', payload),
  });
}

export function useDisciplineOverview(slug: string | undefined) {
  return useQuery({
    queryKey: ['projects', slug ?? '__missing__', 'discipline-overview'],
    enabled: !!slug,
    queryFn: () =>
      http.get<DisciplineOverviewResponse>(
        `/api/projects/${encodeURIComponent(slug!)}/discipline-overview`,
      ),
  });
}

export function useDisciplineTopics(slug: string | undefined) {
  return useQuery({
    queryKey: ['projects', slug ?? '__missing__', 'discipline-topics'],
    enabled: !!slug,
    queryFn: () =>
      http.get<DisciplineTopicsResponse>(
        `/api/projects/${encodeURIComponent(slug!)}/discipline-topics`,
      ),
  });
}

export function useDisciplineLearningPlan(slug: string | undefined) {
  return useQuery({
    queryKey: ['projects', slug ?? '__missing__', 'discipline-learning-plan'],
    enabled: !!slug,
    queryFn: () =>
      http.get<DisciplineLearningPlanResponse>(
        `/api/projects/${encodeURIComponent(slug!)}/learning-plan`,
      ),
  });
}

export function useSaveDisciplineLearningPlan(slug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (plan: DisciplineLearningPlanResponse) =>
      http.put<DisciplineLearningPlanResponse>(
        `/api/projects/${encodeURIComponent(slug)}/learning-plan`,
        plan,
      ),
    onSuccess: (plan) => {
      qc.setQueryData(['projects', slug, 'discipline-learning-plan'], plan);
    },
  });
}

export function useGenerateDisciplineOverview(slug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      http.post<DisciplineOverviewGenerationResponse>(
        `/api/projects/${encodeURIComponent(slug)}/discipline-overview/generate`,
        {},
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.sessions.active() });
      qc.invalidateQueries({ queryKey: qk.sessions.recent() });
    },
  });
}
