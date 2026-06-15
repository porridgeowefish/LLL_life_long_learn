// PracticeProgress — question-number grid at the top of the exam view.
// Each cell shows P-05 state: current / unanswered / answered-not-assessed /
// completed (answered + self-assessed).

import clsx from 'clsx';

import type { PracticeTask } from '@/api/practice';
import type { DraftEntry } from '@/lib/practiceDraft';
import { isEmptyPracticeAnswer } from '@/lib/practiceDraft';
import { Icon } from '@/components/primitive/Icon';

import s from './PracticeProgress.module.css';

interface PracticeProgressProps {
  tasks: PracticeTask[];
  drafts: Record<string, DraftEntry>;
  currentIndex: number;
  onJump: (index: number) => void;
  locked?: boolean; // submitted state — read-only grid
}

type CellState = 'current' | 'unanswered' | 'answered' | 'completed';

function cellState(
  draft: DraftEntry | undefined,
  isCurrent: boolean,
  locked: boolean,
): CellState {
  if (isCurrent && !locked) return 'current';
  if (!draft || isEmptyPracticeAnswer(draft.answer)) return 'unanswered';
  if (draft.selfAssess === 0) return 'answered';
  return 'completed';
}

export function PracticeProgress({
  tasks,
  drafts,
  currentIndex,
  onJump,
  locked = false,
}: PracticeProgressProps) {
  return (
    <div className={s.grid}>
      {tasks.map((task, i) => {
        const draft = drafts[task.id];
        const state = cellState(draft, i === currentIndex, locked);
        return (
          <button
            key={task.id}
            type="button"
            className={clsx(s.cell, s[state], locked && s.locked)}
            onClick={() => !locked && onJump(i)}
            title={`第 ${i + 1} 题`}
            aria-label={`第 ${i + 1} 题，状态 ${state}`}
            aria-current={i === currentIndex ? 'true' : undefined}
          >
            {state === 'completed' ? <Icon name="check" size={11} /> : i + 1}
          </button>
        );
      })}
    </div>
  );
}
