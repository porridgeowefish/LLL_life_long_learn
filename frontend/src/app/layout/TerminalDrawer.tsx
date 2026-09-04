import { useSessionStore } from '@/shared/store/slices/session';
import { useUiStore } from '@/shared/store/slices/ui';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';

import s from './TerminalDrawer.module.css';

// TerminalDrawer — slides in from the right to show live SSE terminal
// output for the active session. W4 wires the open state to user actions
// (invoke / follow-up). W5 adds virtualised rendering for long streams.
export function TerminalDrawer() {
  const open = useUiStore((s) => s.drawerOpen);
  const setDrawerOpen = useUiStore((s) => s.setDrawerOpen);
  const buffer = useSessionStore((s) => s.outputBuffer);

  if (!open) return null;

  return (
    <aside className={s.drawer}>
      <header className={s.head}>
        <span className={s.title}>
          <Icon name="terminal" size={14} /> 实时输出
        </span>
        <Button size="sm" variant="ghost" onClick={() => setDrawerOpen(false)}>
          关闭
        </Button>
      </header>
      <pre className={s.body}>{buffer || '（暂无输出）'}</pre>
    </aside>
  );
}
