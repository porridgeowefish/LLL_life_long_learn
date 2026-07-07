import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { API_BASE } from '@/lib/constants';
import { ApiError } from './client';
import { extractSSEData } from '@/lib/parseAskAiStream';

export interface AskFrame {
  type: 'text' | 'thinking' | 'done' | 'error';
  content: string;
}

export interface AskProvider {
  id: string;
  kind: 'openai' | 'anthropic';
  name?: string;
  baseURL: string;
  apiKey: string; // masked ("••••") from GET; plaintext only on user input
  model: string;
  reasoning?: boolean;
  thinking?: boolean;
}

export interface AskConfig {
  default: string;
  searchEngine: 'google' | 'bing';
  providers: AskProvider[];
}

export interface StreamBody {
  providerId?: string;
  content: string;
  pageArtifactId?: string;
}

/**
 * Stream an Ask-AI turn. Uses raw fetch (POST + ReadableStream); the shared
 * `http` client parses JSON and cannot carry SSE. Frames are split out of the
 * byte stream by extractSSEData.
 */
export async function streamAskAi(
  projectSlug: string,
  confusionId: string,
  body: StreamBody,
  handlers: { onFrame: (f: AskFrame) => void; signal?: AbortSignal },
): Promise<void> {
  let res: Response;
  try {
    res = await fetch(
      `${API_BASE}/api/projects/${encodeURIComponent(projectSlug)}/confusions/${encodeURIComponent(confusionId)}/ask-stream`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: handlers.signal,
      },
    );
  } catch (err) {
    throw new ApiError(0, `network: ${(err as Error).message}`, null);
  }
  if (!res.ok || !res.body) {
    let parsed: unknown = null;
    let message = `ask-stream failed: ${res.status}`;
    try {
      const text = await res.text();
      parsed = text ? JSON.parse(text) : null;
      if (
        parsed &&
        typeof parsed === 'object' &&
        'error' in parsed &&
        typeof (parsed as { error: unknown }).error === 'string'
      ) {
        message = (parsed as { error: string }).error;
      }
    } catch {
      // Fall back to the status-only message.
    }
    throw new ApiError(res.status, message, parsed);
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const { payloads, rest } = extractSSEData(buf);
    buf = rest;
    for (const payload of payloads) {
      try {
        const raw = JSON.parse(payload) as Partial<AskFrame>;
        // Normalize: the declared type requires `content: string`; a `done`
        // frame may arrive without one. Coerce missing content to ''.
        const frame: AskFrame = { type: raw.type ?? 'text', content: raw.content ?? '' };
        handlers.onFrame(frame);
      } catch {
        /* skip malformed */
      }
    }
  }
}

/** Trigger the close->summarize flow (backend returns 202 immediately). */
export async function summarizeAsk(projectSlug: string, confusionId: string): Promise<void> {
  await fetch(
    `${API_BASE}/api/projects/${encodeURIComponent(projectSlug)}/confusions/${encodeURIComponent(confusionId)}/ask/summarize`,
    { method: 'POST' },
  );
}

export const qkAskAi = {
  settings: () => ['ask-ai', 'settings'] as const,
};

export function useAskAiSettings() {
  return useQuery<AskConfig>({
    queryKey: qkAskAi.settings(),
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/api/settings/ask-ai`);
      if (!res.ok) throw new ApiError(res.status, 'settings failed', null);
      return (await res.json()) as AskConfig;
    },
  });
}

export function useSaveAskAiSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (cfg: AskConfig) => {
      const res = await fetch(`${API_BASE}/api/settings/ask-ai`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cfg),
      });
      if (!res.ok) throw new ApiError(res.status, 'save failed', null);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: qkAskAi.settings() }),
  });
}

export function useProbeAskAi() {
  return useMutation({
    mutationFn: async (vars: { providerId?: string; inline?: AskProvider }) => {
      const res = await fetch(`${API_BASE}/api/settings/ask-ai/probe`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(vars),
      });
      if (!res.ok) throw new ApiError(res.status, 'probe failed', null);
      return (await res.json()) as { ok: boolean; error?: string };
    },
  });
}
