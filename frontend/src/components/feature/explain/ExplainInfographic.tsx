import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { useExplainInfographic, useRequestExplainInfographic } from '@/api/explainInfographic';
import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { Modal } from '@/components/primitive/Modal';
import { Button } from '@/components/primitive/Button';

import s from './ExplainInfographic.module.css';

interface ExplainInfographicProps {
  projectSlug: string;
  /** When false the component stays mounted (so generation can start early and
   * SSE keeps working) but renders nothing. Used to show the infographic only
   * on the tutorial's last page. */
  visible?: boolean;
}

export function ExplainInfographic({ projectSlug, visible = true }: ExplainInfographicProps) {
  const query = useExplainInfographic(projectSlug);
  const kick = useRequestExplainInfographic();
  const qc = useQueryClient();
  const hasAttemptedRef = useRef(false);
  const [confirmRegen, setConfirmRegen] = useState(false);

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
      kick.mutate({ projectSlug });
    }
    // Reset flag if we get a successful 'complete' (allows re-generation on explicit user action)
    if (query.data?.status === 'complete') {
      hasAttemptedRef.current = false;
    }
  }, [query.data?.status, kick.isPending, projectSlug]);

  // Not on the last page: stay mounted so effects keep running (early
  // generation + SSE), but render nothing.
  if (!visible) return null;

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
    // Cache-bust with updatedAt so a regenerated image (same URL, new bytes)
    // is always fetched fresh instead of served from the browser cache.
    const src = query.data.updatedAt
      ? `${query.data.url}?t=${encodeURIComponent(query.data.updatedAt)}`
      : query.data.url;
    return (
      <div className={s.container}>
        <img src={src} alt="主题信息图" className={s.image} />
        <div className={s.actions}>
          <button
            type="button"
            className={s.regenerateButton}
            onClick={() => setConfirmRegen(true)}
          >
            重新生成
          </button>
        </div>
        <Modal
          open={confirmRegen}
          onOpenChange={setConfirmRegen}
          title="重新生成信息图"
          description="将再次调用 AI 生成一张新的主题信息图，通常需要 1-3 分钟，现有图片会被覆盖。"
          size="sm"
          footer={
            <>
              <Button variant="outline" onClick={() => setConfirmRegen(false)}>
                取消
              </Button>
              <Button
                variant="primary"
                loading={kick.isPending}
                onClick={() => {
                  kick.mutate({ projectSlug, force: true });
                  setConfirmRegen(false);
                }}
              >
                确认重新生成
              </Button>
            </>
          }
        >
          <p className={s.confirmNote}>确认后将在后台重新生成，期间会显示生成中的占位。</p>
        </Modal>
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
            onClick={() => kick.mutate({ projectSlug })}
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
