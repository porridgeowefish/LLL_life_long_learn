import { FormEvent, memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { CopyIcon, DownloadIcon, EditIcon, RegenerateIcon, SearchIcon, SteerIcon, TrashIcon } from '@/shared/primitive/Icons';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import {
  discardQueuedTurn,
  editQueuedTurn,
  queueTeacherTurn,
  regenerateTeacherResponse,
  stopTeacherResponse,
  steerQueuedTurn,
  streamTeacherTurn,
  resumeTeacherTurn,
  learningWorkspaceKeys,
  useAssistantTasks,
  useConversation,
  useSources,
  type AssistantTask,
  type ConversationMessage,
  type QueuedTurn,
  type TeacherStreamFrame,
} from '@/features/learning/api/learningWorkspace';
import { useAskAiSettings } from '@/features/settings';

import { SourceCitations } from './SourceCitations';
import { LearningProgress, readLearningProgress, withoutLearningProgress } from './LearningProgress';
import s from './TeacherView.module.css';

interface SearchStatus {
  query: string;
  count?: number;
  failed?: boolean;
}

interface LiveResponse {
  learner: ConversationMessage;
  teacherId?: string;
  responseId?: string;
  text: string;
  reasoning: string;
  taskIds: string[];
  searches: SearchStatus[];
  notice?: string;
  reading?: { operationId: string; operation: string; sourceId?: string; displayRange?: string };
}

const STREAM_FLUSH_MS = 64;
const STREAM_SCROLL_MS = 96;

export function TeacherView({ slug, title }: { slug: string; title: string }) {
  const queryClient = useQueryClient();
  const conversation = useConversation(slug);
  const tasksQuery = useAssistantTasks(slug);
  const sourcesQuery = useSources(slug);
  const modelSettings = useAskAiSettings();
  const [input, setInput] = useState('');
  const [providerId, setProviderId] = useState(() => localStorage.getItem(`lll.teacher.provider.${slug}`) ?? '');
  const [live, setLive] = useState<LiveResponse | null>(null);
  const [error, setError] = useState('');
  const [editingQueue, setEditingQueue] = useState<{ queueId: string; content: string } | null>(null);
  const [copiedMessageId, setCopiedMessageId] = useState('');
  const controller = useRef<AbortController | null>(null);
  const transcriptRef = useRef<HTMLDivElement>(null);
  const transcriptContentRef = useRef<HTMLDivElement>(null);
  const followStream = useRef(true);
  const scrollFrame = useRef<number | null>(null);
  const scrollTimer = useRef<number | null>(null);
  const pendingDeltas = useRef({ text: '', reasoning: '' });
  const deltaTimer = useRef<number | null>(null);
  const reasoningOpen = useRef(new Map<string, boolean>());
  const getReasoningOpen = useCallback((messageID: string) => reasoningOpen.current.get(messageID) ?? false, []);
  const setReasoningOpen = useCallback((messageID: string, open: boolean) => {
    reasoningOpen.current.set(messageID, open);
  }, []);
  const tasksById = useMemo(() => new Map((tasksQuery.data ?? []).map((task) => [task.id, task])), [tasksQuery.data]);
  const sourceNames = useMemo(() => new Map((sourcesQuery.data ?? []).map((source) => [source.sourceId, source.displayName])), [sourcesQuery.data]);
  const availableSources = useMemo(() => (sourcesQuery.data ?? []).filter((source) => source.status === 'ready'), [sourcesQuery.data]);
  const availableModels = useMemo(() => (modelSettings.data?.providers ?? []).filter((provider) => provider.id && provider.model && provider.baseURL), [modelSettings.data?.providers]);
  const conversationReady = Boolean(conversation.data);


  useEffect(() => {
    if (!providerId && availableModels.length) setProviderId(modelSettings.data?.bindings?.teacher?.providerId || modelSettings.data?.default || availableModels[0].id);
  }, [availableModels, modelSettings.data?.bindings, modelSettings.data?.default, providerId]);

  const flushDeltas = useCallback(() => {
    if (deltaTimer.current !== null) {
      window.clearTimeout(deltaTimer.current);
      deltaTimer.current = null;
    }
    const pending = pendingDeltas.current;
    if (!pending.text && !pending.reasoning) return;
    pendingDeltas.current = { text: '', reasoning: '' };
    setLive((current) => current ? {
      ...current,
      text: current.text + pending.text,
      reasoning: current.reasoning + pending.reasoning,
    } : current);
  }, []);

  const handleTeacherFrame = useCallback((frame: TeacherStreamFrame) => {
    if (frame.type === 'text-delta' || frame.type === 'reasoning-summary-delta') {
      const delta = String(frame.data.delta ?? '');
      if (frame.type === 'text-delta') pendingDeltas.current.text += delta;
      else pendingDeltas.current.reasoning += delta;
      if (deltaTimer.current === null) {
        deltaTimer.current = window.setTimeout(flushDeltas, STREAM_FLUSH_MS);
      }
      return;
    }
    // Preserve provider ordering when a tool or completion frame follows text
    // that is still waiting in the small render buffer.
    flushDeltas();
    handleFrame(frame, setLive);
  }, [flushDeltas]);

  useEffect(() => {
    if (!followStream.current || scrollTimer.current !== null) return;
    scrollTimer.current = window.setTimeout(() => {
      scrollTimer.current = null;
      if (scrollFrame.current !== null) return;
      scrollFrame.current = requestAnimationFrame(() => {
        scrollFrame.current = null;
        const transcript = transcriptRef.current;
        if (transcript && followStream.current) transcript.scrollTop = transcript.scrollHeight;
      });
    }, STREAM_SCROLL_MS);
  }, [conversation.data?.messages.length, live?.learner.id, live?.text, live?.teacherId, live?.notice]);

  // Initial history and deferred Markdown/Mermaid layout do not finish in the
  // same frame. Keep the viewport pinned while the content settles; once the
  // learner scrolls upward, onScroll flips followStream and the observer stops.
  useEffect(() => {
    if (!conversationReady) return;
    followStream.current = true;
    const scrollToBottom = () => {
      const transcript = transcriptRef.current;
      if (transcript && followStream.current) transcript.scrollTop = transcript.scrollHeight;
    };
    scrollToBottom();
    const frame = requestAnimationFrame(scrollToBottom);
    const content = transcriptContentRef.current;
    const observer = content && typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(scrollToBottom)
      : null;
    if (content && observer) observer.observe(content);
    return () => {
      cancelAnimationFrame(frame);
      observer?.disconnect();
    };
  }, [conversationReady, slug]);

  useEffect(() => () => {
    controller.current?.abort();
    if (scrollFrame.current !== null) cancelAnimationFrame(scrollFrame.current);
    if (scrollTimer.current !== null) window.clearTimeout(scrollTimer.current);
    if (deltaTimer.current !== null) window.clearTimeout(deltaTimer.current);
  }, []);

  // A refresh must not create a second teacher turn. Reattach to the durable
  // backend run and replay its frames into the same live response UI.
  const recoveryStarted = useRef(false);
  useEffect(() => {
    if (!conversation.data || live || recoveryStarted.current) return;
    recoveryStarted.current = true;
    const next = new AbortController();
    controller.current = next;
    const latestLearner = [...conversation.data.messages].reverse().find((message) => message.role === 'learner');
    void resumeTeacherTurn(slug, next.signal, (frame) => {
      if (frame.type === 'turn-accepted') {
        setLive((current) => current ?? {
          learner: latestLearner ?? { id: String(frame.data.learnerMessageId), role: 'learner', status: 'completed', blocks: [], createdAt: new Date().toISOString() },
          responseId: String(frame.data.responseId), text: '', reasoning: '', taskIds: [], searches: [],
        });
      }
      handleTeacherFrame(frame);
    }).then((found) => {
      if (found) void queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
    }).catch((cause) => {
      if (!next.signal.aborted) setError((cause as Error).message || '教师响应恢复失败');
    }).finally(() => {
      recoveryStarted.current = false;
      // Only the stream that still owns the controller may clear live; a
      // superseded recovery attempt must not stomp an in-flight turn.
      if (controller.current === next) {
        controller.current = null;
        if (!next.signal.aborted) setLive(null);
      }
    });
    return () => {
      next.abort();
      recoveryStarted.current = false;
      if (controller.current === next) controller.current = null;
    };
  }, [conversation.data, handleTeacherFrame, queryClient, slug]);

  // runStream owns the AbortController + post-stream refresh shared by every
  // stream origin (direct send, steer, regenerate). Returns false on failure.
  const runStream = useCallback(async (run: (signal: AbortSignal) => Promise<void>): Promise<boolean> => {
    const next = new AbortController();
    controller.current = next;
    try {
      await run(next.signal);
      flushDeltas();
      await queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
      setLive(null);
      return true;
    } catch (cause) {
      if (!next.signal.aborted) setError((cause as Error).message || '教师暂时无法响应');
      await queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
      setLive(null);
      return false;
    } finally {
      if (controller.current === next) controller.current = null;
    }
  }, [flushDeltas, queryClient, slug]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const content = input.trim();
    if (!content) return;
    setError('');
    const attachmentRefs: string[] = [];
    if (live) {
      // Sending while streaming queues durably server-side; the queue chips
      // above the composer reflect and edit that state.
      setInput('');
      try {
        await queueTeacherTurn(slug, { operationId: crypto.randomUUID(), content, attachmentRefs, providerId: providerId || undefined });
        await queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
      } catch (cause) {
        setError((cause as Error).message || '排队失败，请稍后重试');
        setInput(content);
      }
      return;
    }
    setInput('');
    followStream.current = true;
    pendingDeltas.current = { text: '', reasoning: '' };
    const operationId = crypto.randomUUID();
    const temporary: LiveResponse = {
      learner: { id: `local-${operationId}`, role: 'learner', status: 'completed', blocks: [
        { id: 'local', type: 'markdown', source: content },
        ...attachmentRefs.map((sourceId) => ({ id: `local-${sourceId}`, type: 'attachment' as const, artifactRef: sourceId })),
      ], createdAt: new Date().toISOString() },
      text: '', reasoning: '', taskIds: [], searches: [],
    };
    setLive(temporary);
    await runStream(async (signal) => {
      await streamTeacherTurn(slug, { operationId, content, attachmentRefs, providerId: providerId || undefined }, signal, handleTeacherFrame);
    });
  };

  // Steer interrupts the active response server-side; we detach from the old
  // stream and subscribe to the follow-up turn over this connection.
  const steer = async (item: QueuedTurn) => {
    if (live && !live.responseId) return;
    setError('');
    followStream.current = true;
    pendingDeltas.current = { text: '', reasoning: '' };
    controller.current?.abort();
    setLive({
      learner: { id: `local-steer-${item.queueId}`, role: 'learner', status: 'completed', blocks: [
        { id: 'local', type: 'markdown', source: item.content },
        ...(item.attachmentRefs ?? []).map((sourceId) => ({ id: `local-${sourceId}`, type: 'attachment' as const, artifactRef: sourceId })),
      ], createdAt: new Date().toISOString() },
      text: '', reasoning: '', taskIds: [], searches: [],
    });
    await runStream(async (signal) => {
      await steerQueuedTurn(slug, item.queueId, signal, handleTeacherFrame);
    });
  };

  const regenerate = useCallback(async () => {
    const responseId = latestResponseRef.current;
    if (!responseId) return;
    const trigger = [...(conversation.data?.messages ?? [])].reverse().find((message) => message.role === 'learner');
    if (!trigger) return;
    setError('');
    followStream.current = true;
    pendingDeltas.current = { text: '', reasoning: '' };
    setLive({ learner: trigger, text: '', reasoning: '', taskIds: [], searches: [] });
    await runStream(async (signal) => {
      await regenerateTeacherResponse(slug, responseId, signal, handleTeacherFrame);
    });
  }, [conversation.data?.messages, runStream, slug]);

  const copyMessage = useCallback(async (message: ConversationMessage) => {
    const markdown = (message.blocks ?? []).filter((block) => block.type === 'markdown').map((block) => block.source ?? '').join('\n\n').trim();
    try {
      await navigator.clipboard.writeText(markdown);
      setCopiedMessageId(message.id);
      window.setTimeout(() => setCopiedMessageId(''), 1600);
    } catch {
      setError('复制失败：浏览器未授权剪贴板访问');
    }
  }, []);

  const downloadMessage = useCallback((message: ConversationMessage) => {
    const markdown = messageMarkdown(message);
    const url = URL.createObjectURL(new Blob([markdown], { type: 'text/markdown;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `teacher-message-${message.id}.md`;
    anchor.click();
    URL.revokeObjectURL(url);
  }, []);

  const refreshQueue = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
  }, [queryClient, slug]);

  const submitQueueEdit = async () => {
    if (!editingQueue) return;
    const content = editingQueue.content.trim();
    if (!content) return;
    try {
      await editQueuedTurn(slug, editingQueue.queueId, { content });
      setEditingQueue(null);
      await refreshQueue();
    } catch (cause) {
      setError((cause as Error).message || '编辑排队消息失败');
    }
  };

  const discardQueueItem = async (queueId: string) => {
    try {
      await discardQueuedTurn(slug, queueId);
      await refreshQueue();
    } catch (cause) {
      setError((cause as Error).message || '移除排队消息失败');
    }
  };

  const stop = async () => {
    if (!live?.responseId) return;
    try { await stopTeacherResponse(slug, live.responseId); } finally {
      controller.current?.abort();
      pendingDeltas.current = { text: '', reasoning: '' };
      if (deltaTimer.current !== null) window.clearTimeout(deltaTimer.current);
      deltaTimer.current = null;
      setLive(null);
      await queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
    }
  };

  const linksByMessage = useMemo(() => {
    const links = new Map<string, string[]>();
    for (const link of conversation.data?.taskLinks ?? []) {
      links.set(link.messageId, [...(links.get(link.messageId) ?? []), link.taskId]);
    }
    return links;
  }, [conversation.data?.taskLinks]);
  const queueItems = useMemo(() => conversation.data?.queue ?? [], [conversation.data?.queue]);
  const lastTeacherMessage = useMemo(() => {
    const messages = conversation.data?.messages ?? [];
    for (let i = messages.length - 1; i >= 0; i--) if (messages[i].role === 'teacher') return messages[i];
    return null;
  }, [conversation.data?.messages]);
  const latestResponseRef = useRef('');
  useEffect(() => {
    latestResponseRef.current = conversation.data?.latestResponseId ?? '';
  }, [conversation.data?.latestResponseId]);

  const loadPrevious = async () => {
    const transcript = transcriptRef.current;
    if (!transcript || !conversation.hasPrevious || conversation.isLoadingPrevious) return;
    const previousHeight = transcript.scrollHeight;
    const previousTop = transcript.scrollTop;
    followStream.current = false;
    await conversation.loadPrevious();
    requestAnimationFrame(() => {
      const current = transcriptRef.current;
      if (current) current.scrollTop = previousTop + current.scrollHeight - previousHeight;
    });
  };




  return (
    <section className={s.teacher}>
      <div className={s.conversationPane}>
      <div className={s.transcript} ref={transcriptRef} role="log" aria-label="教师对话" aria-live="off" aria-busy={Boolean(live)} onScroll={() => {
        const element = transcriptRef.current;
        if (element) followStream.current = element.scrollHeight - element.scrollTop - element.clientHeight < 160;
      }}>
        <div className={s.transcriptContent} ref={transcriptContentRef}>
        {conversation.isLoading && <div className={s.loading}>教师正在准备对话…</div>}
        {conversation.hasPrevious && <button type="button" className={s.loadPrevious} onClick={() => void loadPrevious()} disabled={conversation.isLoadingPrevious}>{conversation.isLoadingPrevious ? '正在加载…' : '加载更早对话'}</button>}
        {(conversation.data?.messages ?? []).filter((message) => message.id !== live?.learner.id).map((message) => (
          <Message key={message.id} slug={slug} message={message} tasks={(linksByMessage.get(message.id) ?? []).map((id) => tasksById.get(id)).filter(Boolean) as AssistantTask[]} sourceNames={sourceNames} getReasoningOpen={getReasoningOpen} setReasoningOpen={setReasoningOpen} actions={{
            copied: copiedMessageId === message.id,
            onCopy: copyMessage,
            onDownload: downloadMessage,
            canRegenerate: !live && message.id === lastTeacherMessage?.id && Boolean(latestResponseRef.current),
            onRegenerate: regenerate,
          }} />
        ))}
        {live && (
          <>
            <Message slug={slug} message={live.learner} tasks={[]} sourceNames={sourceNames} getReasoningOpen={getReasoningOpen} setReasoningOpen={setReasoningOpen} />
            {live.reading && <div className={s.searches} role="status">
              {live.reading.operation === 'search' ? '正在查找资料' : '正在查阅'}
              {live.reading.sourceId && <> · {sourceNames.get(live.reading.sourceId) ?? '资料'}</>}
              {live.reading.displayRange && <> · {live.reading.displayRange}</>}
            </div>}
            {live.searches.length > 0 && (
              <div className={s.searches} role="status">
                {live.searches.map((search, index) => (
                  <span key={index} className={s.searchChip} data-failed={search.failed || undefined}>
                    <SearchIcon size={12} />
                    已搜索：{search.query}
                    {typeof search.count === 'number' ? ` · ${search.count} 条结果` : search.failed ? ' · 暂不可用' : ' · 检索中…'}
                  </span>
                ))}
              </div>
            )}
            <Message slug={slug} message={{ id: live.teacherId ?? 'streaming-teacher', role: 'teacher', status: 'completed', blocks: [
              ...(live.reasoning ? [{ id: 'reasoning', type: 'reasoning-summary' as const, source: live.reasoning }] : []),
              ...(live.text ? [{ id: 'text', type: 'markdown' as const, source: live.text }] : []),
            ], createdAt: new Date().toISOString() }} tasks={live.taskIds.map((id) => tasksById.get(id) ?? { id, type: 'consolidate', objective: '助教任务', status: 'queued', createdAt: '', updatedAt: '' })} sourceNames={sourceNames} getReasoningOpen={getReasoningOpen} setReasoningOpen={setReasoningOpen} notice={live.notice} streaming />
          </>
        )}
        {error && <div className={s.error}>{error}</div>}
        <div className={s.streamAnchor} aria-hidden="true" />
        </div>
      </div>
      <div className={s.composerDock}>
        {queueItems.length > 0 && (
          <div className={s.queueBar} aria-label="等待队列">
            {queueItems.map((item) => editingQueue?.queueId === item.queueId ? (
              <div key={item.queueId} className={s.queueEdit}>
                <textarea value={editingQueue.content} rows={2} onChange={(event) => setEditingQueue({ queueId: item.queueId, content: event.target.value })} onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void submitQueueEdit(); }
                  if (event.key === 'Escape') setEditingQueue(null);
                }} aria-label="编辑排队消息" autoFocus />
                <button type="button" className={s.iconButton} title="保存" onClick={() => void submitQueueEdit()}>保存</button>
                <button type="button" className={s.iconButton} title="取消" onClick={() => setEditingQueue(null)}>取消</button>
              </div>
            ) : (
              <div key={item.queueId} className={s.queueChip}>
                <span className={s.queueContent}>{item.content}</span>
                <span className={s.queueActions}>
                  <button type="button" className={s.iconButton} title="编辑这条排队消息" aria-label={`编辑排队消息 ${item.content}`} onClick={() => setEditingQueue({ queueId: item.queueId, content: item.content })}><EditIcon /></button>
                  <button type="button" className={s.iconButton} title="立即引导：打断当前回答，先回应这条" aria-label={`立即引导 ${item.content}`} onClick={() => void steer(item)} disabled={Boolean(live && !live.responseId)}><SteerIcon /></button>
                  <button type="button" className={s.iconButton} title="移出队列" aria-label={`移除排队消息 ${item.content}`} onClick={() => void discardQueueItem(item.queueId)}><TrashIcon /></button>
                </span>
              </div>
            ))}
          </div>
        )}

      <form className={s.composer} onSubmit={submit}>
        <textarea value={input} onChange={(event) => setInput(event.target.value)} onKeyDown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); event.currentTarget.form?.requestSubmit(); }
        }} placeholder={live ? '教师正在回答…发送将进入等待队列' : `和 ${title} 的教师继续讨论…`} rows={1} />
        {live ? <button type="submit" className={s.stop} disabled={!input.trim()}>排队</button> : <button type="submit" disabled={!input.trim()}>发送</button>}
        {live?.responseId && <button type="button" className={s.stop} onClick={stop}>停止</button>}
        <div className={s.composerMeta}>
          {availableModels.length > 0 && <label className={s.modelSelect}>模型
            <select value={providerId} onChange={(event) => { setProviderId(event.target.value); localStorage.setItem(`lll.teacher.provider.${slug}`, event.target.value); }}>
              {availableModels.map((provider) => <option key={provider.id} value={provider.id}>{provider.name ? `${provider.name} · ${provider.model}` : provider.model}</option>)}
            </select>
          </label>}
          {availableSources.length > 0 && <span className={s.materialStatus}>有 {availableSources.length} 份资料可查阅</span>}
          <span className={s.keyboardHint}>Enter 发送 · Shift + Enter 换行</span>
        </div>
      </form>
      </div>
      </div>

    </section>
  );
}

