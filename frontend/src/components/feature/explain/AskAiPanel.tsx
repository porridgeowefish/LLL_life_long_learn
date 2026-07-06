import { useEffect, useRef, useState } from 'react';
import { Cross2Icon } from '@radix-ui/react-icons';

import { streamAskAi, summarizeAsk, useAskAiSettings } from '@/api/askAi';
import { useAskAiStore } from '@/store/slices/askAi';
import { MarkdownView } from '@/components/primitive/MarkdownView';
import { flipPosition } from '@/lib/floatingPosition';

import s from './AskAiPanel.module.css';

const PANEL_SIZE = { width: 440, height: 520 };

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

  if (!store.open) return null;

  const pos = flipPosition(store.anchor, PANEL_SIZE, { w: window.innerWidth, h: window.innerHeight });

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
          else if (f.type === 'done') store.finishStream();
        },
      });
    } catch {
      /* aborted or errored */
    } finally {
      store.finishStream();
      abortRef.current = null;
    }
  };

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

  const searchURL =
    (settingsQuery.data?.searchEngine === 'bing' ? 'https://www.bing.com/search?q=' : 'https://www.google.com/search?q=') +
    encodeURIComponent(store.quote);

  return (
    <div className={s.panel} style={{ top: pos.top, left: pos.left }} role="dialog" aria-label="问 AI">
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
    </div>
  );
}
