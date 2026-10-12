import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { ApiError, http } from '@/shared/client';
import { extractSSEData } from '@/shared/lib/parseAskAiStream';

/** Exact learner text of the "我讲完了" button; the backend prompt keys on it. */
export const FEYNMAN_FINISH = '我讲完了，请对照复习稿点评：哪些讲清楚了、哪些讲漏了、哪些讲错了。';

export interface FeynmanMessage { id: string; role: 'learner' | 'teacher'; content: string; status: 'completed' | 'interrupted' | 'failed'; createdAt: string }
export interface FeynmanSession { sessionId: string; reviewVersionId?: string; messages: FeynmanMessage[]; createdAt: string; updatedAt: string }

export const feynmanKeys = { sessions: (slug: string) => ['learning-workspace', slug, 'feynman'] as const };

export function useFeynmanSessions(slug: string) {
  return useQuery({ queryKey: feynmanKeys.sessions(slug), queryFn: async () => (await http.get<{ sessions: FeynmanSession[] }>(`/api/projects/${encodeURIComponent(slug)}/feynman/sessions`)).sessions, enabled: Boolean(slug) });
}

export function useCreateFeynmanSession(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => http.post<FeynmanSession>(`/api/projects/${encodeURIComponent(slug)}/feynman/sessions`, {}),
    onSuccess: () => client.invalidateQueries({ queryKey: feynmanKeys.sessions(slug) }),
  });
}

/** Stream one novice reply. An empty content opens a brand-new round. */
export async function streamFeynman(slug: string, sessionId: string, body: { content: string; providerId?: string }, onText: (delta: string) => void, signal?: AbortSignal): Promise<void> {
  let response: Response;
  try {
    response = await fetch(`/api/projects/${encodeURIComponent(slug)}/feynman/sessions/${encodeURIComponent(sessionId)}/stream`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal });
  } catch (cause) {
    throw new ApiError(0, `network: ${(cause as Error).message}`, null);
  }
  if (!response.ok || !response.body) {
    let message = `讲解对话失败（${response.status}）`;
    try { const parsed = await response.json() as { error?: unknown }; if (typeof parsed.error === 'string') message = parsed.error; } catch { /* status only */ }
    throw new ApiError(response.status, message, null);
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let failure = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const { payloads, rest } = extractSSEData(buffer);
    buffer = rest;
    for (const payload of payloads) {
      try {
        const frame = JSON.parse(payload) as { type?: string; content?: string };
        if (frame.type === 'text' && frame.content) onText(frame.content);
        if (frame.type === 'error') failure = frame.content || '回答中断';
      } catch { /* skip malformed frame */ }
    }
  }
  if (failure) throw new ApiError(502, failure, null);
}
