import { useEffect, useReducer } from 'react';

import { useInvokeAgent } from '@/api/agents';
import {
  isObjectiveTask,
  useCheckObjective,
  useCreatePracticeAttempt,
  usePracticeEvaluation,
  usePracticeTasks,
  useSubmitPracticeAttempt,
  type ObjectiveResult,
  type PracticeAnswer,
  type PracticeTask,
  type Submission,
} from '@/api/practice';
import { useProjectProgress } from '@/api/progress';
import { PERMISSION_MODES, PRACTICE_GEN_POLL_MS, PRACTICE_GEN_TIMEOUT_MS } from '@/lib/constants';
import {
  clearDraft,
  draftMatches,
  loadDraft,
  saveDraft,
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
  objectiveResults: Record<string, ObjectiveResult>;
  errorKind: ErrorKind;
}

const initialState: State = {
  phase: 'idle',
  currentIndex: 0,
  taskIds: [],
  generatedAt: null,
  drafts: {},
  attempt: 0,
  objectiveResults: {},
  errorKind: null,
};

type Action =
  | { type: 'GENERATE' }
  | { type: 'TASKS_READY'; taskIds: string[]; generatedAt: string; drafts: Record<string, DraftEntry>; attempt: number }
  | { type: 'ATTEMPT_READY'; attempt: number }
  | { type: 'TIMEOUT' | 'AGENT_FAILED' | 'CORRUPT' | 'OPEN_CONFIRM' | 'CLOSE_CONFIRM' | 'SUBMIT' | 'SUBMIT_FAILED' | 'REGENERATE' | 'BACK_TO_IDLE' }
  | { type: 'SET_INDEX'; index: number }
  | { type: 'ANSWER'; taskId: string; answer: PracticeAnswer }
  | { type: 'ASSESS'; taskId: string; selfAssess: number }
  | { type: 'OBJECTIVE_CHECKED'; result: ObjectiveResult }
  | { type: 'SUBMITTED'; attempt: number };

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
        attempt: action.attempt,
        currentIndex: 0,
      };
    case 'ATTEMPT_READY':
      return { ...state, attempt: action.attempt };
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
    case 'OBJECTIVE_CHECKED':
      return {
        ...state,
        objectiveResults: { ...state.objectiveResults, [action.result.taskId]: action.result },
      };
    case 'OPEN_CONFIRM':
      return { ...state, phase: 'confirming' };
    case 'CLOSE_CONFIRM':
    case 'SUBMIT_FAILED':
      return { ...state, phase: 'answering' };
    case 'SUBMIT':
      return { ...state, phase: 'submitting' };
    case 'SUBMITTED':
      return { ...state, phase: 'submitted', attempt: action.attempt };
    case 'REGENERATE':
      return { ...initialState, phase: 'generating' };
    case 'BACK_TO_IDLE':
      return { ...initialState };
    default:
      return state;
  }
}

