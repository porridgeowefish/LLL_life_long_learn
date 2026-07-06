import { useEffect, useState } from 'react';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { useSessionStore } from '@/store/slices/session';

export interface RunProgressState {
  active: boolean;
  activity: string | null;
  dismiss: () => void;
}

/**
 * Phase A progress source: indeterminate, driven by artifact-updated activity
 * for the current project. The bar appears once an artifact event arrives for
 * this slug (generation is actually producing output here), updates its activity
 * text as pages/files land, and hides on session-completed/failed (emitted
 * reliably from Phase C hooks) or on user dismiss. Per-page/phase granularity
 * arrives in Phase C via the run-progress event.
 */
export function useRunProgress(projectSlug: string): RunProgressState {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const [activity, setActivity] = useState<string | null>(null);
  const [completed, setCompleted] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Reset run-scoped state when the active session changes.
  useEffect(() => {
    setActivity(null);
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
    const doneEvents = [SSE_EVENTS.sessionCompleted, SSE_EVENTS.sessionFailed];
    const unsubsDone = doneEvents.map((evt) =>
      subscribeToSSE(evt, () => setCompleted(true)),
    );
    return () => {
      unsubArtifact();
      unsubsDone.forEach((u) => u());
    };
  }, [projectSlug]);

  const active = !!activeSessionId && !completed && !dismissed && activity !== null;
  return { active, activity, dismiss: () => setDismissed(true) };
}
