import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { ChevronLeftIcon, ChevronRightIcon } from '@radix-ui/react-icons';

import { useConfusions, useCreateConfusion } from '@/api/confusions';
import { useExplainManifest, useExplainPage } from '@/api/learningArtifacts';
import { useRecordReading } from '@/api/activity';
import { EmptyState } from '@/components/primitive/EmptyState';
import { OutputViewer } from '@/components/feature/project/OutputViewer';
import { ExplainInfographic } from './ExplainInfographic';
import { AskAiPanel } from './AskAiPanel';
import { useAskAiSettings } from '@/api/askAi';
import { useAskAiStore } from '@/store/slices/askAi';
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
            {pages.map((page) => {
              const label = page.kind === 'followup' ? `${page.title}（追问）` : page.title;
              return (
                <button
                  key={page.id}
                  ref={(node) => {
                    if (node) pageTabRefs.current.set(page.id, node);
                    else pageTabRefs.current.delete(page.id);
                  }}
                  type="button"
                  role="tab"
                  aria-selected={page.id === current.id}
                  title={label}
                  aria-label={label}
                  className={`${s.tocItem} ${page.id === current.id ? s.active : ''} ${page.kind === 'followup' ? s.followup : ''}`}
                  onClick={() => setPageID(page.id)}
                >
                  <span className={s.pageNode}>{page.order}</span>
                </button>
              );
            })}
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
        <ExplainPage
          projectSlug={projectSlug}
          file={current.file}
          isSummary={currentIndex === pages.length - 1}
        />
      </main>

      {pages.length > 0 && (
        <ExplainInfographic projectSlug={projectSlug} visible={currentIndex === pages.length - 1} />
      )}
      <AskAiPanel />
    </div>
  );
}

function ExplainPage({
  projectSlug,
  file,
  isSummary,
}: {
  projectSlug: string;
  file: string;
  isSummary?: boolean;
}) {
  const page = useExplainPage(projectSlug, file);
  const { html, mermaid } = useMarkdown(page.data ?? '');
  const hostRef = useRef<HTMLElement>(null);
  const markdownRef = useRef<HTMLDivElement>(null);
  const createConfusion = useCreateConfusion();
  const askAi = useAskAiStore();
  const askAiSettings = useAskAiSettings();
  const { data: allConfusions = [] } = useConfusions(projectSlug);
  const artifactID = `explain/${file}`;
  useEffectiveReading(projectSlug, artifactID, file, markdownRef);
  const pageConfusions = useMemo(
    () => allConfusions.filter((item) =>
      item.state !== 'deleted' && item.sourceArtifactId === artifactID,
    ),
    [allConfusions, artifactID],
  );
  const selectionBarRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<
    (TextAnchor & {
      top: number;
      left: number;
      screenTop: number;
      screenLeft: number;
      rect: SelectionBox;
      overlaps: boolean;
    }) | null
  >(null);
  const openBrowserSearch = (text: string) => {
    const engine = askAiSettings.data?.searchEngine === 'bing' ? 'bing' : 'google';
    const base = engine === 'bing' ? 'https://www.bing.com/search?q=' : 'https://www.google.com/search?q=';
    window.open(base + encodeURIComponent(text), '_blank', 'noopener,noreferrer');
  };

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

  useLayoutEffect(() => {
    if (!selection || !selectionBarRef.current) return;
    const barRect = selectionBarRef.current.getBoundingClientRect();
    const next = toolbarPosition(selection.rect, { width: barRect.width, height: barRect.height });
    if (Math.abs(next.top - selection.top) > 1 || Math.abs(next.left - selection.left) > 1) {
      setSelection({ ...selection, ...next });
    }
  }, [selection]);

  if (page.isLoading) return <div className={s.loading}>加载页面…</div>;
  if (page.error) return <EmptyState title="页面文件不存在" description={file} />;

  return (
    <article
      className={isSummary ? `${s.page} ${s.summary}` : s.page}
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
        const rect = selectionRect(range);
        if (!rect) {
          setSelection(null);
          return;
        }
        const rectBox = toSelectionBox(rect);
        const toolbar = toolbarPosition(rectBox);
        setSelection({
          ...anchor,
          top: toolbar.top,
          left: toolbar.left,
          screenTop: rectBox.top,
          screenLeft: rectBox.left + rectBox.width / 2,
          rect: rectBox,
          overlaps: overlapsExisting(anchor, pageConfusions),
        });
      }}
    >
      {!isSummary && <code>{file}</code>}
      <div
        ref={markdownRef}
        className={s.markdown}
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {selection && createPortal((
        <div ref={selectionBarRef} className={s.selectionBar} style={{ top: selection.top, left: selection.left }}>
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
                anchor: {
                  top: selection.rect.top,
                  left: selection.rect.left + selection.rect.width / 2,
                  right: selection.rect.right,
                  bottom: selection.rect.bottom,
                  width: selection.rect.width,
                  height: selection.rect.height,
                },
                providerId: askAiSettings.data?.default ?? '',
                initialInput: `请你解释「${selection.text.slice(0, 120)}」`,
              });
              setSelection(null);
              window.getSelection()?.removeAllRanges();
            }}
          >
            问 AI
          </button>
          <button
            type="button"
            onClick={() => {
              openBrowserSearch(selection.text);
              setSelection(null);
              window.getSelection()?.removeAllRanges();
            }}
          >
            浏览器搜索
          </button>
          <button type="button" onClick={() => setSelection(null)}>取消</button>
        </div>
      ), document.body)}
    </article>
  );
}

