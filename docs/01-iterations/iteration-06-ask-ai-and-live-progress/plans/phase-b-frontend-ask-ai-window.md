# Iteration 06 — Phase B (Frontend): Ask-AI Window — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the frontend half of Ask-AI — the provider settings UI, the streaming fetch + parser, a Zustand store, an anchored floating window (streaming + thinking + full markdown + multi-turn + provider switch + stop + browser search), the text-selection trigger in ExplainReader, and the sidebar (hover summary + read-only reopen) in ConfusionPanel.

**Architecture:** A new `api/askAi.ts` does request-scoped streaming via raw `fetch` + `ReadableStream` (the shared `http` client parses JSON, so it can't carry SSE); a pure `extractSSEData` splitter is unit-tested in isolation. A Zustand `useAskAiStore` holds the live exchange (`active` vs `review` modes). `AskAiPanel` is a `position: fixed` window anchored to the selection with viewport flipping, rendering each assistant message with `useMarkdown` (GFM+KaTeX+Mermaid) and a collapsible thinking section. ExplainReader adds a "问 AI" button that creates a confusion and opens the panel; ConfusionPanel shows a hover summary (state-dependent) and reopens `asked` confusions read-only. Closing an active panel with replies triggers the backend summarize endpoint.

**Tech Stack:** React 18 + TypeScript, TanStack Query, Zustand, `@radix-ui/react-icons`, Vitest + @testing-library/react. Reuses `useMarkdown`, `Modal`/`Button` primitives, `subscribeToSSE`, `SSE_EVENTS.confusionUpdated`, `qk.confusions.all`.

**Depends on:** Phase B backend plan (`2026-07-06-iteration-06-phase-b-backend-ask-ai-api.md`) — its endpoints and the `Confusion.ask` shape.

## Global Constraints

- The Ask-AI stream is consumed via raw `fetch` (POST + `ReadableStream`), NOT `EventSource` (GET-only) and NOT the shared `http` client (JSON-parsing).
- Normalized frame shape from the backend: `{ type: "text"|"thinking"|"done"|"error", content: string }`.
- `Confusion.ask` shape: `{ messages: { id, role: 'user'|'assistant', content, createdAt }[]; summary?: string; summaryState?: 'idle'|'pending'|'done'|'failed'; providerId?: string }`.
- API keys are masked (`"••••"`) by the backend; the settings form echoes the mask back to preserve the real key.
- The window is `position: fixed` anchored to the selection's screen rect, with viewport edge flipping (no drag/resize in v1).
- Active mode = live multi-turn chat. Review mode = read-only (reopened from sidebar after close); no input, no `ask-stream` calls.
- Closing an active panel that produced ≥1 assistant reply calls `POST .../ask/summarize`; the sidebar then shows the summary via `confusion-updated` (already wired in Phase A's `useArtifactRefresh`) + hover.
- Frontend tests: `cd frontend && npm run test`. Build: `cd frontend && npm run build`.

## File Structure

- Create `frontend/src/lib/parseAskAiStream.ts` (+ `_test.ts`) — pure SSE splitter.
- Create `frontend/src/api/askAi.ts` (+ `_test.ts`) — stream/summarize/settings hooks + types; extend `Confusion` with `ask`.
- Modify `frontend/src/api/confusions.ts` — add `ask?` field + `AskSummaryState` type.
- Create `frontend/src/store/slices/askAi.ts` — `useAskAiStore`.
- Create `frontend/src/components/primitive/MarkdownView.tsx` — `useMarkdown` + mermaid (reusable renderer).
- Create `frontend/src/lib/floatingPosition.ts` (+ `_test.ts`) — viewport flip.
- Create `frontend/src/components/feature/explain/AskAiPanel.tsx` (+ `.module.css`, `.test.tsx`).
- Modify `frontend/src/components/feature/explain/ExplainReader.tsx` — "问 AI" button + mount `AskAiPanel`.
- Modify `frontend/src/components/feature/explain/ConfusionPanel.tsx` — hover summary + review reopen.

---

### Task 1: Pure SSE splitter (TDD)

**Files:**
- Create: `frontend/src/lib/parseAskAiStream.ts`
- Test: `frontend/src/lib/parseAskAiStream.test.ts`

**Interfaces:**
- Produces: `function extractSSEData(buf: string): { payloads: string[]; rest: string }` — splits an SSE byte-buffer on `\n\n`, returns the `data:` payloads (JSON strings, not parsed) and the remaining partial buffer.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/src/lib/parseAskAiStream.test.ts
import { describe, it, expect } from 'vitest';
import { extractSSEData } from './parseAskAiStream';

describe('extractSSEData', () => {
  it('extracts complete data payloads and keeps the partial tail', () => {
    const buf = 'data: {"type":"text","content":"a"}\n\ndata: {"type":"done"}\n\ndata: {"type":"text","con';
    const { payloads, rest } = extractSSEData(buf);
    expect(payloads).toEqual(['{"type":"text","content":"a"}', '{"type":"done"}']);
    expect(rest).toBe('data: {"type":"text","con');
  });

  it('returns empty payloads when no terminator yet', () => {
    const { payloads, rest } = extractSSEData('data: partial only');
    expect(payloads).toEqual([]);
    expect(rest).toBe('data: partial only');
  });

  it('ignores non-data lines', () => {
    const { payloads } = extractSSEData('event: x\ndata: {"a":1}\n\n');
    expect(payloads).toEqual(['{"a":1}']);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- parseAskAiStream`
Expected: FAIL — module not found.

- [ ] **Step 3: Write minimal implementation**

```ts
// frontend/src/lib/parseAskAiStream.ts

/**
 * Split an SSE byte buffer into complete `data:` payloads plus the remaining
 * partial tail. Each SSE frame is terminated by a blank line (\n\n). Only
 * `data:` lines are extracted (event:/id:/comment lines are ignored).
 */
export function extractSSEData(buf: string): { payloads: string[]; rest: string } {
  const payloads: string[] = [];
  let rest = buf;
  let idx: number;
  while ((idx = rest.indexOf('\n\n')) >= 0) {
    const chunk = rest.slice(0, idx);
    rest = rest.slice(idx + 2);
    for (const line of chunk.split('\n')) {
      if (line.startsWith('data:')) {
        const payload = line.slice(5).trim();
        if (payload) payloads.push(payload);
      }
    }
  }
  return { payloads, rest };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- parseAskAiStream`
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/parseAskAiStream.ts frontend/src/lib/parseAskAiStream.test.ts
git commit -m "feat(lib): add extractSSEData splitter"
```

---

### Task 2: `api/askAi.ts` — streaming fetch + summarize + settings hooks; extend `Confusion`

**Files:**
- Create: `frontend/src/api/askAi.ts`
- Modify: `frontend/src/api/confusions.ts` (add `ask` field + types)
- Test: `frontend/src/api/askAi.test.ts`

**Interfaces:**
- Produces:
  - `type AskFrame`, `type AskProvider`, `type AskConfig`
  - `function streamAskAi(projectSlug, confusionId, body, handlers): Promise<void>` — raw fetch + ReadableStream, uses `extractSSEData`, calls `handlers.onFrame` per frame, honors `handlers.signal`.
  - `function summarizeAsk(projectSlug, confusionId): Promise<void>`
  - `useAskAiSettings()`, `useSaveAskAiSettings()`, `useProbeAskAi()`
- Modifies: `Confusion` gains `ask?: AskExchange`.

- [ ] **Step 1: Extend the `Confusion` type**

In `frontend/src/api/confusions.ts`, add the `Ask` types and the field:

```ts
export type AskSummaryState = 'idle' | 'pending' | 'done' | 'failed';

export interface AskMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt: string;
}

export interface AskExchange {
  messages: AskMessage[];
  summary?: string;
  summaryState?: AskSummaryState;
  providerId?: string;
  updatedAt?: string;
}

export interface Confusion {
  id: string;
  sourceArtifactId?: string;
  paragraphId?: string;
  charStart: number;
  charEnd: number;
  quoteSnapshot: string;
  notes?: string;
  state: ConfusionState;
  createdAt: string;
  ask?: AskExchange;
}
```

- [ ] **Step 2: Write the failing test (stream parser over a fake ReadableStream)**

```ts
// frontend/src/api/askAi.test.ts
import { describe, it, expect, vi } from 'vitest';
import { streamAskAi, type AskFrame } from './askAi';

// Minimal ReadableStream stub that yields the given chunks once.
function fakeBody(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  let i = 0;
  return new ReadableStream({
    pull(controller) {
      if (i < chunks.length) {
        controller.enqueue(encoder.encode(chunks[i++]));
      } else {
        controller.close();
      }
    },
  });
}

describe('streamAskAi', () => {
  it('parses streamed frames via extractSSEData', async () => {
    const frames: AskFrame[] = [];
    const original = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: fakeBody([
        'data: {"type":"text","content":"Hel"}\n\n',
        'data: {"type":"text","content":"lo"}\n\ndata: {"type":"done"}\n\n',
      ]),
    } as Response);
    await streamAskAi('p', 'c', { content: 'hi' }, { onFrame: (f) => frames.push(f) });
    global.fetch = original;

    expect(frames).toEqual([
      { type: 'text', content: 'Hel' },
      { type: 'text', content: 'lo' },
      { type: 'done', content: '' },
    ]);
  });

  it('throws on non-2xx', async () => {
    const original = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({ ok: false, status: 400, body: null } as Response);
    await expect(
      streamAskAi('p', 'c', { content: 'hi' }, { onFrame: () => {} }),
    ).rejects.toThrow();
    global.fetch = original;
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd frontend && npm run test -- askAi`
Expected: FAIL — module not found.

- [ ] **Step 4: Write the implementation**

```ts
// frontend/src/api/askAi.ts
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
  const res = await fetch(
    `${API_BASE}/api/projects/${encodeURIComponent(projectSlug)}/confusions/${encodeURIComponent(confusionId)}/ask-stream`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: handlers.signal,
    },
  );
  if (!res.ok || !res.body) {
    throw new ApiError(res.status, `ask-stream failed: ${res.status}`, null);
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
        const frame = JSON.parse(payload) as AskFrame;
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
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd frontend && npm run test -- askAi`
Expected: 2 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/api/askAi.ts frontend/src/api/askAi.test.ts frontend/src/api/confusions.ts
git commit -m "feat(api): ask-ai streaming fetch + summarize + settings hooks"
```

---

### Task 3: `useAskAiStore` (Zustand)

**Files:**
- Create: `frontend/src/store/slices/askAi.ts`

**Interfaces:**
- Produces: `useAskAiStore` with state `{ open, reviewMode, projectSlug, confusionId, quote, anchor, messages, streaming, providerId }` and actions `{ openActive, openReview, close, appendUser, startAssistant, appendDelta, finishStream, setProvider }`. `appendDelta(type, content)` accumulates into the last assistant message's `content` (text) or `thinking` (thinking).

- [ ] **Step 1: Write the store**

```ts
// frontend/src/store/slices/askAi.ts
import { create } from 'zustand';

import type { AskMessage } from '@/api/confusions';

export interface LiveAskMessage extends AskMessage {
  thinking?: string; // live-only; not persisted by the backend
}

export interface AskAnchor {
  top: number; // screen coords (position: fixed)
  left: number;
}

interface AskAiState {
  open: boolean;
  reviewMode: boolean;
  projectSlug: string;
  confusionId: string;
  quote: string;
  anchor: AskAnchor;
  messages: LiveAskMessage[];
  streaming: boolean;
  providerId: string;

  openActive: (args: {
    projectSlug: string;
    confusionId: string;
    quote: string;
    anchor: AskAnchor;
    providerId: string;
  }) => void;
  openReview: (args: {
    projectSlug: string;
    confusionId: string;
    quote: string;
    anchor: AskAnchor;
    messages: AskMessage[];
  }) => void;
  close: () => void;
  appendUser: (content: string) => void;
  startAssistant: () => void;
  appendDelta: (type: 'text' | 'thinking', content: string) => void;
  finishStream: () => void;
  setProvider: (id: string) => void;
}

let msgSeq = 0;
function localId() {
  msgSeq += 1;
  return `live-${msgSeq}-${Date.now()}`;
}

export const useAskAiStore = create<AskAiState>()((set) => ({
  open: false,
  reviewMode: false,
  projectSlug: '',
  confusionId: '',
  quote: '',
  anchor: { top: 0, left: 0 },
  messages: [],
  streaming: false,
  providerId: '',

  openActive: ({ projectSlug, confusionId, quote, anchor, providerId }) =>
    set({
      open: true,
      reviewMode: false,
      projectSlug,
      confusionId,
      quote,
      anchor,
      providerId,
      messages: [],
      streaming: false,
    }),
  openReview: ({ projectSlug, confusionId, quote, anchor, messages }) =>
    set({
      open: true,
      reviewMode: true,
      projectSlug,
      confusionId,
      quote,
      anchor,
      messages: messages.map((m) => ({ ...m })),
      streaming: false,
      providerId: '',
    }),
  close: () => set({ open: false, streaming: false, messages: [] }),
  appendUser: (content) =>
    set((s) => ({
      messages: [...s.messages, { id: localId(), role: 'user', content, createdAt: new Date().toISOString() }],
    })),
  startAssistant: () =>
    set((s) => ({
      streaming: true,
      messages: [
        ...s.messages,
        { id: localId(), role: 'assistant', content: '', thinking: '', createdAt: new Date().toISOString() },
      ],
    })),
  appendDelta: (type, content) =>
    set((s) => {
      if (s.messages.length === 0) return {};
      const msgs = [...s.messages];
      const last = { ...msgs[msgs.length - 1] };
      if (last.role !== 'assistant') return {};
      if (type === 'thinking') last.thinking = (last.thinking ?? '') + content;
      else last.content = (last.content ?? '') + content;
      msgs[msgs.length - 1] = last;
      return { messages: msgs };
    }),
  finishStream: () => set({ streaming: false }),
  setProvider: (id) => set({ providerId: id }),
}));
```

- [ ] **Step 2: Verify it typechecks (build)**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/store/slices/askAi.ts
git commit -m "feat(store): add useAskAiStore"
```

---

### Task 4: `MarkdownView` reusable renderer

**Files:**
- Create: `frontend/src/components/primitive/MarkdownView.tsx`

**Interfaces:**
- Produces: `function MarkdownView(props: { source: string; className?: string }): JSX.Element` — runs `useMarkdown(source)` and lazy-renders Mermaid blocks (mirrors the effect in `ExplainReader.tsx:170-187`).

- [ ] **Step 1: Write the component**

```tsx
// frontend/src/components/primitive/MarkdownView.tsx
import { useEffect, useRef } from 'react';

import { useMarkdown } from '@/hooks/useMarkdown';

export function MarkdownView({ source, className }: { source: string; className?: string }) {
  const { html, mermaid } = useMarkdown(source);
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    let cancelled = false;
    void import('mermaid').then(async (mod) => {
      if (cancelled) return;
      for (const block of mermaid) {
        const target = hostRef.current?.querySelector(`[data-mermaid-id="${block.id}"]`);
        if (!target) continue;
        try {
          const result = await mod.default.render(`${block.id}-mdview-svg`, block.code);
          (target as HTMLElement).innerHTML = result.svg;
        } catch (err) {
          (target as HTMLElement).textContent = `Mermaid render error: ${(err as Error).message}`;
        }
      }
    });
    return () => {
      cancelled = true;
    };
  }, [html, mermaid]);

  return (
    <div ref={hostRef} className={className} dangerouslySetInnerHTML={{ __html: html }} />
  );
}
```

- [ ] **Step 2: Verify build**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/primitive/MarkdownView.tsx
git commit -m "feat(primitive): add MarkdownView reusable renderer"
```

---

### Task 5: `floatingPosition` viewport flip (TDD)

**Files:**
- Create: `frontend/src/lib/floatingPosition.ts`
- Test: `frontend/src/lib/floatingPosition.test.ts`

**Interfaces:**
- Produces: `function flipPosition(anchor, size, viewport, margin?): { top, left }`.

- [ ] **Step 1: Write the failing test**

```ts
// frontend/src/lib/floatingPosition.test.ts
import { describe, it, expect } from 'vitest';
import { flipPosition } from './floatingPosition';

describe('flipPosition', () => {
  it('keeps position when there is room', () => {
    const p = flipPosition({ top: 100, left: 100 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p).toEqual({ top: 108, left: 100 });
  });
  it('flips up when the panel would overflow the bottom', () => {
    const p = flipPosition({ top: 700, left: 100 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p.top).toBeLessThan(700); // moved above the anchor
  });
  it('clamps left when the panel would overflow the right', () => {
    const p = flipPosition({ top: 100, left: 900 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p.left).toBe(1000 - 300 - 8);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- floatingPosition`
Expected: FAIL — module not found.

- [ ] **Step 3: Write minimal implementation**

```ts
// frontend/src/lib/floatingPosition.ts

/**
 * Position a fixed panel near an anchor (screen coords), flipping/clamping to
 * keep it inside the viewport. `anchor.top` is treated as the top of the region
 * to open below; if it would overflow the bottom, the panel opens above.
 */
export function flipPosition(
  anchor: { top: number; left: number },
  size: { width: number; height: number },
  viewport: { w: number; h: number },
  margin = 8,
): { top: number; left: number } {
  let top = anchor.top + margin;
  if (top + size.height > viewport.h) {
    top = Math.max(margin, anchor.top - size.height - margin);
  }
  let left = anchor.left;
  if (left + size.width > viewport.w) {
    left = Math.max(margin, viewport.w - size.width - margin);
  }
  if (left < margin) left = margin;
  return { top, left };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- floatingPosition`
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/floatingPosition.ts frontend/src/lib/floatingPosition.test.ts
git commit -m "feat(lib): add flipPosition viewport flip"
```

---

### Task 6: `AskAiPanel` floating window (TDD)

**Files:**
- Create: `frontend/src/components/feature/explain/AskAiPanel.tsx`
- Create: `frontend/src/components/feature/explain/AskAiPanel.module.css`
- Test: `frontend/src/components/feature/explain/AskAiPanel.test.tsx`

**Interfaces:**
- Consumes: `useAskAiStore`, `streamAskAi`/`summarizeAsk`, `useAskAiSettings`, `MarkdownView`, `flipPosition`, `SSE_EVENTS` (not needed here).
- Produces: `<AskAiPanel />` (no props; reads the store). Renders `null` when `!open`. In active mode: provider switcher, message list (user bubbles + assistant `MarkdownView` with a collapsible `<details>` thinking), input (Enter to send), stop button, browser-search button, close. In review mode: read-only message list + close only. Closing active with ≥1 assistant reply calls `summarizeAsk`.

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/components/feature/explain/AskAiPanel.test.tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { AskAiPanel } from './AskAiPanel';
import { useAskAiStore } from '@/store/slices/askAi';

vi.mock('@/api/askAi', () => ({
  streamAskAi: vi.fn(),
  summarizeAsk: vi.fn(),
  useAskAiSettings: () => ({ data: { default: 'p1', searchEngine: 'google', providers: [{ id: 'p1', kind: 'openai', baseURL: 'x', apiKey: 'k', model: 'm' }] } }),
}));
vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: (src: string) => ({ html: `<div>${src}</div>`, mermaid: [] }),
}));

beforeEach(() => {
  useAskAiStore.setState({ open: false, messages: [], streaming: false, reviewMode: false });
});

describe('AskAiPanel', () => {
  it('renders nothing when closed', () => {
    const { container } = render(<AskAiPanel />);
    expect(container.firstChild).toBeNull();
  });

  it('active mode shows input + provider switcher + browser search', () => {
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    expect(screen.getByPlaceholderText(/问/)).toBeTruthy();
    expect(screen.getByTitle(/浏览器搜索/)).toBeTruthy();
  });

  it('review mode is read-only (no input)', () => {
    useAskAiStore.getState().openReview({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 },
      messages: [{ id: 'm1', role: 'assistant', content: 'prior answer', createdAt: '' }],
    });
    render(<AskAiPanel />);
    expect(screen.queryByPlaceholderText(/问/)).toBeNull();
    expect(screen.getByText('prior answer')).toBeTruthy();
  });

  it('close button closes the panel', () => {
    useAskAiStore.getState().openActive({
      projectSlug: 'p', confusionId: 'c', quote: 'sel',
      anchor: { top: 100, left: 100 }, providerId: 'p1',
    });
    render(<AskAiPanel />);
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    expect(useAskAiStore.getState().open).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- AskAiPanel`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the component + CSS**

```tsx
// frontend/src/components/feature/explain/AskAiPanel.tsx
import { useEffect, useRef, useState } from 'react';
import { Cross2Icon } from '@radix-ui/react-icons';

import { streamAskAi, summarizeAsk, useAskAiSettings } from '@/api/askAi';
import { useAskAiStore } from '@/store/slices/askAi';
import { MarkdownView } from '@/components/primitive/MarkdownView';
import { flipPosition } from '@/lib/floatingPosition';

import s from './AskAiPanel.module.css';

const PANEL_SIZE = { width: 440, height: 520 };

export function AskAiPanel() {
  const store = useAskAiStore();
  const settingsQuery = useAskAiSettings();
  const [input, setInput] = useState('');
  const abortRef = useRef<AbortController | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Auto-scroll on new content.
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [store.messages, store.streaming]);

  if (!store.open) return null;

  const pos = flipPosition(store.anchor, PANEL_SIZE, { w: window.innerWidth, h: window.innerHeight });

  const send = async () => {
    const content = input.trim();
    if (!content || store.streaming || store.reviewMode) return;
    setInput('');
    store.appendUser(content);
    store.startAssistant();
    const ac = new AbortController();
    abortRef.current = ac;
    try {
      await streamAskAi(store.projectSlug, store.confusionId, {
        content,
        providerId: store.providerId || undefined,
      }, {
        signal: ac.signal,
        onFrame: (f) => {
          if (f.type === 'text') store.appendDelta('text', f.content);
          else if (f.type === 'thinking') store.appendDelta('thinking', f.content);
          else if (f.type === 'done') store.finishStream();
        },
      });
    } catch {
      /* aborted or errored */
    } finally {
      store.finishStream();
      abortRef.current = null;
    }
  };

  const stop = () => {
    abortRef.current?.abort();
    store.finishStream();
  };

  const close = () => {
    if (!store.reviewMode && store.messages.some((m) => m.role === 'assistant')) {
      void summarizeAsk(store.projectSlug, store.confusionId);
    }
    stop();
    store.close();
  };

  const searchURL =
    (settingsQuery.data?.searchEngine === 'bing' ? 'https://www.bing.com/search?q=' : 'https://www.google.com/search?q=') +
    encodeURIComponent(store.quote);

  return (
    <div className={s.panel} style={{ top: pos.top, left: pos.left }} role="dialog" aria-label="问 AI">
      <header className={s.header}>
        <span className={s.title}>{store.reviewMode ? '答疑回顾' : '问 AI'}</span>
        {!store.reviewMode && settingsQuery.data && (
          <select
            className={s.provider}
            value={store.providerId}
            onChange={(e) => store.setProvider(e.target.value)}
            aria-label="模型源"
          >
            {settingsQuery.data.providers.map((p) => (
              <option key={p.id} value={p.id}>{p.name || p.id}</option>
            ))}
          </select>
        )}
        <button type="button" className={s.iconBtn} onClick={close} aria-label="关闭" title="关闭">
          <Cross2Icon />
        </button>
      </header>

      <div className={s.messages} ref={listRef}>
        {store.messages.length === 0 && !store.reviewMode && (
          <div className={s.hint}>针对选中文字提问，回答会流式输出并保存到这条疑问。</div>
        )}
        {store.messages.map((m) =>
          m.role === 'user' ? (
            <div key={m.id} className={s.user}>{m.content}</div>
          ) : (
            <div key={m.id} className={s.assistant}>
              {m.thinking ? (
                <details className={s.thinking}>
                  <summary>思考过程</summary>
                  <div className={s.thinkingBody}>{m.thinking}</div>
                </details>
              ) : null}
              <MarkdownView source={m.content || (store.streaming ? '…' : '')} className={s.answer} />
            </div>
          ),
        )}
      </div>

      {!store.reviewMode && (
        <footer className={s.footer}>
          <a className={s.searchLink} href={searchURL} target="_blank" rel="noreferrer" title="在浏览器搜索">
            搜索
          </a>
          <input
            className={s.input}
            placeholder="追问…（Enter 发送）"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                void send();
              }
            }}
          />
          {store.streaming ? (
            <button type="button" className={s.stopBtn} onClick={stop}>停止</button>
          ) : (
            <button type="button" className={s.sendBtn} onClick={() => void send()} disabled={!input.trim()}>
              发送
            </button>
          )}
        </footer>
      )}
    </div>
  );
}
```

```css
/* frontend/src/components/feature/explain/AskAiPanel.module.css */
.panel {
  position: fixed;
  z-index: 210;
  width: 440px;
  height: 520px;
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid #d8dbe0;
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
  overflow: hidden;
  font-size: 14px;
}
.header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid #ececef;
  background: #fafafb;
}
.title { font-weight: 600; flex: 1; }
.provider { font-size: 12px; border: 1px solid #d8dbe0; border-radius: 6px; padding: 2px 4px; }
.iconBtn { border: none; background: transparent; cursor: pointer; display: inline-flex; padding: 4px; border-radius: 6px; }
.iconBtn:hover { background: rgba(0, 0, 0, 0.06); }
.messages { flex: 1; overflow-y: auto; padding: 10px; display: flex; flex-direction: column; gap: 10px; background: #fff; }
.hint { color: #888; font-size: 12px; }
.user { align-self: flex-end; background: #4c6ef5; color: #fff; padding: 6px 10px; border-radius: 12px 12px 2px 12px; max-width: 80%; white-space: pre-wrap; }
.assistant { align-self: flex-start; max-width: 92%; }
.thinking { background: #f4f5f7; border: 1px solid #e6e8eb; border-radius: 8px; padding: 6px 8px; margin-bottom: 6px; font-size: 12px; color: #666; }
.thinking summary { cursor: pointer; }
.thinkingBody { margin-top: 4px; white-space: pre-wrap; }
.answer { line-height: 1.6; }
.answer p { margin: 0 0 8px; }
.answer :global(pre) { background: #f4f5f7; padding: 8px; border-radius: 6px; overflow-x: auto; }
.footer { display: flex; align-items: center; gap: 6px; padding: 8px; border-top: 1px solid #ececef; background: #fafafb; }
.searchLink { font-size: 12px; color: #4c6ef5; text-decoration: none; padding: 4px 6px; }
.input { flex: 1; border: 1px solid #d8dbe0; border-radius: 8px; padding: 6px 8px; font-size: 13px; }
.sendBtn, .stopBtn { border: none; border-radius: 8px; padding: 6px 12px; cursor: pointer; font-size: 13px; }
.sendBtn { background: #4c6ef5; color: #fff; }
.sendBtn:disabled { background: #b9c0e8; cursor: not-allowed; }
.stopBtn { background: #e6e8eb; color: #444; }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- AskAiPanel`
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/feature/explain/AskAiPanel.tsx frontend/src/components/feature/explain/AskAiPanel.module.css frontend/src/components/feature/explain/AskAiPanel.test.tsx
git commit -m "feat(explain): add anchored AskAiPanel floating window"
```

---

### Task 7: Selection trigger + mount in ExplainReader

**Files:**
- Modify: `frontend/src/components/feature/explain/ExplainReader.tsx`

**Interfaces:**
- Consumes: `useCreateConfusion`, `useAskAiSettings`, `useAskAiStore`, `AskAiPanel`.
- Produces: a third "问 AI" button in the `selectionBar`; on click it creates a confusion, then opens the AskAi store in active mode anchored to the selection's screen rect + default provider. `AskAiPanel` is mounted once at the `ExplainReader` root.

- [ ] **Step 1: Add the trigger + mount**

In `frontend/src/components/feature/explain/ExplainReader.tsx`:

Add imports (top, with the others):
```tsx
import { useAskAiStore } from '@/store/slices/askAi';
import { useAskAiSettings } from '@/api/askAi';
import { AskAiPanel } from './AskAiPanel';
```

The selection bar lives in `ExplainPage`. The "问 AI" handler needs a confusion id (created async) and the screen rect. Augment the stored selection to also carry the screen rect so the anchor is correct. In the `onMouseUp` handler, where `rect` is computed (line 220), also capture screen coords. Change the `setSelection({...})` call to include `screenTop: rect.top` and `screenLeft: rect.left + rect.width / 2`:

```tsx
        setSelection({
          ...anchor,
          top: rect.top - (hostRect?.top ?? 0) - 42,
          left: rect.left - (hostRect?.left ?? 0) + rect.width / 2,
          screenTop: rect.top,
          screenLeft: rect.left + rect.width / 2,
          overlaps: overlapsExisting(anchor, pageConfusions),
        });
```
and widen the `selection` state type:
```tsx
  const [selection, setSelection] = useState<
    (TextAnchor & { top: number; left: number; screenTop: number; screenLeft: number; overlaps: boolean }) | null
  >(null);
```

Add the third button inside `selectionBar` (between 保存摘要 and 取消). It needs the settings (default provider) + createConfusion + the store. Add these hooks in `ExplainPage` (near `createConfusion`):
```tsx
  const askAi = useAskAiStore();
  const askAiSettings = useAskAiSettings();
```

Then the button:
```tsx
          <button
            type="button"
            onClick={async () => {
              const created = await createConfusion.mutateAsync({
                projectSlug,
                confusion: {
                  sourceArtifactId: artifactID,
                  quoteSnapshot: selection.text.slice(0, 500),
                  charStart: selection.start,
                  charEnd: Math.min(selection.end, selection.start + 500),
                },
              });
              askAi.openActive({
                projectSlug,
                confusionId: created.confusion.id,
                quote: selection.text.slice(0, 500),
                anchor: { top: selection.screenTop, left: selection.screenLeft },
                providerId: askAiSettings.data?.default ?? '',
              });
              setSelection(null);
              window.getSelection()?.removeAllRanges();
            }}
          >
            问 AI
          </button>
```

Finally, mount `<AskAiPanel />` once. The outer `ExplainReader` component (not `ExplainPage`) is the right place — add it just before the closing `</div>` of the `layout` (after `<ExplainInfographic .../>`):
```tsx
      {pages.length > 0 && (
        <ExplainInfographic projectSlug={projectSlug} visible={currentIndex === pages.length - 1} />
      )}
      <AskAiPanel />
    </div>
```

- [ ] **Step 2: Verify build + tests**

Run:
```bash
cd frontend && npx tsc --noEmit
cd frontend && npm run test
```
Expected: typecheck clean; all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/feature/explain/ExplainReader.tsx
git commit -m "feat(explain): add 问 AI selection trigger + mount AskAiPanel"
```

---

### Task 8: ConfusionPanel hover summary + review reopen

**Files:**
- Modify: `frontend/src/components/feature/explain/ConfusionPanel.tsx`

**Interfaces:**
- Consumes: `useConfusions`, `useAskAiStore`. Each confusion now optionally carries `ask`.
- Produces: for confusions with `ask.summaryState`, a hover title showing the summary (done), `"生成总结中..."` (pending), or `"总结生成失败"` (failed). Clicking an `asked` confusion (with `ask`) opens the AskAi store in review mode anchored near the sidebar.

- [ ] **Step 1: Read the current ConfusionPanel**

Run: `cd frontend && sed -n '1,180p' src/components/feature/explain/ConfusionPanel.tsx` (or open it in the editor) to see the exact list-item render and where `projectSlug` is in scope. The changes below assume a list of confusions mapped to items; adapt the wrapper element to match the existing markup.

- [ ] **Step 2: Add the hover + reopen behavior**

Add imports at the top of `ConfusionPanel.tsx`:
```tsx
import { useAskAiStore } from '@/store/slices/askAi';
```

Inside the component, get the store:
```tsx
  const askAi = useAskAiStore();
```

Define a helper for the hover title (place it near the top of the module or inline):
```tsx
function askHoverTitle(ask?: { summaryState?: string; summary?: string }): string | undefined {
  if (!ask) return undefined;
  if (ask.summaryState === 'pending') return '生成总结中...';
  if (ask.summaryState === 'failed') return '总结生成失败';
  if (ask.summaryState === 'done') return ask.summary;
  return undefined;
}
```

On each list item for a confusion that has `ask`, set the hover title and make clicking it open review:
```tsx
  const hover = askHoverTitle(confusion.ask);
  // on the item's root element (adapt to the existing className/element):
  //   title={hover}
  //   onClick={() => confusion.ask && askAi.openReview({
  //     projectSlug,
  //     confusionId: confusion.id,
  //     quote: confusion.quoteSnapshot,
  //     anchor: { top: 120, left: window.innerWidth - 460 }, // dock near the right edge
  //     messages: confusion.ask.messages,
  //   })}
```
Concretely, wrap the existing item content so that a confusion with `ask` gets `title={hover}` and an `onClick` that calls `askAi.openReview(...)` (and does NOT trigger the existing delete/edit flow — check the existing handler and chain or gate appropriately). If the panel already uses a per-item click for something else, add a small dedicated "查看答疑" button instead of hijacking the row click:

```tsx
  {confusion.ask && (
    <button
      type="button"
      className={s.reviewBtn}            /* add .reviewBtn in the module css */
      title={hover}
      onClick={() =>
        askAi.openReview({
          projectSlug,
          confusionId: confusion.id,
          quote: confusion.quoteSnapshot,
          anchor: { top: 120, left: window.innerWidth - 460 },
          messages: confusion.ask!.messages,
        })
      }
    >
      答疑
    </button>
  )}
```
Add `.reviewBtn { font-size: 11px; border: 1px solid #d8dbe0; background: #fff; border-radius: 6px; padding: 2px 6px; cursor: pointer; }` to `ConfusionPanel.module.css`.

The summary text refreshes automatically: `confusion-updated` (emitted by the summarize endpoint) is already subscribed in Phase A's `useArtifactRefresh`, which invalidates `qk.confusions.all(slug)` → the panel re-fetches → hover shows the new summary.

- [ ] **Step 3: Verify build + tests**

Run:
```bash
cd frontend && npx tsc --noEmit
cd frontend && npm run test
cd frontend && npm run build
```
Expected: typecheck clean; tests PASS; production build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/feature/explain/ConfusionPanel.tsx frontend/src/components/feature/explain/ConfusionPanel.module.css
git commit -m "feat(explain): confusion sidebar hover summary + read-only review"
```

---

## End-to-End Smoke (after all 8 tasks, with Phase B backend running)

1. `cd frontend && npm run dev` (proxies to `go run ./backend-go/cmd/lll`).
2. Settings → add an OpenAI-compatible provider + an Anthropic provider; probe both → ok.
3. Open an Explain page, select a passage → the bar shows 保存摘要 / **问 AI** / 取消. Click **问 AI** → the floating window opens anchored to the selection; type a question, Enter → answer streams in with a collapsible 思考 section and full markdown. Ask a follow-up → multi-turn.
4. Click 搜索 → browser opens a Google search for the quote.
5. Click 关闭 → the sidebar entry appears; hover shows "生成总结中..." then the ≤250-char summary; the confusion becomes `asked`.
6. Click the sidebar 答疑 button → the window reopens read-only (no input) showing the stored exchange.

## Notes / Honest Limitations (Phase B frontend)

- No drag/resize/multiple windows. One panel at a time (a single store singleton).
- The review-reopen anchor docks near the right edge (`left: innerWidth - 460`); the active-mode anchor follows the selection. Both apply viewport flipping.
- `MarkdownView` duplicates the mermaid effect already inline in `ExplainReader.ExplainPage`; a future cleanup can have ExplainPage use `MarkdownView` too (intentionally not refactored here to keep the slice surgical).
- Thinking text is live-only (not persisted by the backend); review mode shows answers only, no thinking.
- The provider switcher is hidden in review mode; review always reflects the stored exchange regardless of current default.