export function PracticeFlow({ projectSlug }: { projectSlug: string }) {
  const invoke = useInvokeAgent();
  const createAttempt = useCreatePracticeAttempt();
  const checkObjective = useCheckObjective();
  const submitAttempt = useSubmitPracticeAttempt();
  const progress = useProjectProgress(projectSlug);
  const [state, dispatch] = useReducer(reducer, initialState);

  const tasksData = usePracticeTasks(projectSlug, {
    refetchInterval: state.phase === 'generating' ? PRACTICE_GEN_POLL_MS : false,
  });
  const tasks = tasksData.data?.tasks ?? [];
  const generatedAt = tasksData.data?.generatedAt;
  const hasTasks = tasksData.data?.generated === true && Array.isArray(tasks) && tasks.length > 0;
  const evalQuery = usePracticeEvaluation(
    state.phase === 'submitted' ? projectSlug : undefined,
    state.attempt,
  );

  useEffect(() => {
    if (tasksData.isLoading || (state.phase !== 'idle' && state.phase !== 'generating')) return;
    if (hasTasks && generatedAt) {
      const stored = loadDraft(projectSlug);
      const matches = draftMatches(stored, tasks.map((task) => task.id), generatedAt);
      dispatch({
        type: 'TASKS_READY',
        taskIds: tasks.map((task) => task.id),
        generatedAt,
        drafts: matches ? stored!.drafts : {},
        attempt: matches ? stored?.attempt ?? 0 : 0,
      });
    }
  }, [generatedAt, hasTasks, projectSlug, state.phase, tasks, tasksData.isLoading]);

  useEffect(() => {
    if (state.phase !== 'answering' || state.attempt > 0 || createAttempt.isPending) return;
    void createAttempt.mutateAsync(projectSlug).then((result) => {
      dispatch({ type: 'ATTEMPT_READY', attempt: result.attempt });
    });
  }, [createAttempt, projectSlug, state.attempt, state.phase]);

  useEffect(() => {
    if (state.phase !== 'generating') return;
    const timer = setTimeout(() => dispatch({ type: 'TIMEOUT' }), PRACTICE_GEN_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [state.phase]);

  useEffect(() => {
    if ((state.phase !== 'answering' && state.phase !== 'confirming') || !state.generatedAt) return;
    saveDraft(projectSlug, {
      generatedAt: state.generatedAt,
      taskIds: state.taskIds,
      drafts: state.drafts,
      attempt: state.attempt || undefined,
    });
  }, [projectSlug, state.attempt, state.drafts, state.generatedAt, state.phase, state.taskIds]);

  const handleGenerate = async (count: number) => {
    try {
      await invoke.mutateAsync({
        agentId: 'practice',
        payload: {
          projectId: projectSlug,
          zone: 'Practice',
          intent: `生成 ${count} 道由易到难的练习题。按一星到五星递进，包含判断题、选择题和主观迁移题；同时写 tasks.json 与 answer-key.json。`,
          permissionMode: PERMISSION_MODES.acceptEdits,
        },
      });
      dispatch({ type: 'GENERATE' });
    } catch {
      dispatch({ type: 'AGENT_FAILED' });
    }
  };

  const handleCheckObjective = async (task: PracticeTask, draft: DraftEntry) => {
    if (state.attempt < 1) return;
    const response = await checkObjective.mutateAsync({
      projectSlug,
      attempt: state.attempt,
      taskId: task.id,
      answer: draft.answer,
    });
    dispatch({ type: 'OBJECTIVE_CHECKED', result: response.result });
  };

  const handleSubmit = async () => {
    if (state.attempt < 1) return;
    dispatch({ type: 'SUBMIT' });
    const submissions: Submission[] = tasks
      .filter((task) => !isObjectiveTask(task))
      .map((task) => {
        const draft = state.drafts[task.id] ?? { answer: '', selfAssess: 0 };
        return { taskId: task.id, answer: draft.answer, selfAssess: draft.selfAssess };
      });
    try {
      const response = await submitAttempt.mutateAsync({
        projectSlug,
        attempt: state.attempt,
        submissions,
      });
      clearDraft(projectSlug);
      dispatch({ type: 'SUBMITTED', attempt: response.attempt });
    } catch {
      dispatch({ type: 'SUBMIT_FAILED' });
    }
  };

  if (state.phase === 'idle') {
    return <GenerationEntry onGenerate={handleGenerate} isInvoking={invoke.isPending} />;
  }
  if (state.phase === 'generating') {
    return <EmptyState title="正在生成渐进题组…" description="题目与答案键就绪后会自动显示。" />;
  }
  if (state.phase === 'error') {
    return (
      <EmptyState
        title="练习题暂时不可用"
        description={`错误类型：${state.errorKind ?? 'unknown'}`}
        action={<Button onClick={() => dispatch({ type: 'BACK_TO_IDLE' })}>返回题量选择</Button>}
      />
    );
  }
  if (tasks.length === 0) {
    return <EmptyState title="没有可作答的题目" description="请重新生成练习题。" />;
  }

  const currentTask = tasks[state.currentIndex] ?? tasks[0];
  const currentDraft = state.drafts[currentTask.id] ?? { answer: '', selfAssess: 0 };
  const readonly = state.phase === 'submitting' || state.phase === 'submitted';
  const evalMap = new Map((evalQuery.data?.results ?? []).map((result) => [result.taskId, result]));
  const isLast = state.currentIndex === tasks.length - 1;

  return (
    <div className={s.exam}>
      <header className={s.examHead}>
        <h3 className={s.examTitle}>渐进练习</h3>
        <span className={s.examMeta}>{tasks.length} 题</span>
        <span className={s.growth}>项目成长值 <strong>{progress.data?.total ?? 0}</strong></span>
      </header>

      <PracticeProgress
        tasks={tasks}
        drafts={state.drafts}
        currentIndex={state.currentIndex}
        onJump={(index) => dispatch({ type: 'SET_INDEX', index })}
        locked={readonly}
      />

      <QuestionCard
        task={currentTask}
        index={state.currentIndex}
        total={tasks.length}
        draft={currentDraft}
        onAnswerChange={(taskId, answer) => dispatch({ type: 'ANSWER', taskId, answer })}
        onAssessChange={(taskId, selfAssess) => dispatch({ type: 'ASSESS', taskId, selfAssess })}
        onCheckObjective={handleCheckObjective}
        checking={checkObjective.isPending}
        readonly={readonly}
        objectiveResult={state.objectiveResults[currentTask.id]}
        feedback={evalMap.get(currentTask.id)}
      />

      <div className={s.nav}>
        {state.phase === 'submitted' ? (
          <>
            <Button variant="outline" onClick={() => dispatch({ type: 'REGENERATE' })}>生成新题组</Button>
            <span className={s.navInfo}>
              {evalQuery.data ? `主观题评估完成：${evalQuery.data.overallScore.toFixed(1)} / 5` : '客观题已即时判定，主观题等待整批评估。'}
            </span>
          </>
        ) : (
          <>
            <Button
              variant="ghost"
              onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex - 1 })}
              disabled={state.currentIndex === 0}
            >
              上一题
            </Button>
            <span className={s.navInfo}>第 {state.currentIndex + 1} / {tasks.length} 题</span>
            {isLast ? (
              <Button
                variant="primary"
                onClick={() => dispatch({ type: 'OPEN_CONFIRM' })}
                disabled={state.attempt < 1}
              >
                提交整组
              </Button>
            ) : (
              <Button variant="outline" onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex + 1 })}>
                下一题
              </Button>
            )}
          </>
        )}
      </div>

      <ConfirmSubmitModal
        open={state.phase === 'confirming' || state.phase === 'submitting'}
        onOpenChange={(open) => { if (!open && state.phase === 'confirming') dispatch({ type: 'CLOSE_CONFIRM' }); }}
        tasks={tasks}
        drafts={state.drafts}
        onConfirm={handleSubmit}
        submitting={state.phase === 'submitting'}
      />
    </div>
  );
}
