import { useQuery } from '@tanstack/react-query';

import { http } from '@/shared/client';

export interface TeacherUsageTurn {
  responseId: string;
  providerId?: string;
  inputTokens: number;
  outputTokens: number;
  occurredAt: string;
}

export interface TeacherUsageConversation {
  projectSlug: string;
  title: string;
  conversationId: string;
  inputTokens: number;
  outputTokens: number;
  updatedAt: string;
  turns: TeacherUsageTurn[];
}

export interface TeacherUsagePage {
  page: number;
  pageSize: number;
  total: number;
  conversations: TeacherUsageConversation[];
}

export function useTeacherUsage(page = 1) {
  return useQuery({
    queryKey: ['teacher-usage', page],
    queryFn: () => http.get<TeacherUsagePage>(`/api/usage/teacher?page=${page}&pageSize=20`),
  });
}