function handleFrame(frame: TeacherStreamFrame, setLive: React.Dispatch<React.SetStateAction<LiveResponse | null>>) {
  setLive((current) => {
    if (!current) return current;
    if (frame.type === 'turn-accepted') return { ...current, responseId: String(frame.data.responseId), learner: { ...current.learner, id: String(frame.data.learnerMessageId) } };
    if (frame.type === 'message-started') return { ...current, teacherId: String(frame.data.teacherMessageId) };
    if (frame.type === 'message-reset') return { ...current, text: '', reasoning: '', notice: '', searches: [], reading: undefined };
    if (frame.type === 'message-completed' || frame.type === 'message-failed') return { ...current, reading: undefined };
    if (frame.type === 'source-reading') {
      const operationId = String(frame.data.operationId ?? '');
      if (frame.data.phase === 'started') return { ...current, reading: { operationId, operation: String(frame.data.operation ?? ''), sourceId: typeof frame.data.sourceId === 'string' ? frame.data.sourceId : undefined, displayRange: typeof frame.data.displayRange === 'string' ? frame.data.displayRange : undefined } };
      return current.reading?.operationId === operationId ? { ...current, reading: undefined } : current;
    }
    if (frame.type === 'text-delta') return { ...current, text: current.text + String(frame.data.delta ?? '') };
    if (frame.type === 'reasoning-summary-delta') return { ...current, reasoning: current.reasoning + String(frame.data.delta ?? '') };
    if (frame.type === 'task-accepted') return { ...current, taskIds: [...current.taskIds, String(frame.data.taskId)] };
    if (frame.type === 'tool-rejected') return { ...current, notice: rejectionText(String(frame.data.code)) };
    if (frame.type === 'search-started') return { ...current, searches: [...current.searches, { query: String(frame.data.query ?? '') }] };
    if (frame.type === 'search-completed' || frame.type === 'search-failed') {
      return { ...current, searches: current.searches.map((search, index) => index === current.searches.length - 1 && !search.count && !search.failed
        ? (frame.type === 'search-completed' ? { ...search, count: Number(frame.data.count ?? 0) } : { ...search, failed: true })
        : search) };
    }
    return current;
  });
}

