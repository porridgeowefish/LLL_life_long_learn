import { useConnectionState } from '@/hooks/useConnectionState';

import s from './ConnectionBadge.module.css';

export function ConnectionBadge() {
  const state = useConnectionState();
  return (
    <span className={s.badge} style={{ color: state.color }}>
      <span className={s.dot} style={{ background: state.color }} />
      {state.label}
    </span>
  );
}
