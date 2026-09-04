// MarkdownEditor — split-pane markdown editor with live preview.
// Reuses useMarkdown hook for rendering (KaTeX + Mermaid + DOMPurify).
//
// Modes:
//   - 'split' (default): textarea left + preview right, draggable divider
//   - 'preview': preview only (read-only view)
//   - 'edit': textarea only (maximise writing space)
//
// Draft persistence: callers may pass initialMarkdown + onSave for full control,
// or let the editor manage localStorage drafts keyed by project + path.

import { useCallback, useEffect, useRef, useState } from 'react';
import clsx from 'clsx';

import { useMarkdown } from '@/hooks/useMarkdown';
import { mountMermaidBlocks } from '@/lib/mermaidRenderer';
import { useProjectFile, useFileWrite } from '@/api/files';
import { Button } from './Button';

import s from './MarkdownEditor.module.css';

type Mode = 'split' | 'edit' | 'preview';

interface MarkdownEditorProps {
  /** Project slug — used for file loading and draft keys. */
  slug: string;
  /** Relative path under the project, e.g. "summary/notes.md". */
  relPath: string;
  /** Initial markdown (overrides file load if provided). */
  initialMarkdown?: string;
  /** Display label for the header. */
  label?: string;
  /** Starting mode. */
  defaultMode?: Mode;
  /** Extra class on the root. */
  className?: string;
  /** Minimum height in px (default 300). */
  minHeight?: number;
}

const DRAFT_PREFIX = 'lll.draft';

function draftKey(slug: string, relPath: string) {
  return `${DRAFT_PREFIX}.${slug}.${relPath}`;
}

function loadDraft(slug: string, relPath: string): string | null {
  try {
    return localStorage.getItem(draftKey(slug, relPath));
  } catch {
    return null;
  }
}

function saveDraft(slug: string, relPath: string, text: string) {
  try {
    localStorage.setItem(draftKey(slug, relPath), text);
  } catch {
    // quota exceeded — silently ignore
  }
}

function clearDraft(slug: string, relPath: string) {
  try {
    localStorage.removeItem(draftKey(slug, relPath));
  } catch {
    // ignore
  }
}

export function MarkdownEditor({
  slug,
  relPath,
  initialMarkdown,
  label,
  defaultMode = 'split',
  className,
  minHeight = 300,
}: MarkdownEditorProps) {
  const [mode, setMode] = useState<Mode>(defaultMode);
  const [source, setSource] = useState(initialMarkdown ?? '');
  const [dirty, setDirty] = useState(false);
  const [dividerPos, setDividerPos] = useState(50); // percent
  const dragging = useRef(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Load file from server if no initialMarkdown provided.
  const { data: fileData, isLoading } = useProjectFile(
    initialMarkdown === undefined ? slug : undefined,
    relPath,
  );

  // Sync server data → source (once).
  useEffect(() => {
    if (fileData !== undefined && fileData !== null) {
      // Check for a localStorage draft (user may have unsaved changes).
      const draft = loadDraft(slug, relPath);
      if (draft !== null) {
        setSource(draft);
        setDirty(true);
      } else {
        setSource(fileData);
        setDirty(false);
      }
    }
  }, [fileData, slug, relPath]);

  // If initialMarkdown is provided, use it directly.
  useEffect(() => {
    if (initialMarkdown !== undefined) {
      setSource(initialMarkdown);
    }
  }, [initialMarkdown]);

  const save = useFileWrite();

  const handleChange = useCallback(
    (next: string) => {
      setSource(next);
      setDirty(true);
      saveDraft(slug, relPath, next);
    },
    [slug, relPath],
  );

  const handleSave = useCallback(() => {
    save.mutate(
      { slug, relPath, body: source },
      {
        onSuccess: () => {
          setDirty(false);
          clearDraft(slug, relPath);
        },
      },
    );
  }, [save, slug, relPath, source]);

  // Divider drag handler.
  const onDividerDown = useCallback(() => {
    dragging.current = true;
    const onMove = (e: MouseEvent) => {
      if (!dragging.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const pct = ((e.clientX - rect.left) / rect.width) * 100;
      setDividerPos(Math.max(20, Math.min(80, pct)));
    };
    const onUp = () => {
      dragging.current = false;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }, []);

  const { html, mermaid } = useMarkdown(source);
  const previewRef = useRef<HTMLDivElement>(null);

  // Mermaid rendering after HTML injection (same pattern as OutputViewer).
  useEffect(() => {
    if (!previewRef.current || mermaid.length === 0) return;
    return mountMermaidBlocks(previewRef.current, mermaid, 'markdown-editor');
  }, [html, mermaid]);

  return (
    <div
      className={clsx(s.root, className)}
      style={{ minHeight }}
      ref={containerRef}
    >
      <header className={s.head}>
        <div className={s.headLeft}>
          {label && <span className={s.label}>{label}</span>}
          <code className={s.path}>{relPath}</code>
          {dirty && <span className={s.dirty}>未保存</span>}
        </div>
        <div className={s.headRight}>
          <div className={s.modeTabs}>
            {(['edit', 'split', 'preview'] as const).map((m) => (
              <button
                key={m}
                className={clsx(s.modeBtn, mode === m && s.modeActive)}
                onClick={() => setMode(m)}
                title={m === 'edit' ? '编辑' : m === 'split' ? '分栏' : '预览'}
              >
                {m === 'edit' ? '✏️' : m === 'split' ? '⬜' : '👁'}
              </button>
            ))}
          </div>
          <Button
            size="sm"
            variant={dirty ? 'primary' : 'outline'}
            onClick={handleSave}
            loading={save.isPending}
            disabled={!dirty}
          >
            {dirty ? '保存' : '已保存'}
          </Button>
        </div>
      </header>

      <div className={s.body}>
        {mode !== 'preview' && (
          <div
            className={s.editPane}
            style={mode === 'split' ? { width: `${dividerPos}%` } : { flex: 1 }}
          >
            <textarea
              className={s.textarea}
              value={isLoading ? '加载中…' : source}
              onChange={(e) => handleChange(e.target.value)}
              spellCheck={false}
              placeholder="在这里写 Markdown…"
            />
          </div>
        )}

        {mode === 'split' && (
          <div
            className={s.divider}
            onMouseDown={onDividerDown}
            role="separator"
            aria-valuenow={dividerPos}
          />
        )}

        {mode !== 'edit' && (
          <div
            className={s.previewPane}
            style={
              mode === 'split'
                ? { width: `${100 - dividerPos}%` }
                : { flex: 1 }
            }
          >
            <div
              ref={previewRef}
              className={s.previewContent}
              dangerouslySetInnerHTML={{ __html: html }}
            />
          </div>
        )}
      </div>

      {save.error && (
        <div className={s.error}>保存失败：{(save.error as Error).message}</div>
      )}
    </div>
  );
}
