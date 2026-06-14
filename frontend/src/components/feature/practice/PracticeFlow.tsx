// PracticeFlow — single-question exam UI with a useReducer state machine.
//
// Phases: idle → generating → answering → confirming → submitting → submitted
//         (error reachable from generating on agent-fail / timeout / corrupt)
//
// Generation: invoke the Practice Agent with an intent naming the count
// (charter takes the count from intent), then poll tasks.json until it
// appears (refetchInterval driven by phase). Drafts persist in localStorage
// keyed by task fingerprint so a regenerate never bleeds old answers.

import { useEffect, useReducer } from 'react';

import { useInvokeAgent } from '@/api/agents';
import { usePracticeTasks, useSubmitPractice, usePracticeEvaluation, type Submission } from '@/api/practice';
import { PERMISSION_MODES, PRACTICE_GEN_POLL_MS, PRACTICE_GEN_TIMEOUT_MS } from '@/lib/constants';
import {
  loadDraft, saveDraft, clearDraft, draftMatches, getLastAttempt, setLastAttempt,
  type DraftEntry,
} from '@/lib/practiceDraft';
import { Button } from '@/components/primitive/Button';
import { EmptyState } from '@/components/primitive/EmptyState';
import { GenerationEntry } from './GenerationEntry';
import { PracticeProgress } from './PracticeProgress';
import { QuestionCard } from './QuestionCard';
import { ConfirmSubmitModal } from './ConfirmSubmitModal';

import s from './PracticeFlow.module.css';

type Phase = 'idle' | 'generating' | 'answering' | 'confirming' | 'submitting' | 'submitted' | 'error';
type ErrorKind = 'agent' | 'timeout' | 'corrupt' | null;

interface State {
  phase: Phase;
  currentIndex: number;
  taskIds: string[];
  generatedAt: string | null;
  drafts: Record<string, DraftEntry>;
  attempt: number;
  errorKind: ErrorKind;
}

const initialState: State = {
  phase: 'idle',
  currentIndex: 0,
  taskIds: [],
  generatedAt: null,
  drafts: {},
  attempt: 0,
  errorKind: null,
};

type Action =
  | { type: 'GENERATE' }
  | { type: 'TASKS_READY'; taskIds: string[]; generatedAt: string; drafts: Record<string, DraftEntry> }
  | { type: 'INIT_SUBMITTED'; attempt: number }
  | { type: 'TIMEOUT' }
  | { type: 'AGENT_FAILED' }
  | { type: 'CORRUPT' }
  | { type: 'SET_INDEX'; index: number }
  | { type: 'ANSWER'; taskId: string; answer: string }
  | { type: 'ASSESS'; taskId: string; selfAssess: number }
  | { type: 'OPEN_CONFIRM' }
  | { type: 'CLOSE_CONFIRM' }
  | { type: 'SUBMIT' }
  | { type: 'SUBMITTED'; attempt: number }
  | { type: 'SUBMIT_FAILED' }
  | { type: 'REGENERATE' }
  | { type: 'BACK_TO_IDLE' };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'GENERATE':
      return { ...state, phase: 'generating', errorKind: null };
    case 'TASKS_READY':
      return {
        ...state,
        phase: 'answering',
        taskIds: action.taskIds,
        generatedAt: action.generatedAt,
        drafts: action.drafts,
        currentIndex: 0,
      };
    case 'INIT_SUBMITTED':
      return { ...state, phase: 'submitted', attempt: action.attempt };
    case 'TIMEOUT':
      return { ...state, phase: 'error', errorKind: 'timeout' };
    case 'AGENT_FAILED':
      return { ...state, phase: 'error', errorKind: 'agent' };
    case 'CORRUPT':
      return { ...state, phase: 'error', errorKind: 'corrupt' };
    case 'SET_INDEX':
      return { ...state, currentIndex: action.index };
    case 'ANSWER': {
      const prev = state.drafts[action.taskId] ?? { answer: '', selfAssess: 0 };
      return { ...state, drafts: { ...state.drafts, [action.taskId]: { ...prev, answer: action.answer } } };
    }
    case 'ASSESS': {
      const prev = state.drafts[action.taskId] ?? { answer: '', selfAssess: 0 };
      return { ...state, drafts: { ...state.drafts, [action.taskId]: { ...prev, selfAssess: action.selfAssess } } };
    }
    case 'OPEN_CONFIRM':
      return { ...state, phase: 'confirming' };
    case 'CLOSE_CONFIRM':
      return { ...state, phase: 'answering' };
    case 'SUBMIT':
      return { ...state, phase: 'submitting' };
    case 'SUBMITTED':
      return { ...state, phase: 'submitted', attempt: action.attempt };
    case 'SUBMIT_FAILED':
      return { ...state, phase: 'answering' };
    case 'REGENERATE':
      return { ...state, phase: 'generating', currentIndex: 0, drafts: {}, errorKind: null };
    case 'BACK_TO_IDLE':
      return { ...state, phase: 'idle', errorKind: null };
    default:
      return state;
  }
}

