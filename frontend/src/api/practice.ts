import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';

export type PracticeType =
  | 'true-false'
  | 'single-choice'
  | 'multiple-choice'
  | 'short-answer'
  | 'essay'
  | 'code';

export type PracticeAnswer = string | boolean | string[];

export interface PracticeOption {
  id: string;
  text: string;
}

export interface PracticeTask {
  id: string;
  question: string;
  type: PracticeType;
  difficulty: number;
  options?: PracticeOption[];
  sourceRefs?: string[];
}

export interface Submission {
  taskId: string;
  answer: PracticeAnswer;
  selfAssess: number;
}

export interface ObjectiveResult {
  taskId: string;
  answer: PracticeAnswer;
  correct: boolean;
  correctAnswer: PracticeAnswer;
  explanation: string;
  lockedAt: string;
  growthDelta: number;
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

export function isObjectiveTask(task: PracticeTask) {
  return task.type === 'true-false' || task.type === 'single-choice' || task.type === 'multiple-choice';
}

export function usePracticeTasks(
  projectSlug: string | undefined,
  options?: { refetchInterval?: number | false },
) {
  return useQuery({
    queryKey: ['practice', 'tasks', projectSlug],
    enabled: !!projectSlug,
    refetchInterval: options?.refetchInterval,
    queryFn: () =>
      http.get<{
        schemaVersion?: number;
        setId?: string;
        tasks: PracticeTask[] | null;
        generated: boolean;
        generatedAt?: string;
      }>(`/api/projects/${encodeURIComponent(projectSlug!)}/practice/tasks`),
  });
}

export function useCreatePracticeAttempt() {
  return useMutation({
    mutationFn: (projectSlug: string) =>
      http.post<{ attempt: number; setId: string }>(
        `/api/projects/${encodeURIComponent(projectSlug)}/practice/attempts`,
        {},
      ),
  });
}

export function useCheckObjective() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      attempt,
      taskId,
      answer,
    }: {
      projectSlug: string;
      attempt: number;
      taskId: string;
      answer: PracticeAnswer;
    }) =>
      http.post<{ result: ObjectiveResult; growthDelta: number }>(
        `/api/projects/${encodeURIComponent(projectSlug)}/practice/attempts/${attempt}/objective/${encodeURIComponent(taskId)}/check`,
        { answer },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['progress', vars.projectSlug] });
    },
  });
}

export function useSubmitPracticeAttempt() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      attempt,
      submissions,
    }: {
      projectSlug: string;
      attempt: number;
      submissions: Submission[];
    }) =>
      http.post<{ attempt: number; saved: number; growthDelta: number }>(
        `/api/projects/${encodeURIComponent(projectSlug)}/practice/attempts/${attempt}/submit`,
        { submissions },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['progress', vars.projectSlug] });
    },
  });
}

export function usePracticeEvaluation(projectSlug: string | undefined, attempt: number) {
  return useQuery({
    queryKey: ['practice', 'evaluation', projectSlug, attempt],
    enabled: !!projectSlug && attempt > 0,
    retry: false,
    refetchInterval: 5000,
    queryFn: async () => {
      const res = await http.get<{ evaluation: Evaluation }>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/practice/evaluation?attempt=${attempt}`,
      );
      return res.evaluation;
    },
  });
}
