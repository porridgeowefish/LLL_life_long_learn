// Practice API — task listing, batch submit, evaluation reading.
// Endpoints under /api/projects/{id}/practice/

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';

export interface PracticeTask {
  id: string;
  question: string;
  type: 'short-answer' | 'essay' | 'code';
}

export interface Submission {
  taskId: string;
  answer: string;
  selfAssess: number; // 1-5
}

export interface EvaluationResult {
  taskId: string;
  score: number;
  feedback: string;
  evidence: string;
  passed: boolean;
}

export interface Evaluation {
  attempt: number;
  results: EvaluationResult[];
  overallScore: number;
  generatedAt: string;
}

/** Get practice tasks for a project. */
export function usePracticeTasks(
  projectSlug: string | undefined,
  options?: { refetchInterval?: number | false },
) {
  return useQuery({
    queryKey: ['practice', 'tasks', projectSlug],
    enabled: !!projectSlug,
    refetchInterval: options?.refetchInterval,
    queryFn: async () => {
      const res = await http.get<{ tasks: PracticeTask[] | null; generated: boolean; generatedAt?: string }>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/practice/tasks`,
      );
      return res;
    },
  });
}

/** Batch submit practice answers. */
export function useSubmitPractice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      submissions,
    }: {
      projectSlug: string;
      submissions: Submission[];
    }) =>
      http.post<{ attempt: number; saved: number }>(
        `/api/projects/${encodeURIComponent(projectSlug)}/practice/submit`,
        { submissions },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['practice', 'tasks', vars.projectSlug] });
    },
  });
}

/** Get evaluation for an attempt. */
export function usePracticeEvaluation(projectSlug: string | undefined, attempt: number) {
  return useQuery({
    queryKey: ['practice', 'evaluation', projectSlug, attempt],
    enabled: !!projectSlug && attempt > 0,
    queryFn: async () => {
      const res = await http.get<{ evaluation: Evaluation }>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/practice/evaluation?attempt=${attempt}`,
      );
      return res.evaluation;
    },
  });
}
