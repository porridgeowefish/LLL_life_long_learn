// Projects API — TanStack Query hooks wrapping the backend endpoints.
// Mirrors the legacy frontend/legacy/js/api.js shape but with typed responses
// and proper cache invalidation.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import type {
  CreateProjectRequest,
  CreateSubprojectRequest,
  ProjectResponse,
  ProjectsListResponse,
  ZoneResponse,
} from '@/types/api';
import type { ZoneName } from '@/types/domain';

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
      qc.invalidateQueries({ queryKey: qk.health() });
    },
  });
}

export function useCreateSubproject(parentSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateSubprojectRequest) =>
      http.post<ProjectResponse>(
        `/api/projects/${encodeURIComponent(parentSlug)}/subprojects`,
        payload,
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.projects.all() });
      qc.invalidateQueries({ queryKey: qk.projects.detail(parentSlug) });
      qc.invalidateQueries({ queryKey: qk.projects.tree(parentSlug) });
    },
  });
}