function rejectionText(code: string) {
  if (code === 'same_type_active') return '同类型助教任务已经在进行，本次没有重复创建。';
  if (code === 'delegation_not_approved') return '助教任务没有创建：需要先说明方案，并由你在后续消息中明确同意。';
  return '助教任务没有创建，你可以继续和教师确认方案。';
}

// MessageActions stays primitive-valued (or stable callbacks) so the memo
// comparator can keep history messages frozen while live text streams.
interface MessageActions {
  copied: boolean;
  onCopy: (message: ConversationMessage) => void;
  onDownload: (message: ConversationMessage) => void;
  canRegenerate: boolean;
  onRegenerate: () => void;
}

const Message = memo(function Message({ slug, message, sourceNames, getReasoningOpen, setReasoningOpen, notice, streaming = false, actions }: { slug: string; message: ConversationMessage; tasks: AssistantTask[]; sourceNames: Map<string, string>; getReasoningOpen: (messageID: string) => boolean; setReasoningOpen: (messageID: string, open: boolean) => void; notice?: string; streaming?: boolean; actions?: MessageActions }) {
  const teacher = message.role === 'teacher';
  const [selectedEvidence, setSelectedEvidence] = useState('');
  const citations = message.sourceCitations ?? [];
  const citationSource = (source: string) => source.replace(/\[\[source:([^\]]+)\]\]/g, (_marker, id: string) => {
    const index = citations.findIndex(item => item.evidenceId === id);
    return index < 0 ? '' : `[资料${index + 1}](#source-${id})`;
  });
  return (
    <article className={teacher ? s.teacherMessage : s.learnerMessage}>
      {teacher && <div className={s.speaker}>教师</div>}
      <div className={s.messageBody} onClick={event => {
        const target = event.target;
        if (!(target instanceof Element)) return;
        const href = target.closest('a')?.getAttribute('href');
        if (!href?.startsWith('#source-')) return;
        const evidenceId = href.slice('#source-'.length);
        if (!citations.some(item => item.evidenceId === evidenceId)) return;
        event.preventDefault();
        setSelectedEvidence(evidenceId);
      }}>
        {streaming && !message.blocks?.some((block) => block.type === 'markdown' && block.source) && <span className={s.thinking} role="status">教师正在生成…</span>}
        {(message.blocks ?? []).map((block) => block.type === 'reasoning-summary'
          ? <ReasoningBlock key={block.id} messageID={message.id} source={block.source ?? ''} streaming={streaming} initialOpen={getReasoningOpen(message.id)} onOpenChange={setReasoningOpen} />
          : block.type === 'markdown' ? <MarkdownView key={block.id} source={citationSource(teacher ? withoutLearningProgress(block.source ?? '') : block.source ?? '')} className={s.markdown} streaming={streaming} />
            : block.type === 'attachment' ? <span key={block.id} className={s.attachment}>资料 · {sourceNames.get(block.artifactRef ?? '') ?? '已选择资料'}</span> : null)}
        {teacher && !streaming && <SourceCitations slug={slug} citations={citations} sourceNames={sourceNames} selected={selectedEvidence} onSelect={setSelectedEvidence} />}
        {teacher && !streaming && message.status === 'completed' && <LearningProgress slug={slug} value={readLearningProgress(messageMarkdown(message))} />}
        {notice && <div className={s.notice}>{notice}</div>}
        {teacher && !streaming && message.status === 'failed' && <div className={s.notice} role="alert">这条回复未完成，请重新生成或继续提问。</div>}        {actions && !streaming && (
          <div className={s.messageActions}>
            <button type="button" className={s.iconButton} title={actions.copied ? '已复制' : '复制原文'} aria-label={actions.copied ? '已复制' : '复制原文'} onClick={() => void actions.onCopy(message)}><CopyIcon />{actions.copied && <span className={s.copiedMark}>已复制</span>}</button>
            <button type="button" className={s.iconButton} title="下载本条文本" aria-label="下载本条文本" onClick={() => actions.onDownload(message)}><DownloadIcon /></button>
            {actions.canRegenerate && <button type="button" className={s.iconButton} title="重新生成这条回复" aria-label="重新生成这条回复" onClick={actions.onRegenerate}><RegenerateIcon /></button>}
          </div>
        )}
      </div>
    </article>
  );
}, (previous, next) => previous.message === next.message
  && previous.streaming === next.streaming
  && previous.notice === next.notice
  && previous.sourceNames === next.sourceNames
  && sameTasks(previous.tasks, next.tasks)
  && previous.actions?.copied === next.actions?.copied
  && previous.actions?.canRegenerate === next.actions?.canRegenerate
  && previous.actions?.onCopy === next.actions?.onCopy
  && previous.actions?.onDownload === next.actions?.onDownload
  && previous.actions?.onRegenerate === next.actions?.onRegenerate);