interface PracticeFlowProps {
  projectSlug: string;
}

export function PracticeFlow({ projectSlug }: PracticeFlowProps) {
  const invoke = useInvokeAgent();
  const submitPractice = useSubmitPractice();
  const [state, dispatch] = useReducer(reducer, initialState);

  const tasksData = usePracticeTasks(projectSlug, {
    refetchInterval: state.phase === 'generating' ? PRACTICE_GEN_POLL_MS : false,
  });
  const tasks = tasksData.data?.tasks ?? [];
  const generatedAt = tasksData.data?.generatedAt;
  const hasTasks = tasksData.data?.generated === true && Array.isArray(tasks) && tasks.length > 0;

  // submitted → poll evaluation for the attempt (agent writes it async)
  const evalQuery = usePracticeEvaluation(
    state.phase === 'submitted' ? projectSlug : undefined,
    state.attempt,
  );

  // Hydrate + poll-hit: when tasks appear (initial or after generate), enter answering.
  useEffect(() => {
    if (tasksData.isLoading) return;
    if (state.phase !== 'idle' && state.phase !== 'generating') return;
    if (hasTasks && generatedAt) {
      const stored = loadDraft(projectSlug);
      const drafts = draftMatches(stored, tasks.map((t) => t.id), generatedAt)
        ? stored!.drafts
        : {};
      dispatch({ type: 'TASKS_READY', taskIds: tasks.map((t) => t.id), generatedAt, drafts });
    } else if (state.phase === 'idle') {
      const lastAttempt = getLastAttempt(projectSlug);
      if (lastAttempt) dispatch({ type: 'INIT_SUBMITTED', attempt: lastAttempt });
    }
  }, [hasTasks, generatedAt, tasksData.isLoading, state.phase, projectSlug, tasks]);

  // corrupt guard: generated=true but tasks not a real array
  useEffect(() => {
    if (tasksData.data?.generated === true && !Array.isArray(tasksData.data.tasks)) {
      dispatch({ type: 'CORRUPT' });
    }
  }, [tasksData.data]);

  // poll timeout
  useEffect(() => {
    if (state.phase !== 'generating') return;
    const timer = setTimeout(() => dispatch({ type: 'TIMEOUT' }), PRACTICE_GEN_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [state.phase]);

  // draft persistence: save on every draft/index change while answering
  useEffect(() => {
    if (state.phase !== 'answering' && state.phase !== 'confirming') return;
    if (!state.generatedAt || state.taskIds.length === 0) return;
    saveDraft(projectSlug, {
      generatedAt: state.generatedAt,
      taskIds: state.taskIds,
      drafts: state.drafts,
    });
  }, [state.drafts, state.currentIndex, state.phase, state.generatedAt, state.taskIds, projectSlug]);

  // ---- handlers ----
  const handleGenerate = async (count: number) => {
    try {
      await invoke.mutateAsync({
        agentId: 'practice',
        payload: {
          projectId: projectSlug,
          zone: 'Practice',
          intent: `生成 ${count} 道迁移练习题，覆盖讲解中的核心概念。每题给一个新场景+约束+自检问题，不含答案或解析。`,
          permissionMode: PERMISSION_MODES.acceptEdits,
        },
      });
      dispatch({ type: 'GENERATE' });
    } catch {
      dispatch({ type: 'AGENT_FAILED' });
    }
  };

  const handleSubmit = async () => {
    dispatch({ type: 'SUBMIT' });
    const submissions: Submission[] = state.taskIds.map((id) => {
      const d = state.drafts[id] ?? { answer: '', selfAssess: 0 };
      return { taskId: id, answer: d.answer, selfAssess: d.selfAssess };
    });
    try {
      const res = await submitPractice.mutateAsync({ projectSlug, submissions });
      clearDraft(projectSlug);
      setLastAttempt(projectSlug, res.attempt);
      dispatch({ type: 'SUBMITTED', attempt: res.attempt });
    } catch {
      dispatch({ type: 'SUBMIT_FAILED' });
    }
  };

  // ---- render by phase ----
  if (state.phase === 'idle') {
    return <GenerationEntry onGenerate={handleGenerate} isInvoking={invoke.isPending} />;
  }

  if (state.phase === 'generating') {
    return (
      <EmptyState
        title="正在生成练习题…"
        description={<>练习智能体已启动，题目就绪后自动显示（每 3 秒检查一次，最长等 5 分钟）。</>}
      />
    );
  }

  if (state.phase === 'error') {
    const msgs: Record<string, { title: string; desc: string }> = {
      timeout: {
        title: '生成超时',
        desc: '超过 5 分钟未收到题目。练习智能体可能仍在后台运行，或已失败。',
      },
      agent: {
        title: '调用失败',
        desc: '练习智能体未能启动。请确认后端在线、claude 可用后重试。',
      },
      corrupt: {
        title: '题目文件损坏',
        desc: 'practice/tasks.json 格式异常，无法解析。可重新生成。',
      },
    };
    const m = msgs[state.errorKind ?? 'agent'];
    return (
      <EmptyState
        title={m.title}
        description={m.desc}
        action={
          <Button variant="primary" onClick={() => dispatch({ type: 'BACK_TO_IDLE' })}>
            返回题量选择
          </Button>
        }
      />
    );
  }

  // answering / confirming / submitting / submitted → exam view
  if (tasks.length === 0) {
    return <EmptyState title="没有可作答的题目" description="请重新生成练习题。" />;
  }

  const currentTask = tasks[state.currentIndex] ?? tasks[0];
  const currentDraft = state.drafts[currentTask.id] ?? { answer: '', selfAssess: 0 };
  const readonly = state.phase === 'submitting' || state.phase === 'submitted';
  const evalMap = new Map((evalQuery.data?.results ?? []).map((r) => [r.taskId, r]));

  const isLast = state.currentIndex === tasks.length - 1;

  return (
    <div className={s.exam}>
      <header className={s.examHead}>
        <h3 className={s.examTitle}>练习作答</h3>
        <span className={s.examMeta}>{tasks.length} 题</span>
        {state.phase === 'submitted' && (
          <span className={s.submittedBadge}>第 {state.attempt} 次已提交 · 等待评估</span>
        )}
      </header>

      <PracticeProgress
        tasks={tasks}
        drafts={state.drafts}
        currentIndex={state.currentIndex}
        onJump={(i) => dispatch({ type: 'SET_INDEX', index: i })}
        locked={readonly}
      />

      <QuestionCard
        task={currentTask}
        index={state.currentIndex}
        total={tasks.length}
        draft={currentDraft}
        onAnswerChange={(id, answer) => dispatch({ type: 'ANSWER', taskId: id, answer })}
        onAssessChange={(id, selfAssess) => dispatch({ type: 'ASSESS', taskId: id, selfAssess })}
        readonly={readonly}
        feedback={evalMap.get(currentTask.id)}
      />

      {state.phase === 'submitted' ? (
        <div className={s.nav}>
          <Button variant="outline" onClick={() => dispatch({ type: 'REGENERATE' })}>
            重新生成（新题）
          </Button>
          <span className={s.navInfo}>
            已提交的第 {state.attempt} 次作答保留在历史中，不会被覆盖。
          </span>
        </div>
      ) : (
        <div className={s.nav}>
          <Button
            variant="ghost"
            onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex - 1 })}
            disabled={state.currentIndex === 0}
          >
            ← 上一题
          </Button>
          <span className={s.navInfo}>第 {state.currentIndex + 1} / {tasks.length} 题</span>
          {isLast ? (
            <Button variant="primary" onClick={() => dispatch({ type: 'OPEN_CONFIRM' })}>
              提交全部
            </Button>
          ) : (
            <Button
              variant="outline"
              onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex + 1 })}
            >
              下一题 →
            </Button>
          )}
        </div>
      )}

      <ConfirmSubmitModal
        open={state.phase === 'confirming' || state.phase === 'submitting'}
        onOpenChange={(o) => { if (!o && state.phase === 'confirming') dispatch({ type: 'CLOSE_CONFIRM' }); }}
        tasks={tasks}
        drafts={state.drafts}
        onConfirm={handleSubmit}
        submitting={state.phase === 'submitting'}
      />
    </div>
  );
}
