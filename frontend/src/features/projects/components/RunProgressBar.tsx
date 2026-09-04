import { Icon } from '@/shared/primitive/Icon';

import s from './RunProgressBar.module.css';

export interface RunProgressBarProps {
  active: boolean;
  activity: string | null;
  /** Pages written so far (Phase C run-progress). 0 when not reported. */
  pagesDone?: number;
  /** Total pages planned (Phase C run-progress). 0 → indeterminate mode. */
  pagesPlanned?: number;
  onDismiss?: () => void;
}

/**
 * Run progress indicator.
 *
 * - Indeterminate (Phase A fallback): when pagesPlanned is 0/unknown, show the
 *   pulsing bar + the latest activity text. Non-Claude runtimes live here.
 * - Determinate (Phase C): when pagesPlanned > 0, render a filled "N / M 页"
 *   bar driven by the run-progress events Claude Code hooks report.
 *
 * A dismiss (×) lets the learner clear a lingering bar.
 */
export function RunProgressBar({ active, activity, pagesDone = 0, pagesPlanned = 0, onDismiss }: RunProgressBarProps) {
  if (!active) return null;
  const determinate = pagesPlanned > 0;
  const pct = determinate ? Math.min(100, Math.round((pagesDone / pagesPlanned) * 100)) : 0;
  const label = determinate ? `${pagesDone} / ${pagesPlanned} 页` : activity ?? '运行中…';
  return (
    <div className={s.bar} role="status" aria-live="polite">
      <div className={s.track} aria-hidden="true">
        {determinate ? (
          <div className={s.fillDeterminate} style={{ width: `${pct}%` }} />
        ) : (
          <div className={s.fill} />
        )}
      </div>
      <span className={s.label}>{label}</span>
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
