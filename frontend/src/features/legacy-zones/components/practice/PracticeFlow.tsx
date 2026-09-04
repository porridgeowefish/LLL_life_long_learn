import { useEffect, useReducer, useRef } from 'react';

import { useInvokeAgent } from '@/features/agents';
import {
  isObjectiveTask,
  savePracticeDraftFile,
  useCheckObjective,
  useCreatePracticeAttempt,
  useLatestPracticeAttempt,
  usePracticeDraft,
  usePracticeEvaluation,
  usePracticeTasks,
  useRequestPracticeEvaluation,
  useSubmitPracticeAttempt,
  type PracticeDraftFile,
  type ObjectiveResult,
  type PracticeAnswer,
  type PracticeTask,
  type Submission,
} from '@/features/legacy-zones/api/practice';
import { useProjectProgress } from '@/features/legacy-zones/api/progress';
import { PERMISSION_MODES, PRACTICE_GEN_TIMEOUT_MS } from '@/shared/lib/constants';
import {
  draftMatches,
  isEmptyPracticeAnswer,
  loadDraft,
  saveDraft,
  type DraftEntry,
} from '@/shared/lib/practiceDraft';
import { Button } from '@/shared/primitive/Button';
import { EmptyState } from '@/shared/primitive/EmptyState';
import { GenerationEntry } from './GenerationEntry';
import { PracticeProgress } from './PracticeProgress';
import { QuestionCard } from './QuestionCard';
import { ConfirmSubmitModal } from './ConfirmSubmitModal';

import s from './PracticeFlow.module.css';

type Phase = 'idle' | 'choosing' | 'generating' | 'answering' | 'confirming' | 'submitting' | 'submitted' | 'error';
type ErrorKind = 'agent' | 'timeout' | 'corrupt' | null;
type EvaluationStatus = 'idle' | 'waiting' | 'failed' | 'complete';

interface State {
  phase: Phase;
  currentIndex: number;
  taskIds: string[];
  generatedAt: string | null;
  drafts: Record<string, DraftEntry>;
  attempt: number;
  objectiveResults: Record<string, ObjectiveResult>;
  evaluationStatus: EvaluationStatus;
  generationBaselineSetId: string | null;
  generationBaselineAt: string | null;
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
  evaluationStatus: 'idle',
  generationBaselineSetId: null,
  generationBaselineAt: null,
  errorKind: null,
};

