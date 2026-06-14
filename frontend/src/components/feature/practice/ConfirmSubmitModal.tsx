// ConfirmSubmitModal — P-07 pre-submit confirmation.
// Summarises: total questions, unanswered (warn, allows partial), and
// self-assess gaps (hard block — disables submit until every answered
// question is self-assessed, since self-assessment precedes AI evaluation).

import clsx from 'clsx';

import { Modal } from '@/components/primitive/Modal';
import { Button } from '@/components/primitive/Button';
import type { PracticeTask } from '@/api/practice';
import type { DraftEntry } from '@/lib/practiceDraft';

import s from './ConfirmSubmitModal.module.css';

interface ConfirmSubmitModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tasks: PracticeTask[];
  drafts: Record<string, DraftEntry>;
  onConfirm: () => void;
  submitting?: boolean;
}

export function ConfirmSubmitModal({
  open,
  onOpenChange,
  tasks,
  drafts,
  onConfirm,
  submitting = false,
}: ConfirmSubmitModalProps) {
  let unanswered = 0;
  let missingAssess = 0;
  for (const t of tasks) {
    const d = drafts[t.id];
    if (!d || d.answer.trim() === '') {
      unanswered++;
    } else if (d.selfAssess === 0) {
      missingAssess++;
    }
  }
  const blocked = missingAssess > 0;

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="确认提交"
      description="提交后将由练习智能体整批评估，形成新的一次 attempt。"
      size="md"
      footer={
        <>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            返回修改
          </Button>
          <Button variant="primary" onClick={onConfirm} disabled={blocked} loading={submitting}>
            {blocked ? '请先完成自评' : unanswered > 0 ? `仍要提交（${unanswered} 题为空）` : '确认提交'}
          </Button>
        </>
      }
    >
      <ul className={s.checks}>
        <li className={s.check}>
          <span className={s.checkIcon}>✅</span>
          共 {tasks.length} 题
        </li>
        <li className={clsx(s.check, unanswered === 0 && s.checkOk)}>
          <span className={s.checkIcon}>{unanswered === 0 ? '✅' : '⚠️'}</span>
          {unanswered === 0 ? '所有题已作答' : `${unanswered} 题未作答（允许部分提交）`}
        </li>
        <li className={clsx(s.check, missingAssess === 0 && s.checkOk)}>
          <span className={s.checkIcon}>{missingAssess === 0 ? '✅' : '⏳'}</span>
          {missingAssess === 0
            ? '已作答题均完成自评'
            : `${missingAssess} 题已作答但未自评（必须完成才能提交）`}
        </li>
      </ul>
      {blocked && (
        <p className={s.note}>
          自评发生在 AI 评价之前——它是你对自己掌握程度的校准。请回到未自评的题目补上。
        </p>
      )}
    </Modal>
  );
}
