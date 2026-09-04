// Flashcard API — list, grade.
// Endpoints: GET /api/projects/{id}/summary/flashcards
//            POST /api/projects/{id}/summary/flashcards/grade

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from '@/shared/client';

export interface Flashcard {
  id: string;
  front: string;
  back: string;
  category?: 'concept' | 'relationship' | 'boundary' | 'misconception' | 'transfer' | string;
  sourceRefs?: string[];
  generatedReason?: string;
  zone?: string;
  sourceId?: string;
}

export interface CardProgress {
  cardId: string;
  timesSeen: number;
  timesRight: number;
  lastGrade: string;
}

export type Grade = 'forgot' | 'fuzzy' | 'got-it' | 'easy';

interface FlashcardsResponse {
  flashcards: Flashcard[] | null;
  progress: CardProgress[] | null;
}

/** Get flashcards + progress for a project. */
export function useFlashcards(projectSlug: string | undefined) {
  return useQuery({
    queryKey: ['summary', 'flashcards', projectSlug],
    enabled: !!projectSlug,
    queryFn: async () => {
      const res = await http.get<FlashcardsResponse>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/summary/flashcards`,
      );
      return res;
    },
  });
}

/** Grade a flashcard. */
export function useGradeFlashcard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectSlug,
      cardId,
      grade,
    }: {
      projectSlug: string;
      cardId: string;
      grade: Grade;
    }) =>
      http.post(
        `/api/projects/${encodeURIComponent(projectSlug)}/summary/flashcards/grade`,
        { cardId, grade, eventId: crypto.randomUUID() },
      ),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['summary', 'flashcards', vars.projectSlug] });
    },
  });
}