const EFFECTIVE_READING_MS = 3 * 60 * 1000;

function useEffectiveReading(
  projectSlug: string,
  artifactID: string,
  title: string,
  contentRef: React.RefObject<HTMLElement>,
) {
  const { mutate: recordReading } = useRecordReading(projectSlug);
  const sentRef = useRef('');

  useEffect(() => {
    let engaged = false;
    let accumulated = 0;
    let previous = Date.now();
    const markEngaged = (event: Event) => {
      const target = event.target as Node | null;
      if (!target || contentRef.current?.contains(target) || event.type === 'scroll') engaged = true;
    };
    const timer = window.setInterval(() => {
      const now = Date.now();
      const elapsed = Math.min(now - previous, 6_000);
      previous = now;
      if (document.visibilityState === 'visible' && engaged) accumulated += elapsed;
      const localDate = new Date().toLocaleDateString('sv-SE');
      const id = `${artifactID}:${localDate}`;
      if (accumulated >= EFFECTIVE_READING_MS && sentRef.current !== id) {
        sentRef.current = id;
        recordReading({ id, sourceId: artifactID, title: '有效阅读', detail: title, activityDelta: 2 });
      }
    }, 5_000);
    document.addEventListener('scroll', markEngaged, true);
    document.addEventListener('pointerdown', markEngaged, true);
    document.addEventListener('selectionchange', markEngaged);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('scroll', markEngaged, true);
      document.removeEventListener('pointerdown', markEngaged, true);
      document.removeEventListener('selectionchange', markEngaged);
    };
  }, [artifactID, contentRef, recordReading, title]);
}

interface SelectionBox {
  top: number;
  right: number;
  bottom: number;
  left: number;
  width: number;
  height: number;
}

function selectionRect(range: Range): DOMRect | null {
  const rects = Array.from(range.getClientRects()).filter((rect) =>
    rect.width > 0 && rect.height > 0 &&
    Number.isFinite(rect.top) && Number.isFinite(rect.left),
  );
  if (rects.length > 0) {
    return rects.reduce((best, rect) => {
      if (rect.top < best.top) return rect;
      if (Math.abs(rect.top - best.top) < 1 && rect.left < best.left) return rect;
      return best;
    }, rects[0]);
  }

  const rect = range.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0 || !Number.isFinite(rect.top) || !Number.isFinite(rect.left)) {
    return null;
  }
  return rect;
}

function toSelectionBox(rect: DOMRect): SelectionBox {
  return {
    top: rect.top,
    right: rect.right,
    bottom: rect.bottom,
    left: rect.left,
    width: rect.width,
    height: rect.height,
  };
}

function toolbarPosition(rect: SelectionBox, measured?: { width: number; height: number }) {
  const toolbarWidth = measured?.width ?? 360;
  const toolbarHeight = measured?.height ?? 46;
  const halfToolbarWidth = toolbarWidth / 2;
  const margin = 8;
  const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0;
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0;
  const spaceAbove = rect.top - margin;
  const spaceBelow = viewportHeight - rect.bottom - margin;
  const preferredTop = spaceBelow >= toolbarHeight || spaceBelow > spaceAbove
    ? rect.bottom + margin
    : rect.top - toolbarHeight - margin;
  return {
    top: Math.min(Math.max(preferredTop, margin), Math.max(margin, viewportHeight - toolbarHeight - margin)),
    left: Math.min(
      Math.max(rect.left + rect.width / 2, halfToolbarWidth + margin),
      Math.max(halfToolbarWidth + margin, viewportWidth - halfToolbarWidth - margin),
    ),
  };
}
