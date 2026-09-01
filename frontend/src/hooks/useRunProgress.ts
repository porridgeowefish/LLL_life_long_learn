import { useEffect, useState } from 'react';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { useSessionStore } from '@/store/slices/session';
import { useActiveSessions } from '@/api/sessions';

export interface RunProgressState {
  active: boolean;
  activity: string | null;
  pagesDone: number;
  pagesPlanned: number;
  dismiss: () => void;
}

/**
 * Run progress source.
 *
 * The bar must survive a page refresh. `activeSessionId` is client-only (not
 * persisted) and resets to null on reload, so we treat the BACKEND's active
 * session list as the source of truth for "is a run in progress for this
 * project". `activeSessionId` is kept only as an instant fallback for the moment
 * a run is invoked (before the active-sessions query refetches).
 *
 * - Phase A fallback: artifact-updated events for this project keep an
 *   indeterminate bar alive with a "最近更新：…" label (works on every runtime).
 * - Phase C (Claude hooks): run-progress events stream per-page granularity,
 *   correlated by runId === the active run id.
 *
 * The bar shows immediately (as "运行中…") once a run is known to be active — no
 * need to wait for the first event, so it reappears right after a refresh. It
 * hides on session-completed/failed (Phase C Stop hook for Claude) or on dismiss.
 */
export function useRunProgress(projectSlug: string): RunProgressState {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const setActiveSession = useSessionStore((s) => s.setActiveSession);
  const { data: activeSessionsData, refetch: refetchActiveSessions } = useActiveSessions();
  const activeRun = (activeSessionsData?.sessions ?? []).find(
    (s) => s.projectSlug === projectSlug,
  );
  const runId = activeRun?.id ?? activeSessionId ?? '';

  const [activity, setActivity] = useState<string | null>(null);
  const [pagesDone, setPagesDone] = useState(0);
  const [pagesPlanned, setPagesPlanned] = useState(0);
  const [completed, setCompleted] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Reset run-scoped state when the run we're tracking changes.
  useEffect(() => {
    setActivity(null);
    setPagesDone(0);
    setPagesPlanned(0);
    setCompleted(false);
    setDismissed(false);
  }, [runId]);

  useEffect(() => {
    const unsubArtifact = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const p = data as { projectSlug?: string; zone?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      setActivity(`最近更新：${p.zone ?? '产物'}`);
      setDismissed(false);
    });
    // Phase C: per-run granularity from Claude Code hooks. Only match the
    // active run's id so events from a different run don't leak in.
    const unsubRunProgress = subscribeToSSE(SSE_EVENTS.runProgress, (data) => {
      const p = data as { runId?: string; activity?: string; pagesDone?: number; pagesPlanned?: number } | undefined;
      if (!p || !runId || p.runId !== runId) return;
      if (typeof p.pagesDone === 'number') setPagesDone(p.pagesDone);
      if (typeof p.pagesPlanned === 'number') setPagesPlanned(p.pagesPlanned);
      if (typeof p.activity === 'string' && p.activity.length > 0) setActivity(p.activity);
      setDismissed(false);
    });
    const doneEvents = [SSE_EVENTS.sessionCompleted, SSE_EVENTS.sessionFailed];
    const unsubsDone = doneEvents.map((evt) =>
      subscribeToSSE(evt, (data) => {
        const p = data as { runId?: string; sessionId?: string } | undefined;
        const finishedRunId = p?.runId ?? p?.sessionId ?? '';
        if (!runId || finishedRunId !== runId) return;
        setCompleted(true);
        if (activeSessionId === runId) setActiveSession(null);
        void refetchActiveSessions();
      }),
    );
    return () => {
      unsubArtifact();
      unsubRunProgress();
      unsubsDone.forEach((u) => u());
    };
  }, [activeSessionId, projectSlug, refetchActiveSessions, runId, setActiveSession]);

  // Active whenever a run is known (backend active-session OR just-invoked
  // client id) and not completed/dismissed. No event needed — the bar can show
  // "运行中…" immediately, including right after a refresh.
  const active = runId !== '' && !completed && !dismissed;
  return { active, activity, pagesDone, pagesPlanned, dismiss: () => setDismissed(true) };
}
