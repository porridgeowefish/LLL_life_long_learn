// QuestionCard — single-question exam card (one question shown at a time).
// Header (number + type tag) → markdown stem → AnswerField → 1-5 self-assess.
// submitted state shows feedback.

import { Tag } from '@/components/primitive/Tag';
import { Card } from '@/components/primitive/Card';
import { useMarkdown } from '@/hooks/useMarkdown';
import { AnswerField } from './AnswerField';
import type { PracticeTask, EvaluationResult } from '@/api/practice';
import type { DraftEntry } from '@/lib/practiceDraft';

import s from './QuestionCard.module.css';

// AC P-06 self-assessment labels (1-5 mastery scale).
const ASSESS_LABELS = ['', '完全不会', '勉强', '基本会', '较熟练', '能迁移'];

const TYPE_TAG: Record<PracticeTask['type'], { tone: 'sky' | 'orange' | 'pink'; label: string }> = {
  'short-answer': { tone: 'sky', label: '简答' },
  essay: { tone: 'orange', label: '论述' },
  code: { tone: 'pink', label: '代码' },
};

interface QuestionCardProps {
  task: PracticeTask;
  index: number;
  total: number;
  draft: DraftEntry;
  onAnswerChange: (taskId: string, answer: string) => void;
  onAssessChange: (taskId: string, selfAssess: number) => void;
  readonly?: boolean;
  feedback?: EvaluationResult;
}

export function QuestionCard({
  task,
  index,
  total,
  draft,
  onAnswerChange,
  onAssessChange,
  readonly = false,
  feedback,
}: QuestionCardProps) {
  const { html } = useMarkdown(task.question);
  const tag = TYPE_TAG[task.type] ?? TYPE_TAG.essay;

  return (
    <Card variant="outlined" className={s.root}>
      <header className={s.head}>
        <span className={s.number}>第 {index + 1} 题</span>
        <Tag tone={tag.tone}>{tag.label}</Tag>
        <span className={s.progress}>{index + 1} / {total}</span>
      </header>

      <div
        className={s.stem}
        // eslint-disable-next-line react/no-danger -- sanitised in useMarkdown
        dangerouslySetInnerHTML={{ __html: html }}
      />

      <div className={s.answerSection}>
        <div className={s.answerLabel}>作答</div>
        <AnswerField
          type={task.type}
          value={draft.answer}
          onChange={(v) => onAnswerChange(task.id, v)}
          readonly={readonly}
        />
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
          {draft.selfAssess > 0 && (
            <span className={s.assessText}>{ASSESS_LABELS[draft.selfAssess]}</span>
          )}
        </div>
      </div>

      {feedback && (
        <div className={s.feedback}>
          <div className={s.feedbackHead}>
            评估 <strong>{feedback.score}/5</strong> {feedback.passed ? '✅' : '❌'}
          </div>
          <div className={s.feedbackBody}>{feedback.feedback}</div>
        </div>
      )}
    </Card>
  );
}
