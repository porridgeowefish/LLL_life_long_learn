import { useEffect, useState } from 'react';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { useSessionStore } from '@/store/slices/session';

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
 * - Phase A fallback (always on): artifact-updated events for this project
 *   keep an indeterminate bar alive with a "最近更新：…" activity label, so
 *   non-Claude runtimes (which never emit run-progress) still show progress.
 * - Phase C (Claude runs with injected hooks): run-progress events stream
 *   per-page granularity. We correlate by runId === activeSessionId and
 *   surface pagesDone/pagesPlanned so the bar can switch to a determinate
 *   "N / M 页" mode.
 *
 * The bar hides on session-completed/failed (emitted reliably from Phase C
 * hooks for Claude runs; for non-Claude runs it arrives via other paths) or
 * on user dismiss.
 */
export function useRunProgress(projectSlug: string): RunProgressState {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const [activity, setActivity] = useState<string | null>(null);
  const [pagesDone, setPagesDone] = useState(0);
  const [pagesPlanned, setPagesPlanned] = useState(0);
  const [completed, setCompleted] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Reset run-scoped state when the active session changes.
  useEffect(() => {
    setActivity(null);
    setPagesDone(0);
    setPagesPlanned(0);
    setCompleted(false);
    setDismissed(false);
  }, [activeSessionId]);

  useEffect(() => {
    const unsubArtifact = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const p = data as { projectSlug?: string; zone?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      setActivity(`最近更新：${p.zone ?? '产物'}`);
      setDismissed(false);
    });
    // Phase C: per-run granularity from Claude Code hooks. Only match the
    // active session's runId so events from a different run don't leak in.
    const unsubRunProgress = subscribeToSSE(SSE_EVENTS.runProgress, (data) => {
      const p = data as { runId?: string; activity?: string; pagesDone?: number; pagesPlanned?: number } | undefined;
      if (!p || !activeSessionId || p.runId !== activeSessionId) return;
      if (typeof p.pagesDone === 'number') setPagesDone(p.pagesDone);
      if (typeof p.pagesPlanned === 'number') setPagesPlanned(p.pagesPlanned);
      if (typeof p.activity === 'string' && p.activity.length > 0) setActivity(p.activity);
      setDismissed(false);
    });
    const doneEvents = [SSE_EVENTS.sessionCompleted, SSE_EVENTS.sessionFailed];
    const unsubsDone = doneEvents.map((evt) =>
      subscribeToSSE(evt, () => setCompleted(true)),
    );
    return () => {
      unsubArtifact();
      unsubRunProgress();
      unsubsDone.forEach((u) => u());
    };
  }, [projectSlug, activeSessionId]);

  const active = !!activeSessionId && !completed && !dismissed && activity !== null;
  return { active, activity, pagesDone, pagesPlanned, dismiss: () => setDismissed(true) };
}
