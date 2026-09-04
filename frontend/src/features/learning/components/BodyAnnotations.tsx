import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { TrashIcon } from '@radix-ui/react-icons';

import { useAskAiSettings } from '@/features/settings';
import {
  type BodyAnnotation,
  useBodyAnnotations,
  useCreateBodyAnnotation,
  useDeleteBodyAnnotation,
} from '@/features/learning/api/learningWorkspace';
import { AskAiPanel } from '@/features/learning/components/AskAiPanel';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import { selectionEndpointRect, type SelectionRect } from '@/shared/lib/selectionGeometry';
import { applyHighlights, selectionToTextAnchor } from '@/shared/lib/summaryHighlights';
import { useAskAiStore } from '@/shared/store/slices/askAi';

import s from './BodyAnnotations.module.css';

interface SelectionState {
  text: string;
  start: number;
  end: number;
  prefix: string;
  suffix: string;
  rect: SelectionRect;
}

export function BodyAnnotations({ slug, content, assetVersionId, pageIndex }: { slug: string; content: string; assetVersionId: string; pageIndex?: number }) {
  const markdownRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<SelectionState | null>(null);
  const annotations = useBodyAnnotations(slug);
  const create = useCreateBodyAnnotation(slug);
  const remove = useDeleteBodyAnnotation(slug);
  const askAi = useAskAiStore();
  const settings = useAskAiSettings();

  useLayoutEffect(() => {
    const root = markdownRef.current;
    if (!root || pageIndex === undefined) return;
    const children = Array.from(root.children) as HTMLElement[];
    const hasPages = children.some((child) => child.tagName === 'H2');
    if (!hasPages) {
      for (const child of children) child.hidden = false;
      return;
    }
    let section = -1;
    for (const child of children) {
      if (child.tagName === 'H2') section += 1;
      child.hidden = section === pageIndex || (section < 0 && pageIndex === 0) ? false : true;
    }
  }, [content, pageIndex]);

  useEffect(() => {
    if (!markdownRef.current) return;
    applyHighlights(markdownRef.current, (annotations.data ?? []).map((item) => ({
      id: item.annotationId,
      charStart: item.anchors.start,
      charEnd: item.anchors.end,
      quoteSnapshot: item.quoteSnapshot,
    })));
  }, [annotations.data, content]);

  const captureSelection = useCallback((event?: PointerEvent) => {
    const selected = window.getSelection();
    const root = markdownRef.current;
    if (!selected || selected.isCollapsed || !root || selected.rangeCount === 0) {
      setSelection(null);
      return;
    }
    const range = selected.getRangeAt(0);
    const anchor = selectionToTextAnchor(root, range);
    const rect = selectionEndpointRect(range, event ? { x: event.clientX, y: event.clientY } : undefined);
    if (!anchor || !rect) {
      setSelection(null);
      return;
    }
    const full = root.textContent ?? '';
    setSelection({
      ...anchor,
      prefix: full.slice(Math.max(0, anchor.start - 40), anchor.start),
      suffix: full.slice(anchor.end, anchor.end + 40),
      rect,
    });
  }, []);

  useEffect(() => {
    const root = markdownRef.current;
    if (!root) return;
    const onPointerUp = (event: PointerEvent) => {
      if (root.contains(event.target as Node)) captureSelection(event);
    };
    root.addEventListener('pointerup', onPointerUp);
    return () => root.removeEventListener('pointerup', onPointerUp);
  }, [captureSelection, content]);

  const createFromSelection = async (): Promise<BodyAnnotation | null> => {
    if (!selection) return null;
    const response = await create.mutateAsync({
      assetVersionId,
      quoteSnapshot: selection.text.slice(0, 1000),
      anchors: { start: selection.start, end: selection.end, prefix: selection.prefix, suffix: selection.suffix },
    });
    window.getSelection()?.removeAllRanges();
    setSelection(null);
    return response.annotation;
  };

  const openQuestion = async () => {
	const currentSelection = selection;
    const annotation = await createFromSelection();
	if (!annotation || !currentSelection) return;
    askAi.openActive({
      projectSlug: slug,
      confusionId: annotation.annotationId,
      quote: annotation.quoteSnapshot,
	  anchor: currentSelection.rect,
      providerId: settings.data?.default ?? '',
      initialInput: `请解释「${annotation.quoteSnapshot.slice(0, 120)}」`,
    });
  };

  return (
    <>
      <MarkdownView ref={markdownRef} source={content} className={s.markdown} />
      {selection && createPortal(
        <div className={s.selectionBar} style={{ top: selection.rect.bottom + 8, left: selection.rect.left + selection.rect.width / 2 }} role="toolbar" aria-label="选中文本操作">
          <button type="button" onClick={() => void createFromSelection()} disabled={create.isPending}>保存批注</button>
          <button type="button" onClick={() => void openQuestion()} disabled={create.isPending}>问 AI</button>
        </div>,
        document.body,
      )}
      <aside className={s.panel} aria-label="正文批注">
        <header><h3>批注</h3><span>{annotations.data?.length ?? 0}</span></header>
        <p className={s.hint}>选中正文即可保存，或就原文向 AI 提问。</p>
        <div className={s.list}>
          {annotations.isLoading && <p className={s.empty}>加载中…</p>}
          {!annotations.isLoading && !annotations.data?.length && <p className={s.empty}>还没有批注。</p>}
          {(annotations.data ?? []).map((annotation) => (
            <article key={annotation.annotationId} className={s.item}>
              <blockquote>{annotation.quoteSnapshot}</blockquote>
              <div>
                {annotation.ask?.messages?.length ? (
                  <button type="button" onClick={(event) => {
                    const rect = event.currentTarget.getBoundingClientRect();
                    askAi.openReview({
                      projectSlug: slug,
                      confusionId: annotation.annotationId,
                      quote: annotation.quoteSnapshot,
                      anchor: rect,
                      messages: annotation.ask!.messages.map((message) => ({
						id: message.id,
						role: message.role === 'learner' ? 'user' as const : 'assistant' as const,
						content: message.content,
						createdAt: message.createdAt,
					  })),
                    });
                  }}>查看答疑</button>
                ) : <span>正文批注</span>}
                <button type="button" aria-label="删除批注" title="删除批注" onClick={() => remove.mutate(annotation.annotationId)}><TrashIcon /></button>
              </div>
            </article>
          ))}
        </div>
      </aside>
      <AskAiPanel />
    </>
  );
}
