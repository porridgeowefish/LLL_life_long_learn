// PracticeFlow — state machine for the Practice zone:
// idle → drafting → self-assessing → submitting → evaluating → evaluated
//
// Tasks come from the Explain Agent writing practice/tasks.json.
// Learner answers each question, self-assesses 1-5, then batch submits.

import { useCallback, useState } from 'react';

import { usePracticeTasks, useSubmitPractice, type Submission } from '@/api/practice';
import { QuestionCard } from './QuestionCard';
import { Button } from '@/components/primitive/Button';
import { EmptyState } from '@/components/primitive/EmptyState';

import s from './PracticeFlow.module.css';

type Phase = 'idle' | 'drafting' | 'submitting' | 'evaluating' | 'evaluated';

interface PracticeFlowProps {
  projectSlug: string;
}

interface Draft {
  answer: string;
  selfAssess: number;
}

export function PracticeFlow({ projectSlug }: PracticeFlowProps) {
  const { data: tasksData, isLoading } = usePracticeTasks(projectSlug);
  const submitPractice = useSubmitPractice();

  const [phase, setPhase] = useState<Phase>('idle');
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [attempt, setAttempt] = useState(0);

  const tasks = tasksData?.tasks ?? [];
  const hasTasks = tasksData?.generated === true && tasks.length > 0;

  const getDraft = useCallback(
    (taskId: string): Draft => drafts[taskId] ?? { answer: '', selfAssess: 0 },
    [drafts],
  );

  const handleChange = useCallback((taskId: string, answer: string, selfAssess: number) => {
    setDrafts((prev) => ({ ...prev, [taskId]: { answer, selfAssess } }));
  }, []);

  const allAnswered = tasks.length > 0 && tasks.every((t) => {
    const d = drafts[t.id];
    return d && d.answer.trim().length > 0 && d.selfAssess > 0;
  });

  const handleSubmit = async () => {
    const submissions: Submission[] = tasks.map((t) => {
      const d = getDraft(t.id);
      return { taskId: t.id, answer: d.answer, selfAssess: d.selfAssess };
    });
    setPhase('submitting');
    try {
      const res = await submitPractice.mutateAsync({ projectSlug, submissions });
      setAttempt(res.attempt);
      setPhase('evaluating');
      // In iter-03, evaluation is triggered via Agent invoke (Practice Agent).
      // The UI shows "已提交，等待评估" until evaluation file appears.
    } catch {
      setPhase('drafting');
    }
  };

  if (isLoading) {
    return <div className={s.loading}>加载中…</div>;
  }

  if (!hasTasks) {
    return (
      <EmptyState
        title="练习题尚未生成"
        description={<>调用 Practice Agent 生成练习题后，在这里作答。</>}
      />
    );
  }

  return (
    <div className={s.root}>
      <header className={s.head}>
        <h3 className={s.title}>练习作答</h3>
        <span className={s.count}>{tasks.length} 题</span>
        {phase === 'evaluating' && (
          <span className={s.evalHint}>已提交第 {attempt} 次，等待 Practice Agent 评估…</span>
        )}
      </header>

      <div className={s.cards}>
        {tasks.map((task) => (
          <QuestionCard
            key={task.id}
            task={task}
            answer={getDraft(task.id).answer}
            selfAssess={getDraft(task.id).selfAssess}
            onChange={handleChange}
            readonly={phase === 'submitting' || phase === 'evaluating' || phase === 'evaluated'}
          />
        ))}
      </div>

      {(phase === 'idle' || phase === 'drafting') && (
        <footer className={s.foot}>
          <Button
            variant="primary"
            onClick={handleSubmit}
            disabled={!allAnswered}
            loading={submitPractice.isPending}
          >
            {allAnswered ? '提交全部答案' : `请完成所有题目（${tasks.filter((t) => {
              const d = drafts[t.id];
              return d && d.answer.trim() && d.selfAssess > 0;
            }).length}/${tasks.length}）`}
          </Button>
        </footer>
      )}
    </div>
  );
}
