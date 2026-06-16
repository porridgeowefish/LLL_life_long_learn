import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { useExplainInfographic, useRequestExplainInfographic } from '@/api/explainInfographic';
import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';

import s from './ExplainInfographic.module.css';

interface ExplainInfographicProps {
  projectSlug: string;
}

export function ExplainInfographic({ projectSlug }: ExplainInfographicProps) {
  const query = useExplainInfographic(projectSlug);
  const kick = useRequestExplainInfographic();
  const qc = useQueryClient();
  const hasAttemptedRef = useRef(false);

  // Listen for the backend's artifact-updated SSE event so the finished
  // infographic swaps in immediately instead of waiting up to 3s for the
  // next poll. Only react to events for THIS project's infographic.
  useEffect(() => {
    const unsubscribe = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const payload = data as { slug?: string; artifact?: string } | undefined;
      if (
        payload?.artifact === 'explain/infographic.png' &&
        payload?.slug === projectSlug
      ) {
        qc.invalidateQueries({ queryKey: ['explain', 'infographic', projectSlug] });
      }
    });
    return unsubscribe;
  }, [projectSlug, qc]);

  useEffect(() => {
    // Auto-kick generation when status is 'missing' and we haven't tried yet
    if (query.data?.status === 'missing' && !kick.isPending && !hasAttemptedRef.current) {
      hasAttemptedRef.current = true;
      kick.mutate(projectSlug);
    }
    // Reset flag if we get a successful 'complete' (allows re-generation on explicit user action)
    if (query.data?.status === 'complete') {
      hasAttemptedRef.current = false;
    }
  }, [query.data?.status, kick.isPending, projectSlug]);

  // Feature not configured (503) or not applicable — render nothing
  if (query.data?.status === 'missing' && !query.isLoading && !kick.isPending && hasAttemptedRef.current) {
    // A POST was attempted but we're still 'missing' — feature likely disabled (503 not configured)
    return null;
  }

  // Loading states: initial query load OR pending/running generation
  if (query.isLoading || query.data?.status === 'pending' || query.data?.status === 'running') {
    return (
      <div className={s.container}>
        <div className={s.skeleton}>正在生成主题信息图…</div>
      </div>
    );
  }

  // Success: show the image
  if (query.data?.status === 'complete' && query.data?.url) {
    return (
      <div className={s.container}>
        <img src={query.data.url} alt="主题信息图" className={s.image} />
      </div>
    );
  }

  // Failed: show error + retry button
  if (query.data?.status === 'failed') {
    return (
      <div className={s.container}>
        <div className={s.error}>
          <span className={s.errorText}>
            {query.data.error || '信息图生成失败，请稍后重试'}
          </span>
          <button
            type="button"
            className={s.retryButton}
            onClick={() => kick.mutate(projectSlug)}
            disabled={kick.isPending}
          >
            {kick.isPending ? '生成中…' : '重试'}
          </button>
        </div>
      </div>
    );
  }

  // Fallback: status is 'missing' but we haven't triggered yet — initial load, show nothing
  // (The useEffect will trigger the POST on next render)
  return null;
}