function messageMarkdown(message: ConversationMessage) {
  return (message.blocks ?? [])
    .filter((block) => block.type === 'markdown')
    .map((block) => block.source ?? '')
    .join('\n\n')
    .trim();
}

function ReasoningBlock({ messageID, source, streaming, initialOpen, onOpenChange }: { messageID: string; source: string; streaming: boolean; initialOpen: boolean; onOpenChange: (messageID: string, open: boolean) => void }) {
  const [open, setOpen] = useState(() => initialOpen);
  const detailsRef = useRef<HTMLDetailsElement>(null);
  const initialOpenRef = useRef(initialOpen);
  useLayoutEffect(() => {
    if (initialOpenRef.current && detailsRef.current) detailsRef.current.open = true;
  }, []);
  return (
    // Let the browser own the disclosure state. A controlled <details> can be
    // written back while Markdown is replaced during streaming, which makes
    // an expanded reasoning block appear stuck or unexpectedly re-opened.
    <details ref={detailsRef} className={s.reasoning} onToggle={(event) => {
      const nextOpen = event.currentTarget.open;
      setOpen(nextOpen);
      onOpenChange(messageID, nextOpen);
    }}>
      <summary>思考过程</summary>
      {open && <MarkdownView source={source} streaming={streaming} />}
    </details>
  );
}

function sameTasks(left: AssistantTask[], right: AssistantTask[]) {
  return left.length === right.length && left.every((task, index) => {
    const other = right[index];
    return task.id === other?.id && task.status === other.status && task.updatedAt === other.updatedAt;
  });
}
