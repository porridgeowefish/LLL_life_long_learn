import { type CSSProperties, useEffect, useRef, useState } from 'react';
import { Cross2Icon } from '@radix-ui/react-icons';

import { streamAskAi, summarizeAsk, useAskAiSettings } from '@/api/askAi';
import { useAskAiStore } from '@/store/slices/askAi';
import { MarkdownView } from '@/components/primitive/MarkdownView';
import { flipPosition } from '@/lib/floatingPosition';

import s from './AskAiPanel.module.css';

const PANEL_MAX_SIZE = { width: 460, height: 540 };

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
  const panelSize = {
    width: Math.min(PANEL_MAX_SIZE.width, viewport.w - 16),
    height: Math.min(PANEL_MAX_SIZE.height, viewport.h - 16),
  };
  const pos = flipPosition(store.anchor, panelSize, viewport);
  const panelStyle: CSSProperties = store.reviewMode
    ? {
      top: Math.min(Math.max(store.anchor.top, 8), Math.max(8, viewport.h - panelSize.height - 8)),
      right: 8,
    }
    : { top: pos.top, left: pos.left };

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
          else if (f.type === 'thinking') store.appendDelta('thinking', f.content);
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

  const searchURL =
    (settingsQuery.data?.searchEngine === 'bing' ? 'https://www.bing.com/search?q=' : 'https://www.google.com/search?q=') +
    encodeURIComponent(store.quote);

  return (
    <div className={s.panel} style={panelStyle} role="dialog" aria-label={store.reviewMode ? '答疑回顾' : '问 AI'}>
      <header className={s.header}>
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
              {m.thinking ? (
                <details className={s.thinking}>
                  <summary>思考过程</summary>
                  <div className={s.thinkingBody}>{m.thinking}</div>
                </details>
              ) : null}
              <MarkdownView source={m.content || (store.streaming ? '…' : '')} className={s.answer} />
            </div>
          ),
        )}
      </div>

      {!store.reviewMode && (
        <footer className={s.footer}>
          <a className={s.searchLink} href={searchURL} target="_blank" rel="noreferrer" title="在浏览器搜索">
            搜索
          </a>
          <input
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
    </div>
  );
}
