import { ChangeEvent, FormEvent, memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { MarkdownView } from '@/shared/primitive/MarkdownView';
import {
  stopTeacherResponse,
  streamTeacherTurn,
  resumeTeacherTurn,
  learningWorkspaceKeys,
  useAssistantTasks,
  useAssets,
  useConversation,
  useSources,
  useUploadSource,
  type AssistantTask,
  type ConversationMessage,
  type TeacherStreamFrame,
} from '@/features/learning/api/learningWorkspace';
import { useAskAiSettings } from '@/features/settings';

import s from './TeacherView.module.css';

interface LiveResponse {
  learner: ConversationMessage;
  teacherId?: string;
  responseId?: string;
  text: string;
  reasoning: string;
  taskIds: string[];
  notice?: string;
}

const STREAM_FLUSH_MS = 64;
const STREAM_SCROLL_MS = 96;
const terminalTaskStatuses = new Set<AssistantTask['status']>(['succeeded', 'partial', 'failed', 'cancelled']);

export function TeacherView({ slug, title }: { slug: string; title: string }) {
  const queryClient = useQueryClient();
  const conversation = useConversation(slug);
  const tasksQuery = useAssistantTasks(slug);
  const sourcesQuery = useSources(slug);
  const assetsQuery = useAssets(slug);
  const modelSettings = useAskAiSettings();
  const upload = useUploadSource(slug);
  const [input, setInput] = useState('');
  const [selectedSourceIds, setSelectedSourceIds] = useState<string[]>([]);
  const [providerId, setProviderId] = useState(() => localStorage.getItem(`lll.teacher.provider.${slug}`) ?? '');
  const [pendingUpload, setPendingUpload] = useState<File | null>(null);
  const [cloudAccepted, setCloudAccepted] = useState(false);
  const [live, setLive] = useState<LiveResponse | null>(null);
  const [error, setError] = useState('');
  const [taskNotice, setTaskNotice] = useState<{ task: AssistantTask; text: string } | null>(null);
  const controller = useRef<AbortController | null>(null);
  const transcriptRef = useRef<HTMLDivElement>(null);
  const transcriptContentRef = useRef<HTMLDivElement>(null);
  const followStream = useRef(true);
  const scrollFrame = useRef<number | null>(null);
  const scrollTimer = useRef<number | null>(null);
  const pendingDeltas = useRef({ text: '', reasoning: '' });
  const deltaTimer = useRef<number | null>(null);
  const previousTaskStates = useRef<Map<string, AssistantTask['status']> | null>(null);
  const tasksById = useMemo(() => new Map((tasksQuery.data ?? []).map((task) => [task.id, task])), [tasksQuery.data]);
  const sourceNames = useMemo(() => new Map((sourcesQuery.data ?? []).map((source) => [source.sourceId, source.displayName])), [sourcesQuery.data]);
  const availableSources = useMemo(() => (sourcesQuery.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted'), [sourcesQuery.data]);
  const availableModels = useMemo(() => (modelSettings.data?.providers ?? []).filter((provider) => provider.id && provider.model && provider.baseURL), [modelSettings.data?.providers]);
  const activeTasks = useMemo(() => (tasksQuery.data ?? []).filter((task) => task.status === 'queued' || task.status === 'running'), [tasksQuery.data]);
  const selectedModel = availableModels.find((provider) => provider.id === providerId);
  const conversationReady = Boolean(conversation.data);

  useEffect(() => {
    const current = new Map((tasksQuery.data ?? []).map((task) => [task.id, task.status]));
    const seen = readSeenTasks(slug);
    const completed = (tasksQuery.data ?? []).filter((task) => terminalTaskStatuses.has(task.status) && !seen.has(task.id));
    const transitioned = previousTaskStates.current
      ? completed.filter((task) => {
        const before = previousTaskStates.current?.get(task.id);
        return before === 'queued' || before === 'running';
      })
      : completed;
    const task = [...transitioned].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))[0];
    if (task) {
      setTaskNotice({ task, text: taskCompletionText(task) });
    }
    previousTaskStates.current = current;
  }, [slug, tasksQuery.data]);

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
  }, [conversation.data?.messages.length, live?.learner.id, live?.text, live?.teacherId, live?.notice, taskNotice]);

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
          responseId: String(frame.data.responseId), text: '', reasoning: '', taskIds: [],
        });
      }
      handleTeacherFrame(frame);
    }).then((found) => {
      if (found) void queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
    }).catch((cause) => {
      if (!next.signal.aborted) setError((cause as Error).message || '教师响应恢复失败');
    }).finally(() => {
      recoveryStarted.current = false;
      if (controller.current === next) controller.current = null;
      if (!next.signal.aborted) setLive(null);
    });
    return () => {
      next.abort();
      recoveryStarted.current = false;
      if (controller.current === next) controller.current = null;
    };
  }, [conversation.data, handleTeacherFrame, queryClient, slug]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const content = input.trim();
    if (!content || live) return;
    setInput('');
    setSelectedSourceIds([]);
    setError('');
    followStream.current = true;
    pendingDeltas.current = { text: '', reasoning: '' };
    const operationId = crypto.randomUUID();
    const temporary: LiveResponse = {
      learner: { id: `local-${operationId}`, role: 'learner', status: 'completed', blocks: [
        { id: 'local', type: 'markdown', source: content },
        ...selectedSourceIds.map((sourceId) => ({ id: `local-${sourceId}`, type: 'attachment' as const, artifactRef: sourceId })),
      ], createdAt: new Date().toISOString() },
      text: '', reasoning: '', taskIds: [],
    };
    setLive(temporary);
    const next = new AbortController();
    controller.current = next;
    try {
      await streamTeacherTurn(slug, { operationId, content, attachmentRefs: selectedSourceIds, providerId: providerId || undefined }, next.signal, handleTeacherFrame);
      flushDeltas();
      await queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
      setLive(null);
    } catch (cause) {
      if (!next.signal.aborted) setError((cause as Error).message || '教师暂时无法响应');
      if (!next.signal.aborted) setSelectedSourceIds(selectedSourceIds);
      await queryClient.resetQueries({ queryKey: learningWorkspaceKeys.conversation(slug) });
      setLive(null);
    } finally {
      controller.current = null;
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

  const chooseUpload = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    event.target.value = '';
    setCloudAccepted(false);
    setPendingUpload(file);
  };

  const finishUpload = async (parseApproved: boolean) => {
    if (!pendingUpload) return;
    const result = await upload.mutateAsync({ file: pendingUpload, parseApproved, cloudDisclosureAccepted: parseApproved && cloudAccepted });
    if (result.source?.sourceId) setSelectedSourceIds((current) => [...new Set([...current, result.source.sourceId])]);
    setPendingUpload(null);
    setCloudAccepted(false);
  };

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

  const dismissTaskNotice = () => {
    if (!taskNotice) return;
    const key = `lll.teacher.seenTasks.${slug}`;
    const seen = readSeenTasks(slug);
    seen.add(taskNotice.task.id);
    localStorage.setItem(key, JSON.stringify([...seen].slice(-200)));
    setTaskNotice(null);
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
          <Message key={message.id} slug={slug} message={message} tasks={(linksByMessage.get(message.id) ?? []).map((id) => tasksById.get(id)).filter(Boolean) as AssistantTask[]} sourceNames={sourceNames} />
        ))}
        {live && (
          <>
            <Message slug={slug} message={live.learner} tasks={[]} sourceNames={sourceNames} />
            <Message slug={slug} message={{ id: live.teacherId ?? 'streaming-teacher', role: 'teacher', status: 'completed', blocks: [
              ...(live.reasoning ? [{ id: 'reasoning', type: 'reasoning-summary' as const, source: live.reasoning }] : []),
              ...(live.text ? [{ id: 'text', type: 'markdown' as const, source: live.text }] : []),
            ], createdAt: new Date().toISOString() }} tasks={live.taskIds.map((id) => tasksById.get(id) ?? { id, type: 'consolidate', objective: '助教任务', status: 'queued', createdAt: '', updatedAt: '' })} sourceNames={sourceNames} notice={live.notice} streaming />
          </>
        )}
        {taskNotice && <div className={s.taskNotice} role="status"><div><strong>{taskNotice.text}</strong><span>{taskNotice.task.objective}</span></div><div>{(taskNotice.task.type === 'source-processing' || !!taskNotice.task.result?.deliverables?.length) && <a href={`/project/${encodeURIComponent(slug)}/sources`}>到资料页查看</a>}<button type="button" onClick={dismissTaskNotice}>知道了</button></div></div>}
        {error && <div className={s.error}>{error}</div>}
        <div className={s.streamAnchor} aria-hidden="true" />
        </div>
      </div>
      <form className={s.composer} onSubmit={submit}>
        <label className={s.uploadButton} title="上传资料">
          <span aria-hidden="true">＋</span><span className={s.srOnly}>上传资料</span>
          <input type="file" onChange={chooseUpload} />
        </label>
        {!!availableSources.length && <details className={s.sourcePicker}>
          <summary>资料{selectedSourceIds.length ? ` ${selectedSourceIds.length}` : ''}</summary>
          <div className={s.sourceMenu}>
            <strong>本条消息引用</strong>
            {availableSources.map((source) => <label key={source.sourceId}>
              <input type="checkbox" checked={selectedSourceIds.includes(source.sourceId)} onChange={(event) => setSelectedSourceIds((current) => event.target.checked ? [...current, source.sourceId] : current.filter((id) => id !== source.sourceId))} />
              <span>{source.displayName}</span><small>{source.status}</small>
            </label>)}
          </div>
        </details>}
        <textarea value={input} onChange={(event) => setInput(event.target.value)} onKeyDown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); event.currentTarget.form?.requestSubmit(); }
        }} placeholder={`和 ${title} 的教师继续讨论…`} rows={1} disabled={Boolean(live)} />
        {live ? <button type="button" className={s.stop} onClick={stop} disabled={!live.responseId}>停止</button> : <button type="submit" disabled={!input.trim()}>发送</button>}
        <div className={s.composerMeta}>
          {availableModels.length > 0 && <label className={s.modelSelect}>模型
            <select value={providerId} onChange={(event) => { setProviderId(event.target.value); localStorage.setItem(`lll.teacher.provider.${slug}`, event.target.value); }}>
              {availableModels.map((provider) => <option key={provider.id} value={provider.id}>{provider.name ? `${provider.name} · ${provider.model}` : provider.model}</option>)}
            </select>
          </label>}
          <span>Enter 发送 · Shift + Enter 换行</span>
        </div>
        {pendingUpload && <div className={s.uploadPanel} role="dialog" aria-label="上传资料">
          <strong>{pendingUpload.name}</strong>
          <p>可以只保存原件，或交给助教异步解析；解析不会阻塞当前对话。</p>
          <label><input type="checkbox" checked={cloudAccepted} onChange={(event) => setCloudAccepted(event.target.checked)} /> 我知道解析时内容可能发送到当前 CLI Agent 的模型服务</label>
          {upload.isError && <span className={s.taskFailure}>上传失败，请稍后重试。</span>}
          <div><button type="button" className={s.secondary} onClick={() => { setPendingUpload(null); setCloudAccepted(false); }}>取消</button><button type="button" className={s.secondary} disabled={upload.isPending} onClick={() => void finishUpload(false)}>仅保存</button><button type="button" disabled={!cloudAccepted || upload.isPending} onClick={() => void finishUpload(true)}>保存并解析</button></div>
        </div>}
      </form>
      </div>
      <aside className={s.dataRail} aria-label="学习单元数据">
        <header><span>UNIT SIGNALS</span><h2>学习数据</h2></header>
        <dl>
          <div><dt>对话</dt><dd>{conversation.data?.totalMessages ?? conversation.data?.messages.length ?? 0}<small> 条消息</small></dd></div>
          <div><dt>资料</dt><dd>{availableSources.length}<small> 份可引用</small></dd></div>
          <div><dt>助教</dt><dd>{activeTasks.length ? activeTasks.length : (tasksQuery.data?.length ?? 0)}<small>{activeTasks.length ? ' 项进行中' : ' 项记录'}</small></dd></div>
        </dl>
        <section>
          <span>当前模型</span>
          <strong>{selectedModel?.name || selectedModel?.model || '使用默认配置'}</strong>
          {selectedModel?.name && <small>{selectedModel.model}</small>}
        </section>
        <section>
          <span>教学资产</span>
          <ul>{(assetsQuery.data ?? []).map((assetMeta) => <li key={assetMeta.key}><i /><span>{assetMeta.title}</span><small>v{assetMeta.editRevision}</small></li>)}</ul>
        </section>
        <p>助教在后台异步工作；任务被接受后会显示在对应教师消息下方。</p>
      </aside>
    </section>
  );
}

