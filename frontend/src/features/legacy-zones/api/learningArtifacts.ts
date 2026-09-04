import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from '@/shared/client';
import { qk } from '@/shared/queryKeys';

export type AssessedLevel = 'unknown' | 'none' | 'basic' | 'working' | 'solid';
export type PrerequisiteStatus = 'ready' | 'weak' | 'missing';

export interface ProjectDraft {
  title: string;
  current: string;
  target: string;
  why: string;
  standard: string;
}

export interface PrerequisiteAssessment {
  id: string;
  title: string;
  assessedLevel: AssessedLevel;
  status: PrerequisiteStatus;
  summary?: string;
  impact: string;
  evidence: string;
  projectDraft?: ProjectDraft; // legacy iteration-04 assessment compatibility
}

export interface IntroAssessment {
  schemaVersion: number;
  baseline: string;
  prerequisites: PrerequisiteAssessment[];
}

export interface IntroSurveyQuestion {
  id: string;
  label: string;
  prompt: string;
  answer?: string;
}

export interface IntroSurvey {
  schemaVersion: number;
  updatedAt: string;
  questions: IntroSurveyQuestion[];
}

export interface ExplainPageEntry {
  id: string;
  title: string;
  file: string;
  kind: 'core' | 'followup';
  order: number;
  parentPageId: string | null;
  createdAt: string;
}

export interface ExplainManifest {
  schemaVersion: number;
  title: string;
  pages: ExplainPageEntry[];
  updatedAt: string;
}

export function useIntroAssessment(projectSlug: string) {
  return useQuery({
    queryKey: qk.files.raw(projectSlug, 'intro/assessment.json'),
    retry: false,
    queryFn: () =>
      http.get<IntroAssessment>(
        `/files/projects/${encodeURIComponent(projectSlug)}/intro/assessment.json`,
      ),
  });
}

export function useIntroSurvey(projectSlug: string) {
  return useQuery({
    queryKey: qk.files.raw(projectSlug, 'intro/survey.json'),
    retry: false,
    queryFn: () =>
      http.get<IntroSurvey>(
        `/files/projects/${encodeURIComponent(projectSlug)}/intro/survey.json`,
      ),
  });
}

export function useIntroOutput(projectSlug: string) {
  return useQuery({
    queryKey: qk.files.raw(projectSlug, 'intro/output.md'),
    retry: false,
    queryFn: () =>
      http.get<string>(
        `/files/projects/${encodeURIComponent(projectSlug)}/intro/output.md`,
        { rawText: true },
      ),
  });
}

export function useSaveIntroSurvey(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (survey: IntroSurvey) =>
      http.post<{ ok: boolean; bytes: number }>(
        `/files/projects/${encodeURIComponent(projectSlug)}/intro/survey.json`,
        JSON.stringify(survey, null, 2),
        { headers: { 'Content-Type': 'application/json' } },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.files.raw(projectSlug, 'intro/survey.json') });
    },
  });
}

export function useExplainManifest(projectSlug: string) {
  return useQuery({
    queryKey: qk.files.raw(projectSlug, 'explain/manifest.json'),
    retry: false,
    queryFn: () =>
      http.get<ExplainManifest>(
        `/files/projects/${encodeURIComponent(projectSlug)}/explain/manifest.json`,
      ),
  });
}

export function useExplainPage(projectSlug: string, file: string | undefined) {
  return useQuery({
    queryKey: file ? qk.files.raw(projectSlug, `explain/${file}`) : ['explain', projectSlug, 'missing-page'],
    enabled: !!file,
    queryFn: () =>
      http.get<string>(
        `/files/projects/${encodeURIComponent(projectSlug)}/explain/${file}`,
        { rawText: true },
      ),
  });
}
