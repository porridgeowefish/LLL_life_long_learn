import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';

import { useMarkdown } from '@/hooks/useMarkdown';
import { qk } from '@/api/queryKeys';
import { http } from '@/api/client';
import { useCreateConfusion } from '@/api/confusions';
import { ZONE_FILENAME, type ZoneName } from '@/types/domain';
import { EmptyState } from '@/components/primitive/EmptyState';
import { Tag } from '@/components/primitive/Tag';

import s from './OutputViewer.module.css';

interface OutputViewerProps {
  slug: string;
  zone: ZoneName;
}

// Selection toolbar actions for Explain zone text selection.
interface SelectionToolbar {
  text: string;
  rect: { top: number; left: number };
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

  // Selection toolbar state — only for Explain zone.
  const [toolbar, setToolbar] = useState<SelectionToolbar | null>(null);
  const createConfusion = useCreateConfusion();

  // After HTML is set, replace each mermaid placeholder with a rendered SVG.
  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    let cancelled = false;
    void import('mermaid').then(async (mod) => {
      if (cancelled) return;
      const mermaidApi = mod.default;
      for (const block of mermaid) {
        const placeholder = hostRef.current?.querySelector(
          `[data-mermaid-id="${block.id}"]`,
        );
        if (!placeholder) continue;
        try {
          const result = await mermaidApi.render(`${block.id}-svg`, block.code);
          (placeholder as HTMLElement).innerHTML = result.svg;
        } catch (err) {
          (placeholder as HTMLElement).textContent = `Mermaid render error: ${(err as Error).message}`;
        }
      }
    });
    return () => {
      cancelled = true;
    };
  }, [html, mermaid]);

  // Listen for text selection in Explain zone → show floating toolbar.
  const handleMouseUp = useCallback(() => {
    if (zone !== 'Explain') return;
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || !sel.toString().trim()) {
      setToolbar(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const rect = range.getBoundingClientRect();
    setToolbar({
      text: sel.toString().trim(),
      rect: { top: rect.top + window.scrollY, left: rect.left + rect.width / 2 },
    });
  }, [zone]);

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
        className={s.markdown}
        onMouseUp={handleMouseUp}
        // eslint-disable-next-line react/no-danger -- HTML is sanitised via DOMPurify in useMarkdown.
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {toolbar && (
        <div
          className={s.selectionBar}
          style={{ top: toolbar.rect.top - 40, left: toolbar.rect.left }}
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
      )}
    </article>
  );
}