function readSeenTasks(slug: string) {
  try {
    const value = JSON.parse(localStorage.getItem(`lll.teacher.seenTasks.${slug}`) || '[]');
    return new Set<string>(Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []);
  } catch {
    return new Set<string>();
  }
}

function taskCompletionText(task: AssistantTask) {
  if (task.status === 'cancelled') return '助教任务已在终端取消。';
  if (task.status === 'failed') return '助教任务执行失败，请查看任务说明。';
  if (task.status === 'partial') return '助教已部分完成，可查看已提交的成果。';
  if (task.type === 'source-processing') return '助教已完成资料解析，可在资料页查看。';
  if (task.result?.deliverables?.length) return '助教已完成，成果已同步到资产与资料。';
  if (Object.values(task.result?.assetUpdates ?? {}).includes('updated')) return '助教已完成，教学资产已经更新。';
  return '助教任务已完成。';
}

function handleFrame(frame: TeacherStreamFrame, setLive: React.Dispatch<React.SetStateAction<LiveResponse | null>>) {
  setLive((current) => {
    if (!current) return current;
    if (frame.type === 'turn-accepted') return { ...current, responseId: String(frame.data.responseId), learner: { ...current.learner, id: String(frame.data.learnerMessageId) } };
    if (frame.type === 'message-started') return { ...current, teacherId: String(frame.data.teacherMessageId) };
    if (frame.type === 'text-delta') return { ...current, text: current.text + String(frame.data.delta ?? '') };
    if (frame.type === 'reasoning-summary-delta') return { ...current, reasoning: current.reasoning + String(frame.data.delta ?? '') };
    if (frame.type === 'task-accepted') return { ...current, taskIds: [...current.taskIds, String(frame.data.taskId)] };
    if (frame.type === 'tool-rejected') return { ...current, notice: rejectionText(String(frame.data.code)) };
    return current;
  });
}

