import { useEffect, useMemo, useRef, useState } from 'react';
import { ChevronLeftIcon, ChevronRightIcon } from '@radix-ui/react-icons';

import { useConfusions, useCreateConfusion } from '@/api/confusions';
import { useExplainManifest, useExplainPage } from '@/api/learningArtifacts';
import { EmptyState } from '@/components/primitive/EmptyState';
import { OutputViewer } from '@/components/feature/project/OutputViewer';
import { ExplainInfographic } from './ExplainInfographic';
import { useMarkdown } from '@/hooks/useMarkdown';
import {
  applyHighlights,
  overlapsExisting,
  selectionToTextAnchor,
  type TextAnchor,
} from '@/lib/summaryHighlights';

import s from './ExplainReader.module.css';

interface ExplainReaderProps {
  projectSlug: string;
}

export function ExplainReader({ projectSlug }: ExplainReaderProps) {
  const manifestQuery = useExplainManifest(projectSlug);
  const pages = useMemo(
    () => [...(manifestQuery.data?.pages ?? [])].sort((a, b) => a.order - b.order),
    [manifestQuery.data?.pages],
  );
  const [pageID, setPageID] = useState<string>('');
  const pageTabRefs = useRef(new Map<string, HTMLButtonElement>());

  useEffect(() => {
    if (pages.length === 0) return;
    if (!pageID || !pages.some((page) => page.id === pageID)) {
      setPageID(pages[0].id);
    }
  }, [pageID, pages]);

  const currentIndex = Math.max(0, pages.findIndex((page) => page.id === pageID));
  const current = pages[currentIndex];

  useEffect(() => {
    if (!current) return;
    pageTabRefs.current.get(current.id)?.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'center',
    });
  }, [current]);

  if (manifestQuery.isLoading) {
    return <div className={s.loading}>正在检查多页讲解…</div>;
  }

  if (manifestQuery.error || pages.length === 0) {
    return <OutputViewer slug={projectSlug} zone="Explain" />;
  }

  const movePage = (offset: number) => {
    const nextIndex = Math.min(Math.max(currentIndex + offset, 0), pages.length - 1);
    setPageID(pages[nextIndex].id);
  };

  return (
    <div className={s.layout}>
      <header className={s.toc}>
        <div className={s.tocTitle}>{manifestQuery.data?.title}</div>
        <div className={s.tocNavigation}>
          <button
            type="button"
            className={s.slideButton}
            onClick={() => movePage(-1)}
            disabled={currentIndex === 0}
            aria-label="上一页"
          >
            <ChevronLeftIcon aria-hidden="true" />
          </button>

          <div
            className={s.tocScroller}
            role="tablist"
            aria-label="讲解页面"
            onKeyDown={(event) => {
              if (event.key === 'ArrowLeft') {
                event.preventDefault();
                movePage(-1);
              }
              if (event.key === 'ArrowRight') {
                event.preventDefault();
                movePage(1);
              }
            }}
          >
            {pages.map((page) => (
              <button
                key={page.id}
                ref={(node) => {
                  if (node) pageTabRefs.current.set(page.id, node);
                  else pageTabRefs.current.delete(page.id);
                }}
                type="button"
                role="tab"
                aria-selected={page.id === current.id}
                className={`${s.tocItem} ${page.id === current.id ? s.active : ''}`}
                onClick={() => setPageID(page.id)}
              >
                <span className={s.pageNode}>{page.order}</span>
                <span className={s.pageTitle}>{page.title}</span>
                {page.kind === 'followup' && <small>追问</small>}
              </button>
            ))}
          </div>

          <button
            type="button"
            className={s.slideButton}
            onClick={() => movePage(1)}
            disabled={currentIndex === pages.length - 1}
            aria-label="下一页"
          >
            <ChevronRightIcon aria-hidden="true" />
          </button>
        </div>
      </header>

      <main className={s.reader}>
        <ExplainPage projectSlug={projectSlug} file={current.file} />
      </main>

      {pages.length > 0 && (
        <ExplainInfographic projectSlug={projectSlug} visible={currentIndex === pages.length - 1} />
      )}
    </div>
  );
}

