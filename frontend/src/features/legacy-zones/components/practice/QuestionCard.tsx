import { Button } from '@/shared/primitive/Button';
import { Tag } from '@/shared/primitive/Tag';
import { Card } from '@/shared/primitive/Card';
import { useMarkdown } from '@/shared/hooks/useMarkdown';
import { AnswerField } from './AnswerField';
import { isEmptyPracticeAnswer, type DraftEntry } from '@/shared/lib/practiceDraft';
import {
  isObjectiveTask,
  type EvaluationResult,
  type ObjectiveResult,
  type PracticeTask,
} from '@/features/legacy-zones/api/practice';

import s from './QuestionCard.module.css';

const ASSESS_LABELS = ['', '完全不会', '勉强', '基本会', '较熟练', '能迁移'];

const TYPE_TAG: Record<PracticeTask['type'], { tone: 'sky' | 'orange' | 'pink' | 'accent'; label: string }> = {
  'true-false': { tone: 'sky', label: '判断' },
  'single-choice': { tone: 'sky', label: '单选' },
  'multiple-choice': { tone: 'accent', label: '多选' },
  'short-answer': { tone: 'sky', label: '简答' },
  essay: { tone: 'orange', label: '论述' },
  code: { tone: 'pink', label: '代码' },
};

interface QuestionCardProps {
  task: PracticeTask;
  index: number;
  total: number;
  draft: DraftEntry;
  onAnswerChange: (taskId: string, answer: DraftEntry['answer']) => void;
  onAssessChange: (taskId: string, selfAssess: number) => void;
  onCheckObjective: (task: PracticeTask, draft: DraftEntry) => void;
  checking?: boolean;
  readonly?: boolean;
  objectiveResult?: ObjectiveResult;
  feedback?: EvaluationResult;
}

export function QuestionCard({
  task,
  index,
  total,
  draft,
  onAnswerChange,
  onAssessChange,
  onCheckObjective,
  checking = false,
  readonly = false,
  objectiveResult,
  feedback,
}: QuestionCardProps) {
  const { html } = useMarkdown(task.question);
  const { html: suggestedAnswerHtml } = useMarkdown(feedback?.suggestedAnswer ?? '');
  const tag = TYPE_TAG[task.type] ?? TYPE_TAG.essay;
  const objective = isObjectiveTask(task);
  const answerLocked = readonly || !!objectiveResult;

  return (
    <Card variant="outlined" className={s.root}>
      <header className={s.head}>
        <span className={s.number}>第 {index + 1} 题</span>
        <Tag tone={tag.tone}>{tag.label}</Tag>
        <span className={s.stars} aria-label={`${task.difficulty} 星难度`}>
          {'★'.repeat(task.difficulty)}{'☆'.repeat(5 - task.difficulty)}
        </span>
        <span className={s.progress}>{index + 1} / {total}</span>
      </header>

      <div className={s.stem} dangerouslySetInnerHTML={{ __html: html }} />

      <div className={s.answerSection}>
        <div className={s.answerLabel}>作答</div>
        <AnswerField
          task={task}
          value={draft.answer}
          onChange={(answer) => onAnswerChange(task.id, answer)}
          readonly={answerLocked}
        />
        {objective && !objectiveResult && !readonly && (
          <Button
            variant="primary"
            size="sm"
            onClick={() => onCheckObjective(task, draft)}
            disabled={isEmptyPracticeAnswer(draft.answer)}
            loading={checking}
          >
            提交本题并查看解析
          </Button>
        )}
      </div>

      <div className={s.assessSection}>
        <div className={s.assessLabel}>掌握程度自评</div>
        <div className={s.scaleRow}>
          {[1, 2, 3, 4, 5].map((n) => (
            <button
              key={n}
              type="button"
              className={`${s.assessBtn} ${draft.selfAssess === n ? s.assessActive : ''}`}
              onClick={() => !readonly && onAssessChange(task.id, n)}
              disabled={readonly}
              title={ASSESS_LABELS[n]}
            >
              {n}
            </button>
          ))}
          {draft.selfAssess > 0 && <span className={s.assessText}>{ASSESS_LABELS[draft.selfAssess]}</span>}
        </div>
      </div>

      {objectiveResult && (
        <div className={`${s.feedback} ${objectiveResult.correct ? s.correct : s.incorrect}`}>
          <div className={s.feedbackHead}>
            {objectiveResult.correct ? '回答正确' : '回答错误'}
            {objectiveResult.growthDelta > 0 && <strong> +{objectiveResult.growthDelta} 成长值</strong>}
          </div>
          <div className={s.feedbackBody}>
            <strong>正确答案：</strong>{formatAnswer(objectiveResult.correctAnswer)}
            <br />
            {objectiveResult.explanation}
          </div>
        </div>
      )}

      {feedback && (
        <div className={s.feedback}>
          <div className={s.feedbackHead}>AI 评估 <strong>{feedback.score}/5</strong></div>
          <div className={s.feedbackBody}>
            <strong>点评</strong>
            <p>{feedback.feedback}</p>
            {feedback.suggestedAnswer && (
              <>
                <strong>参考回答</strong>
                <div
                  className={s.suggestedAnswer}
                  dangerouslySetInnerHTML={{ __html: suggestedAnswerHtml }}
                />
              </>
            )}
          </div>
        </div>
      )}
    </Card>
  );
}

function formatAnswer(answer: ObjectiveResult['correctAnswer']) {
  if (Array.isArray(answer)) return answer.join('、');
  if (typeof answer === 'boolean') return answer ? '正确' : '错误';
  return answer;
}