function rejectionText(code: string) {
  if (code === 'same_type_active') return '同类型助教任务已经在进行，本次没有重复创建。';
  if (code === 'delegation_not_approved') return '助教任务没有创建：需要先说明方案，并由你在后续消息中明确同意。';
  return '助教任务没有创建，你可以继续和教师确认方案。';
}

const Message = memo(function Message({ slug, message, tasks, sourceNames, notice, streaming = false }: { slug: string; message: ConversationMessage; tasks: AssistantTask[]; sourceNames: Map<string, string>; notice?: string; streaming?: boolean }) {
  const teacher = message.role === 'teacher';
  return (
    <article className={teacher ? s.teacherMessage : s.learnerMessage}>
      {teacher && <div className={s.speaker}>教师</div>}
      <div className={s.messageBody}>
        {streaming && !message.blocks?.some((block) => block.type === 'markdown' && block.source) && <span className={s.thinking} role="status">教师正在生成…</span>}
        {(message.blocks ?? []).map((block) => block.type === 'reasoning-summary'
          ? <ReasoningBlock key={block.id} source={block.source ?? ''} streaming={streaming} />
          : block.type === 'markdown' ? <MarkdownView key={block.id} source={block.source ?? ''} className={s.markdown} streaming={streaming} />
            : block.type === 'attachment' ? <span key={block.id} className={s.attachment}>资料 · {sourceNames.get(block.artifactRef ?? '') ?? '已选择资料'}</span> : null)}
        {notice && <div className={s.notice}>{notice}</div>}
        {tasks.map((task) => <TaskCard key={task.id} slug={slug} task={task} />)}
      </div>
    </article>
  );
}, (previous, next) => previous.message === next.message
  && previous.streaming === next.streaming
  && previous.notice === next.notice
  && previous.sourceNames === next.sourceNames
  && sameTasks(previous.tasks, next.tasks));

