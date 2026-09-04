// ConfirmSubmitModal — P-07 pre-submit confirmation.
// Summarises: total questions, unanswered (warn, allows partial), and
// unanswered questions are blocked so accidental empty attempts cannot replace
// the learner's last meaningful submission.

import clsx from 'clsx';

import { Modal } from '@/shared/primitive/Modal';
import { Button } from '@/shared/primitive/Button';
import type { PracticeTask } from '@/features/legacy-zones/api/practice';
import type { DraftEntry } from '@/shared/lib/practiceDraft';
import { isEmptyPracticeAnswer } from '@/shared/lib/practiceDraft';

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
    if (!d || isEmptyPracticeAnswer(d.answer)) {
      unanswered++;
    } else if (d.selfAssess === 0) {
      missingAssess++;
    }
  }
  const blocked = missingAssess > 0 || unanswered > 0;

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="确认提交"
      description="提交后保留全部题目与答案，并在后台生成主观题反馈。"
      size="md"
      footer={
        <>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            返回修改
          </Button>
          <Button variant="primary" onClick={onConfirm} disabled={blocked} loading={submitting}>
            {unanswered > 0 ? '请先完成全部题目' : blocked ? '请先完成自评' : '确认提交'}
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
          {unanswered === 0 ? '所有题已作答' : `${unanswered} 题未作答（提交前必须完成）`}
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
