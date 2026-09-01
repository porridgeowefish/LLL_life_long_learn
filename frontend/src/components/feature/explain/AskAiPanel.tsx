import { type CSSProperties, type PointerEvent as ReactPointerEvent, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { Cross2Icon } from '@radix-ui/react-icons';

import { streamAskAi, summarizeAsk, useAskAiSettings } from '@/api/askAi';
import { useAskAiStore } from '@/store/slices/askAi';
import { MarkdownView } from '@/components/primitive/MarkdownView';

import s from './AskAiPanel.module.css';

const ACTIVE_PANEL_SIZE = { width: 460, height: 380 };
const REVIEW_PANEL_SIZE = { width: 520, height: 540 };

function formatAskAiError(error: unknown): string {
  const status = error instanceof Error && 'status' in error ? (error as { status: number }).status : undefined;
  const message = error instanceof Error ? error.message : String(error);
  if (status === 0) {
    return '回答失败：无法连接 Ask-AI 服务，请确认本地后端还在运行。';
  }
  if (message.includes('ask-ai not configured')) {
    return '回答失败：Ask-AI 模型源还没配置。请到设置里添加模型源，并先点“探测”确认可用。';
  }
  if (message.includes('provider not found')) {
    return '回答失败：当前选择的模型源不存在。请切换一个可用模型源后重试。';
  }
  if (message.trim()) {
    return `回答失败：${message}`;
  }
  return '回答失败：模型源没有返回有效内容，请稍后重试。';
}

export function AskAiPanel() {
  const store = useAskAiStore();
  const settingsQuery = useAskAiSettings();
  const [input, setInput] = useState('');
  const abortRef = useRef<AbortController | null>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (store.open && !store.reviewMode) {
      setInput(store.initialInput);
      window.setTimeout(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      }, 0);
    }
  }, [store.confusionId, store.initialInput, store.open, store.reviewMode]);

  // Auto-scroll on new content. Guard for jsdom (no scrollTo) — no-op there.
  useEffect(() => {
    const el = listRef.current;
    if (el && typeof el.scrollTo === 'function') {
      el.scrollTo({ top: el.scrollHeight });
    }
  }, [store.messages, store.streaming]);

  const stop = () => {
    abortRef.current?.abort();
    store.finishStream();
  };

  const close = () => {
    if (!store.reviewMode && store.messages.some((m) => m.role === 'assistant')) {
      void summarizeAsk(store.projectSlug, store.confusionId);
    }
    stop();
    store.close();
  };

  useEffect(() => {
    if (!store.open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') close();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });

  if (!store.open) return null;

  const viewport = { w: window.innerWidth, h: window.innerHeight };
  const defaultPanelSize = panelSize(store.reviewMode, viewport);
  const geometry = store.geometry ?? initialPanelGeometry(store.anchor, defaultPanelSize, viewport, store.reviewMode);
  const clampGeometry = (next: typeof geometry) => {
    const width = Math.min(Math.max(next.width, 320), Math.max(320, viewport.w - 16));
    const height = Math.min(Math.max(next.height, 280), Math.max(280, viewport.h - 16));
    return {
      width,
      height,
      top: Math.min(Math.max(next.top, 8), Math.max(8, viewport.h - height - 8)),
      left: Math.min(Math.max(next.left, 8), Math.max(8, viewport.w - width - 8)),
    };
  };
  const beginDrag = (event: ReactPointerEvent<HTMLElement>) => {
    const target = event.target as HTMLElement;
    if (target.closest('button,select,input,a')) return;
    const start = { x: event.clientX, y: event.clientY, ...geometry };
    const move = (moveEvent: PointerEvent) => {
      store.setGeometry(clampGeometry({
        ...start,
        top: start.top + moveEvent.clientY - start.y,
        left: start.left + moveEvent.clientX - start.x,
      }));
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };
  const beginResize = (event: ReactPointerEvent<HTMLSpanElement>) => {
    event.preventDefault();
    const start = { x: event.clientX, y: event.clientY, ...geometry };
    const move = (moveEvent: PointerEvent) => {
      store.setGeometry(clampGeometry({
        ...start,
        width: start.width + moveEvent.clientX - start.x,
        height: start.height + moveEvent.clientY - start.y,
      }));
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };
  const panelStyle: CSSProperties = {
    top: geometry.top,
    left: geometry.left,
    width: geometry.width,
    height: geometry.height,
  };

  const send = async () => {
    const content = input.trim();
    if (!content || store.streaming || store.reviewMode) return;
    setInput('');
    store.appendUser(content);
    store.startAssistant();
    const ac = new AbortController();
    abortRef.current = ac;
    try {
      await streamAskAi(store.projectSlug, store.confusionId, {
        content,
        providerId: store.providerId || undefined,
      }, {
        signal: ac.signal,
        onFrame: (f) => {
          if (f.type === 'text') store.appendDelta('text', f.content);
          else if (f.type === 'error') {
            store.appendDelta('text', formatAskAiError(new Error(f.content)));
            store.finishStream();
          } else if (f.type === 'done') store.finishStream();
        },
      });
    } catch (err) {
      if (!ac.signal.aborted) {
        store.appendDelta('text', formatAskAiError(err));
      }
    } finally {
      store.finishStream();
      abortRef.current = null;
    }
  };

  return createPortal((
    <div className={s.panel} style={panelStyle} role="dialog" aria-label={store.reviewMode ? '答疑回顾' : '问 AI'}>
      <header className={s.header} onPointerDown={beginDrag}>
        <span className={s.title}>{store.reviewMode ? '答疑回顾' : '问 AI'}</span>
        {!store.reviewMode && settingsQuery.data && (
          <select
            className={s.provider}
            value={store.providerId}
            onChange={(e) => store.setProvider(e.target.value)}
            aria-label="模型源"
          >
            {settingsQuery.data.providers.map((p) => (
              <option key={p.id} value={p.id}>{p.name || p.id}</option>
            ))}
          </select>
        )}
        <button type="button" className={s.iconBtn} onClick={close} aria-label="关闭" title="关闭">
          <Cross2Icon />
        </button>
      </header>

      <div className={s.messages} ref={listRef}>
        {store.messages.length === 0 && !store.reviewMode && (
          <div className={s.hint}>针对选中文字提问，回答会流式输出并保存到这条疑问。</div>
        )}
        {store.messages.map((m) =>
          m.role === 'user' ? (
            <div key={m.id} className={s.user}>{m.content}</div>
          ) : (
            <div key={m.id} className={s.assistant}>
              <MarkdownView source={m.content || (store.streaming ? '…' : '')} className={s.answer} />
            </div>
          ),
        )}
      </div>

      {!store.reviewMode && (
        <footer className={s.footer}>
          <input
            ref={inputRef}
            className={s.input}
            placeholder="追问…（Enter 发送）"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                void send();
              }
            }}
          />
          {store.streaming ? (
            <button type="button" className={s.stopBtn} onClick={stop}>停止</button>
          ) : (
            <button type="button" className={s.sendBtn} onClick={() => void send()} disabled={!input.trim()}>
              发送
            </button>
          )}
        </footer>
      )}
      {store.reviewMode && (
        <footer className={s.footer}>
          <button type="button" className={s.reviewCloseBtn} onClick={close}>
            关闭
          </button>
        </footer>
      )}
      <span className={s.resizeHandle} onPointerDown={beginResize} aria-hidden="true" />
    </div>
  ), document.body);
}

function panelSize(reviewMode: boolean, viewport: { w: number; h: number }) {
  const target = reviewMode ? REVIEW_PANEL_SIZE : ACTIVE_PANEL_SIZE;
  return {
    width: Math.min(target.width, Math.max(320, viewport.w - 16)),
    height: Math.min(target.height, Math.max(280, viewport.h - 16)),
  };
}

function initialPanelGeometry(
  anchor: { top: number; left: number; right?: number; bottom?: number },
  size: { width: number; height: number },
  viewport: { w: number; h: number },
  reviewMode: boolean,
) {
  const margin = 12;
  if (reviewMode) {
    return clampPanel({
      top: Math.min(88, Math.max(margin, viewport.h * 0.12)),
      left: (viewport.w - size.width) / 2,
      ...size,
    }, viewport, margin);
  }

  const leftEdge = anchor.left;
  const rightEdge = anchor.right ?? anchor.left;
  const topEdge = anchor.top;
  const bottomEdge = anchor.bottom ?? anchor.top;

  if (viewport.w - rightEdge >= size.width + margin) {
    return clampPanel({
      top: topEdge - size.height * 0.25,
      left: rightEdge + margin,
      ...size,
    }, viewport, margin);
  }

  if (leftEdge >= size.width + margin) {
    return clampPanel({
      top: topEdge - size.height * 0.25,
      left: leftEdge - size.width - margin,
      ...size,
    }, viewport, margin);
  }

  const top = viewport.h - bottomEdge >= size.height + margin
    ? bottomEdge + margin
    : topEdge - size.height - margin;
  return clampPanel({
    top,
    left: anchor.left - size.width / 2,
    ...size,
  }, viewport, margin);
}

function clampPanel<T extends { top: number; left: number; width: number; height: number }>(
  geometry: T,
  viewport: { w: number; h: number },
  margin = 8,
) {
  const width = Math.min(Math.max(geometry.width, 320), Math.max(320, viewport.w - margin * 2));
  const height = Math.min(Math.max(geometry.height, 260), Math.max(260, viewport.h - margin * 2));
  return {
    ...geometry,
    width,
    height,
    top: Math.min(Math.max(geometry.top, margin), Math.max(margin, viewport.h - height - margin)),
    left: Math.min(Math.max(geometry.left, margin), Math.max(margin, viewport.w - width - margin)),
  };
}
