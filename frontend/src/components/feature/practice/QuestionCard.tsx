// QuestionCard — one practice question with MarkdownEditor for answer + self-assess.

import { useState } from 'react';
import clsx from 'clsx';

import type { PracticeTask } from '@/api/practice';

import s from './QuestionCard.module.css';

interface QuestionCardProps {
  task: PracticeTask;
  answer: string;
  selfAssess: number;
  onChange: (taskId: string, answer: string, selfAssess: number) => void;
  readonly?: boolean;
  score?: number;
  feedback?: string;
}

const MASTERY_LABELS = ['', '完全不会', '有印象', '理解思路', '能写出', '熟练掌握'];

export function QuestionCard({
  task,
  answer,
  selfAssess,
  onChange,
  readonly = false,
  score,
  feedback,
}: QuestionCardProps) {
  const [expanded, setExpanded] = useState(true);

  return (
    <div className={clsx(s.root, readonly && s.readonly)}>
      <header className={s.head} onClick={() => setExpanded(!expanded)}>
        <span className={s.toggle}>{expanded ? '▾' : '▸'}</span>
        <span className={s.typeTag}>{task.type}</span>
        <span className={s.questionPreview}>
          {task.question.slice(0, 80)}{task.question.length > 80 ? '…' : ''}
        </span>
        {score !== undefined && (
          <span className={clsx(s.scoreTag, score >= 3 ? s.pass : s.fail)}>
            {score}/5
          </span>
        )}
      </header>

      {expanded && (
        <div className={s.body}>
          <div className={s.questionFull}>{task.question}</div>

          {readonly ? (
            <div className={s.answerDisplay}>
              <strong>你的回答：</strong>
              <p>{answer || '（未作答）'}</p>
            </div>
          ) : (
            <textarea
              className={s.answerInput}
              placeholder="在这里写回答…"
              value={answer}
              onChange={(e) => onChange(task.id, e.target.value, selfAssess)}
              rows={5}
              spellCheck={false}
            />
          )}

          {readonly ? (
            <div className={s.assessDisplay}>
              自评：{MASTERY_LABELS[selfAssess] || `${selfAssess}/5`}
            </div>
          ) : (
            <div className={s.assessRow}>
              <span className={s.assessLabel}>掌握程度：</span>
              {[1, 2, 3, 4, 5].map((n) => (
                <button
                  key={n}
                  className={clsx(s.assessBtn, selfAssess === n && s.assessActive)}
                  onClick={() => onChange(task.id, answer, n)}
                  title={MASTERY_LABELS[n]}
                  type="button"
                >
                  {n}
                </button>
              ))}
              {selfAssess > 0 && (
                <span className={s.assessHint}>{MASTERY_LABELS[selfAssess]}</span>
              )}
            </div>
          )}

          {feedback && (
            <div className={s.feedback}>
              <strong>反馈：</strong> {feedback}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
