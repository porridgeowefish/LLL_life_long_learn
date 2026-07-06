import { Icon } from '@/components/primitive/Icon';

import s from './RunProgressBar.module.css';

export interface RunProgressBarProps {
  active: boolean;
  activity: string | null;
  onDismiss?: () => void;
}

/**
 * Indeterminate run-progress indicator. Shown while a generation is producing
 * artifacts. Determinate per-page/phase progress and reliable completion land in
 * Phase C (Claude Code hooks); for now the bar is indeterminate with the latest
 * activity text. A dismiss (×) lets the learner clear a lingering bar in Phase A
 * (sessions don't reach "completed" until hooks are wired).
 */
export function RunProgressBar({ active, activity, onDismiss }: RunProgressBarProps) {
  if (!active) return null;
  return (
    <div className={s.bar} role="status" aria-live="polite">
      <div className={s.track} aria-hidden="true">
        <div className={s.fill} />
      </div>
      <span className={s.label}>{activity ?? '运行中…'}</span>
      {onDismiss && (
        <button
          type="button"
          className={s.dismiss}
          onClick={onDismiss}
          title="收起"
          aria-label="收起进度"
        >
          <Icon name="x" size={12} />
        </button>
      )}
    </div>
  );
}