type Action =
  | { type: 'GENERATE'; baselineSetId: string | null; baselineAt: string | null }
  | {
      type: 'TASKS_READY';
      taskIds: string[];
      generatedAt: string;
      drafts: Record<string, DraftEntry>;
      attempt: number;
      submitted: boolean;
      objectiveResults: Record<string, ObjectiveResult>;
    }
  | { type: 'ATTEMPT_READY'; attempt: number }
  | { type: 'TIMEOUT' | 'AGENT_FAILED' | 'CORRUPT' | 'OPEN_CONFIRM' | 'CLOSE_CONFIRM' | 'SUBMIT' | 'SUBMIT_FAILED' | 'CHOOSE_NEW_SET' | 'BACK_TO_IDLE' }
  | { type: 'SET_INDEX'; index: number }
  | { type: 'ANSWER'; taskId: string; answer: PracticeAnswer }
  | { type: 'ASSESS'; taskId: string; selfAssess: number }
  | { type: 'OBJECTIVE_CHECKED'; result: ObjectiveResult }
  | { type: 'SUBMITTED'; attempt: number }
  | { type: 'EVALUATION_WAITING' | 'EVALUATION_FAILED' | 'EVALUATION_COMPLETE' };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'GENERATE':
      return {
        ...state,
        phase: 'generating',
        generationBaselineSetId: action.baselineSetId,
        generationBaselineAt: action.baselineAt,
        errorKind: null,
      };
    case 'TASKS_READY':
      return {
        ...state,
        phase: action.submitted ? 'submitted' : 'answering',
        taskIds: action.taskIds,
        generatedAt: action.generatedAt,
        drafts: action.drafts,
        attempt: action.attempt,
        objectiveResults: action.objectiveResults,
        evaluationStatus: action.submitted ? 'waiting' : 'idle',
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
      return { ...state, phase: 'submitted', attempt: action.attempt, evaluationStatus: 'waiting' };
    case 'EVALUATION_WAITING':
      return { ...state, evaluationStatus: 'waiting' };
    case 'EVALUATION_FAILED':
      return { ...state, evaluationStatus: 'failed' };
    case 'EVALUATION_COMPLETE':
      return { ...state, evaluationStatus: 'complete' };
    case 'CHOOSE_NEW_SET':
      return { ...initialState, phase: 'choosing' };
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
  const requestEvaluation = useRequestPracticeEvaluation();
  const progress = useProjectProgress(projectSlug);
  const latestAttempt = useLatestPracticeAttempt(projectSlug);
  const serverDraft = usePracticeDraft(projectSlug);
  const [state, dispatch] = useReducer(reducer, initialState);
  const requestedEvaluations = useRef(new Set<number>());

  const tasksData = usePracticeTasks(projectSlug);
  const tasks = tasksData.data?.tasks ?? [];
  const generatedAt = tasksData.data?.generatedAt;
  const hasTasks = tasksData.data?.generated === true && Array.isArray(tasks) && tasks.length > 0;
  const evalQuery = usePracticeEvaluation(
    state.phase === 'submitted' ? projectSlug : undefined,
    state.attempt,
  );

  useEffect(() => {
    if (
      tasksData.isLoading ||
      latestAttempt.isLoading ||
      serverDraft.isLoading ||
      (state.phase !== 'idle' && state.phase !== 'generating')
    ) return;
    if (hasTasks && generatedAt) {
      const taskIds = tasks.map((task) => task.id);
      if (
        state.phase === 'generating' &&
        tasksData.data?.setId === state.generationBaselineSetId &&
        generatedAt === state.generationBaselineAt
      ) {
        return;
      }
      const stored = loadDraft(projectSlug);
      const localMatches = draftMatches(stored, taskIds, generatedAt);
      const diskDraft = serverDraft.data;
      const diskMatches = draftFileMatches(diskDraft, tasksData.data?.setId, taskIds, generatedAt);
      const restored = diskMatches
        ? { ...diskDraft!.drafts }
        : (localMatches ? { ...stored!.drafts } : {});
      const serverAttempt = latestAttempt.data;
      const restoresSubmittedAttempt =
        serverAttempt?.status === 'submitted' &&
        serverAttempt.setId === tasksData.data?.setId;
      if (restoresSubmittedAttempt) {
        for (const submission of serverAttempt.submissions ?? []) {
          const previous = restored[submission.taskId];
          const submittedDraft = {
            answer: submission.answer,
            selfAssess: submission.selfAssess,
          };
          restored[submission.taskId] =
            isEmptyPracticeAnswer(submission.answer) &&
            previous &&
            !isEmptyPracticeAnswer(previous.answer)
              ? previous
              : submittedDraft;
        }
        for (const result of Object.values(serverAttempt.objectiveResults ?? {})) {
          const previous = restored[result.taskId] ?? { answer: result.answer, selfAssess: 0 };
          restored[result.taskId] = { ...previous, answer: result.answer };
        }
      }
      dispatch({
        type: 'TASKS_READY',
        taskIds,
        generatedAt,
        drafts: restored,
        attempt: restoresSubmittedAttempt
          ? serverAttempt.attempt
          : (diskMatches ? diskDraft?.attempt ?? 0 : (localMatches ? stored?.attempt ?? 0 : 0)),
        submitted: restoresSubmittedAttempt,
        objectiveResults: restoresSubmittedAttempt ? serverAttempt.objectiveResults ?? {} : {},
      });
    }
  }, [
    generatedAt,
    hasTasks,
    latestAttempt.data,
    latestAttempt.isLoading,
    projectSlug,
    serverDraft.data,
    serverDraft.isLoading,
    state.phase,
    tasks,
    tasksData.data?.setId,
    tasksData.isLoading,
    state.generationBaselineAt,
    state.generationBaselineSetId,
  ]);

  useEffect(() => {
    if (state.phase !== 'generating') return;
    const timer = setTimeout(() => dispatch({ type: 'TIMEOUT' }), PRACTICE_GEN_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [state.phase]);

  useEffect(() => {
    if ((state.phase !== 'answering' && state.phase !== 'confirming') || !state.generatedAt) return;
    const localDraft = {
      generatedAt: state.generatedAt,
      taskIds: state.taskIds,
      drafts: state.drafts,
      attempt: state.attempt || undefined,
    };
    saveDraft(projectSlug, localDraft);
    const setId = tasksData.data?.setId;
    if (!setId || state.taskIds.length === 0) return;
    const timer = window.setTimeout(() => {
      void savePracticeDraftFile(projectSlug, {
        setId,
        generatedAt: state.generatedAt!,
        taskIds: state.taskIds,
        drafts: state.drafts,
        attempt: state.attempt || undefined,
      });
    }, 400);
    return () => window.clearTimeout(timer);
  }, [
    projectSlug,
    state.attempt,
    state.drafts,
    state.generatedAt,
    state.phase,
    state.taskIds,
    tasksData.data?.setId,
  ]);

  useEffect(() => {
    if (evalQuery.data && state.evaluationStatus !== 'complete') {
      dispatch({ type: 'EVALUATION_COMPLETE' });
      void progress.refetch();
    }
  }, [evalQuery.data, progress, state.evaluationStatus]);

  useEffect(() => {
    if (
      state.phase !== 'submitted' ||
      state.attempt < 1 ||
      evalQuery.data ||
      !evalQuery.isError ||
      requestedEvaluations.current.has(state.attempt)
    ) {
      return;
    }
    const hasSubjectiveAnswer = tasks.some((task) => {
      const answer = state.drafts[task.id]?.answer;
      return !isObjectiveTask(task) && typeof answer === 'string' && answer.trim() !== '';
    });
    if (!hasSubjectiveAnswer) return;
    requestedEvaluations.current.add(state.attempt);
    dispatch({ type: 'EVALUATION_WAITING' });
    void requestEvaluation.mutateAsync({ projectSlug, attempt: state.attempt }).catch(() => {
      dispatch({ type: 'EVALUATION_FAILED' });
    });
  }, [
    evalQuery.data,
    evalQuery.isError,
    projectSlug,
    requestEvaluation,
    state.attempt,
    state.drafts,
    state.phase,
    tasks,
  ]);

  const handleGenerate = async (count: number) => {
    dispatch({
      type: 'GENERATE',
      baselineSetId: tasksData.data?.setId ?? null,
      baselineAt: generatedAt ?? null,
    });
    try {
      await invoke.mutateAsync({
        agentId: 'practice',
        payload: {
          projectId: projectSlug,
          zone: 'Practice',
          practiceQuestionCount: count,
          intent: `生成 ${count} 道由易到难的练习题。按一星到五星递进，包含判断题、选择题和主观迁移题；同时写 tasks.json 与 answer-key.json。`,
          permissionMode: PERMISSION_MODES.auto,
        },
      });
    } catch {
      dispatch({ type: 'AGENT_FAILED' });
    }
  };

  const ensureAttempt = async () => {
    if (state.attempt > 0) return state.attempt;
    const result = await createAttempt.mutateAsync(projectSlug);
    dispatch({ type: 'ATTEMPT_READY', attempt: result.attempt });
    return result.attempt;
  };

  const handleCheckObjective = async (task: PracticeTask, draft: DraftEntry) => {
    const attempt = await ensureAttempt();
    const response = await checkObjective.mutateAsync({
      projectSlug,
      attempt,
      taskId: task.id,
      answer: draft.answer,
    });
    dispatch({ type: 'OBJECTIVE_CHECKED', result: response.result });
  };

  const handleSubmit = async () => {
    dispatch({ type: 'SUBMIT' });
    const submissions: Submission[] = tasks
      .filter((task) => !isObjectiveTask(task))
      .map((task) => {
        const draft = state.drafts[task.id] ?? { answer: '', selfAssess: 0 };
        return { taskId: task.id, answer: draft.answer, selfAssess: draft.selfAssess };
      });
    let response: { attempt: number; saved: number; growthDelta: number };
    try {
      const attempt = await ensureAttempt();
      response = await submitAttempt.mutateAsync({
        projectSlug,
        attempt,
        submissions,
      });
      saveDraft(projectSlug, {
        generatedAt: state.generatedAt ?? generatedAt ?? '',
        taskIds: state.taskIds,
        drafts: state.drafts,
        attempt: response.attempt,
      });
      dispatch({ type: 'SUBMITTED', attempt: response.attempt });
    } catch {
      dispatch({ type: 'SUBMIT_FAILED' });
      return;
    }
    const hasSubjectiveAnswer = submissions.some(
      (submission) => typeof submission.answer === 'string' && submission.answer.trim() !== '',
    );
    if (hasSubjectiveAnswer) {
      requestedEvaluations.current.add(response.attempt);
      dispatch({ type: 'EVALUATION_WAITING' });
      try {
        await requestEvaluation.mutateAsync({ projectSlug, attempt: response.attempt });
      } catch {
        dispatch({ type: 'EVALUATION_FAILED' });
      }
    }
  };

  if (state.phase === 'idle' || state.phase === 'choosing') {
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
  const hasSubjectiveAnswers = tasks.some((task) => {
    const answer = state.drafts[task.id]?.answer;
    return !isObjectiveTask(task) && typeof answer === 'string' && answer.trim() !== '';
  });

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

      {state.phase === 'submitted' && hasSubjectiveAnswers && (
        <section className={s.evaluationSummary}>
          <div className={s.evaluationSummaryHead}>
            <strong>AI 反馈总结</strong>
            {evalQuery.data && <span>{evalQuery.data.overallScore.toFixed(1)} / 5</span>}
          </div>
          {evalQuery.data?.summary ? (
            <p>{evalQuery.data.summary}</p>
          ) : (
            <p>{evaluationStatusText(state.evaluationStatus)}</p>
          )}
        </section>
      )}

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
            <Button
              variant="ghost"
              onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex - 1 })}
              disabled={state.currentIndex === 0}
            >
              上一题
            </Button>
            <span className={s.navInfo}>
              {evalQuery.data ? '逐题反馈与参考回答已生成，可点击上方题号查看。' : evaluationStatusText(state.evaluationStatus)}
            </span>
            {isLast ? (
              <Button variant="outline" onClick={() => dispatch({ type: 'CHOOSE_NEW_SET' })}>生成新题组</Button>
            ) : (
              <Button variant="outline" onClick={() => dispatch({ type: 'SET_INDEX', index: state.currentIndex + 1 })}>
                下一题
              </Button>
            )}
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

function evaluationStatusText(status: EvaluationStatus) {
  if (status === 'waiting') return '答案已保留，AI 正在后台生成总体反馈和逐题参考回答。';
  if (status === 'failed') return '答案已保留，但 AI 反馈生成失败。';
  if (status === 'complete') return 'AI 评估已完成。';
  return '答案已提交，等待 AI 评估。';
}

function draftFileMatches(
  draft: PracticeDraftFile | null | undefined,
  setId: string | undefined,
  taskIds: string[],
  generatedAt: string | undefined,
) {
  if (!draft || !setId || !generatedAt) return false;
  if (draft.setId !== setId || draft.generatedAt !== generatedAt) return false;
  if (draft.taskIds.length !== taskIds.length) return false;
  return taskIds.every((id, index) => draft.taskIds[index] === id);
}
