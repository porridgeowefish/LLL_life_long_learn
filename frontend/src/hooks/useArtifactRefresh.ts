import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { qk } from '@/api/queryKeys';

function invalidatePractice(
  qc: ReturnType<typeof useQueryClient>,
  slug: string,
) {
  qc.invalidateQueries({ queryKey: ['practice', 'tasks', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'evaluation', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'draft', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'attempt', 'latest', slug] });
}

/**
 * Drive server-state refresh from SSE instead of polling. Mount once per active
 * project (in ProjectPage). On artifact-updated for this project, invalidate the
 * affected file/practice queries; on confusion-updated, invalidate the
 * confusions list. The 'files' prefix covers explain manifest/pages and the
 * intro/extend/summary outputs read via qk.files.raw.
 */
export function useArtifactRefresh(projectSlug: string): void {
  const qc = useQueryClient();
  useEffect(() => {
    const unsubArtifact = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const p = data as { projectSlug?: string; zone?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      qc.invalidateQueries({ queryKey: ['files', projectSlug] });
      // generatedZones is derived by GET /api/projects/:slug from the files on
      // disk. Refresh the project detail as well as the artifact itself so the
      // sidebar timeline reflects newly generated content without a reload.
      qc.invalidateQueries({ queryKey: qk.projects.detail(projectSlug) });
      if (p.zone === 'overview') {
        qc.invalidateQueries({ queryKey: ['projects', projectSlug, 'discipline-overview'] });
      }
      if (p.zone === 'learning-plan') {
        qc.invalidateQueries({ queryKey: ['projects', projectSlug, 'discipline-learning-plan'] });
      }
      if (p.zone === 'discipline-topics') {
        qc.invalidateQueries({ queryKey: ['projects', projectSlug, 'discipline-topics'] });
      }
      if (p.zone === 'practice') invalidatePractice(qc, projectSlug);
    });
    const unsubConfusion = subscribeToSSE(SSE_EVENTS.confusionUpdated, (data) => {
      const p = data as { projectSlug?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      qc.invalidateQueries({ queryKey: qk.confusions.all(projectSlug) });
    });
    return () => {
      unsubArtifact();
      unsubConfusion();
    };
  }, [projectSlug, qc]);
}