function ReasoningBlock({ source, streaming }: { source: string; streaming: boolean }) {
  const [open, setOpen] = useState(false);
  return (
    <details className={s.reasoning} open={open} onToggle={(event) => setOpen(event.currentTarget.open)}>
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

function TaskCard({ slug, task }: { slug: string; task: AssistantTask }) {
  const label: Record<AssistantTask['status'], string> = { queued: '等待助教', running: '助教执行中', succeeded: '助教已完成', partial: '部分完成', failed: '执行失败', cancelled: '已在终端取消' };
  return (
    <div className={s.taskCard} data-status={task.status}>
      <div><span className={s.taskDot} /><strong>{label[task.status] ?? '助教任务'}</strong><small>{task.phase ? ` · ${task.phase}` : ''}</small></div>
      <p>{task.objective || '助教正在处理已确认的工作。'}</p>
      {task.result?.summary && <p className={s.taskResult}>{task.result.summary}</p>}
      {!!task.result?.deliverables?.length && <div className={s.deliverables}>{task.result.deliverables.map((artifactId) => <a key={artifactId} href={`/api/projects/${encodeURIComponent(slug)}/generated/${encodeURIComponent(artifactId)}/open`} target="_blank" rel="noreferrer">打开成果</a>)}<a href={`/project/${encodeURIComponent(slug)}/sources`}>在资料页查看</a></div>}
      {task.failure && <p className={s.taskFailure}>{task.failure.message}{task.failure.suggestion ? ` ${task.failure.suggestion}` : ''}</p>}
    </div>
  );
}