function ExplainPage({ projectSlug, file }: { projectSlug: string; file: string }) {
  const page = useExplainPage(projectSlug, file);
  const { html, mermaid } = useMarkdown(page.data ?? '');
  const hostRef = useRef<HTMLElement>(null);
  const markdownRef = useRef<HTMLDivElement>(null);
  const createConfusion = useCreateConfusion();
  const { data: allConfusions = [] } = useConfusions(projectSlug);
  const artifactID = `explain/${file}`;
  const pageConfusions = useMemo(
    () => allConfusions.filter((item) =>
      item.state !== 'deleted' && item.sourceArtifactId === artifactID,
    ),
    [allConfusions, artifactID],
  );
  const [selection, setSelection] = useState<
    (TextAnchor & { top: number; left: number; overlaps: boolean }) | null
  >(null);

  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    let cancelled = false;
    void import('mermaid').then(async (mod) => {
      if (cancelled) return;
      for (const block of mermaid) {
        const target = hostRef.current?.querySelector(`[data-mermaid-id="${block.id}"]`);
        if (!target) continue;
        try {
          const result = await mod.default.render(`${block.id}-explain-svg`, block.code);
          (target as HTMLElement).innerHTML = result.svg;
        } catch (error) {
          (target as HTMLElement).textContent = `Mermaid render error: ${(error as Error).message}`;
        }
      }
    });
    return () => { cancelled = true; };
  }, [html, mermaid]);

  useEffect(() => {
    if (!markdownRef.current) return;
    applyHighlights(markdownRef.current, pageConfusions.map((item) => ({
      id: item.id,
      charStart: item.charStart,
      charEnd: item.charEnd,
      quoteSnapshot: item.quoteSnapshot,
    })));
  }, [html, pageConfusions]);

  if (page.isLoading) return <div className={s.loading}>加载页面…</div>;
  if (page.error) return <EmptyState title="页面文件不存在" description={file} />;

  return (
    <article
      className={s.page}
      ref={hostRef}
      onMouseUp={() => {
        const selected = window.getSelection();
        if (!selected || selected.isCollapsed || !selected.toString().trim()) {
          setSelection(null);
          return;
        }
        const markdown = markdownRef.current;
        const range = selected.getRangeAt(0);
        if (!markdown || !markdown.contains(range.commonAncestorContainer)) {
          setSelection(null);
          return;
        }
        const anchor = selectionToTextAnchor(markdown, range);
        if (!anchor) return;
        const rect = selected.getRangeAt(0).getBoundingClientRect();
        const hostRect = hostRef.current?.getBoundingClientRect();
        setSelection({
          ...anchor,
          top: rect.top - (hostRect?.top ?? 0) - 42,
          left: rect.left - (hostRect?.left ?? 0) + rect.width / 2,
          overlaps: overlapsExisting(anchor, pageConfusions),
        });
      }}
    >
      <code>{file}</code>
      <div
        ref={markdownRef}
        className={s.markdown}
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {selection && (
        <div className={s.selectionBar} style={{ top: selection.top, left: selection.left }}>
          <button
            type="button"
            disabled={selection.overlaps}
            onClick={async () => {
              await createConfusion.mutateAsync({
                projectSlug,
                confusion: {
                  sourceArtifactId: artifactID,
                  quoteSnapshot: selection.text.slice(0, 500),
                  charStart: selection.start,
                  charEnd: Math.min(selection.end, selection.start + 500),
                },
              });
              setSelection(null);
              window.getSelection()?.removeAllRanges();
            }}
          >
            {selection.overlaps ? '已在摘要中' : '保存摘要'}
          </button>
          <button type="button" onClick={() => setSelection(null)}>取消</button>
        </div>
      )}
    </article>
  );
}
