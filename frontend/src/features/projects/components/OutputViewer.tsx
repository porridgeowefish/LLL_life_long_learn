import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

import { useMarkdown } from '@/shared/hooks/useMarkdown';
import { mountMermaidBlocks } from '@/shared/lib/mermaidRenderer';
import { qk } from '@/shared/queryKeys';
import { http } from '@/shared/client';
import { useCreateConfusion } from '@/features/legacy-zones';
import { ZONE_FILENAME, type ZoneName } from '@/shared/types/domain';
import { EmptyState } from '@/shared/primitive/EmptyState';
import { Tag } from '@/shared/primitive/Tag';
import {
  selectionToolbarPosition,
  visibleViewportBounds,
} from '@/shared/lib/floatingPosition';
import { selectionEndpointRect } from '@/shared/lib/selectionGeometry';

import s from './OutputViewer.module.css';

interface OutputViewerProps {
  slug: string;
  zone: ZoneName;
}

// Selection toolbar actions for Explain zone text selection.
interface SelectionToolbar {
  text: string;
  anchor: { top: number; bottom: number; centerX: number };
  top: number;
  left: number;
}

export function OutputViewer({ slug, zone }: OutputViewerProps) {
  const filename = ZONE_FILENAME[zone];
  const { data, isLoading, error } = useQuery({
    queryKey: qk.files.raw(slug, `${zone.toLowerCase()}/${filename}`),
    queryFn: () =>
      http.get<string>(
        `/files/projects/${encodeURIComponent(slug)}/${zone.toLowerCase()}/${filename}`,
        { rawText: true },
      ),
  });

  const text = data ?? '';
  const { html, mermaid } = useMarkdown(text);
  const hostRef = useRef<HTMLDivElement>(null);
  const markdownRef = useRef<HTMLDivElement>(null);

  // Selection toolbar state — only for Explain zone.
  const [toolbar, setToolbar] = useState<SelectionToolbar | null>(null);
  const toolbarRef = useRef<HTMLDivElement>(null);
  const selectingRef = useRef(false);
  const createConfusion = useCreateConfusion();

  useLayoutEffect(() => {
    if (!toolbar || !toolbarRef.current) return;
    const measured = toolbarRef.current.getBoundingClientRect();
    const next = selectionToolbarPosition(
      toolbar.anchor,
      { width: measured.width, height: measured.height },
      visibleViewportBounds(),
    );
    if (Math.abs(next.top - toolbar.top) > 1 || Math.abs(next.left - toolbar.left) > 1) {
      setToolbar({ ...toolbar, ...next });
    }
  }, [toolbar]);

  // After HTML is set, replace each mermaid placeholder with a rendered SVG.
  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    return mountMermaidBlocks(hostRef.current, mermaid, 'output-viewer');
  }, [html, mermaid]);

  // Listen for text selection in Explain zone → show floating toolbar.
  const updateSelection = useCallback((point?: { x: number; y: number }) => {
    if (zone !== 'Explain') return;
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || !sel.toString().trim()) {
      setToolbar(null);
      return;
    }
    if (sel.rangeCount === 0) {
      setToolbar(null);
      return;
    }
    const range = sel.getRangeAt(0);
    if (!markdownRef.current?.contains(range.commonAncestorContainer)) {
      setToolbar(null);
      return;
    }
    const rect = selectionEndpointRect(range, point);
    if (!rect) {
      setToolbar(null);
      return;
    }
    const anchor = {
      top: rect.top,
      bottom: rect.bottom,
      centerX: rect.left + rect.width / 2,
    };
    const position = selectionToolbarPosition(
      anchor,
      { width: 240, height: 40 },
      visibleViewportBounds(),
    );
    setToolbar({
      text: sel.toString().trim(),
      anchor,
      ...position,
    });
  }, [zone]);

  useEffect(() => {
    let selectionFrame = 0;
    const finishAtPoint = (event: MouseEvent | PointerEvent) => {
      selectingRef.current = false;
      window.cancelAnimationFrame(selectionFrame);
      // Native selection is finalized after pointerup in some browsers. Read
      // the range on the next frame so the toolbar uses the actual release
      // line instead of a stale or middle range box.
      selectionFrame = window.requestAnimationFrame(() => {
        updateSelection({ x: event.clientX, y: event.clientY });
      });
    };
    const finishWithoutPoint = () => {
      selectingRef.current = false;
      window.cancelAnimationFrame(selectionFrame);
      updateSelection();
    };
    const handlePointerDown = (event: PointerEvent) => {
      if (!markdownRef.current?.contains(event.target as Node)) return;
      selectingRef.current = true;
      setToolbar(null);
    };
    const handleWindowExit = (event: MouseEvent) => {
      if (selectingRef.current && event.relatedTarget === null) finishWithoutPoint();
    };
    window.addEventListener('pointerdown', handlePointerDown);
    window.addEventListener('pointerup', finishAtPoint);
    window.addEventListener('pointercancel', finishWithoutPoint);
    window.addEventListener('mouseout', handleWindowExit);
    window.addEventListener('blur', finishWithoutPoint);
    return () => {
      window.cancelAnimationFrame(selectionFrame);
      window.removeEventListener('pointerdown', handlePointerDown);
      window.removeEventListener('pointerup', finishAtPoint);
      window.removeEventListener('pointercancel', finishWithoutPoint);
      window.removeEventListener('mouseout', handleWindowExit);
      window.removeEventListener('blur', finishWithoutPoint);
    };
  }, [updateSelection]);

  useEffect(() => {
    const reposition = () => {
      setToolbar((current) => {
        if (!current || !toolbarRef.current) return current;
        const measured = toolbarRef.current.getBoundingClientRect();
        const next = selectionToolbarPosition(
          current.anchor,
          { width: measured.width, height: measured.height },
          visibleViewportBounds(),
        );
        return { ...current, ...next };
      });
    };
    const visual = window.visualViewport;
    window.addEventListener('resize', reposition);
    visual?.addEventListener('resize', reposition);
    visual?.addEventListener('scroll', reposition);
    return () => {
      window.removeEventListener('resize', reposition);
      visual?.removeEventListener('resize', reposition);
      visual?.removeEventListener('scroll', reposition);
    };
  }, []);

  const handleMarkConfusion = () => {
    if (!toolbar) return;
    createConfusion.mutate({
      projectSlug: slug,
      confusion: {
        quoteSnapshot: toolbar.text.slice(0, 500),
        charStart: 0,
        charEnd: toolbar.text.length,
      },
    });
    setToolbar(null);
    window.getSelection()?.removeAllRanges();
  };

  if (isLoading) {
    return <div className={s.loading}>加载中…</div>;
  }
  if (error) {
    return (
      <EmptyState
        title={`${zone} 阶段尚未产出`}
        description={
          <>
            调用 {zone} Agent 后产物会落到 <code>{zone.toLowerCase()}/{filename}</code>。
          </>
        }
      />
    );
  }
  return (
    <article className={s.host} ref={hostRef}>
      <header className={s.head}>
        <Tag tone="accent">{zone}</Tag>
        <code className={s.path}>{zone.toLowerCase()}/{filename}</code>
      </header>
      <div
        ref={markdownRef}
        className={s.markdown}
        // HTML is sanitised via DOMPurify in useMarkdown.
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {toolbar && createPortal((
        <div
          ref={toolbarRef}
          className={s.selectionBar}
          style={{
            top: toolbar.top,
            left: toolbar.left,
            width: 'max-content',
            maxWidth: Math.max(0, visibleViewportBounds().w - 16),
          }}
          role="toolbar"
          aria-label="选中文本操作"
        >
          <button
            className={s.selBtn}
            onClick={handleMarkConfusion}
            type="button"
          >
            标困惑
          </button>
          <button
            className={s.selBtn}
            onClick={() => {
              navigator.clipboard.writeText(toolbar.text);
              setToolbar(null);
            }}
            type="button"
          >
            复制
          </button>
          <button
            className={s.selBtnMuted}
            onClick={() => setToolbar(null)}
            type="button"
          >
            取消
          </button>
        </div>
      ), document.body)}
    </article>
  );
}
