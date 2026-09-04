import { Outlet } from 'react-router-dom';

import { useAppearance } from '@/features/settings/appearance';
import { useSSE } from '@/app/hooks/useSSE';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';

import s from './AppShell.module.css';

// AppShell — persistent top-level layout. Mounts:
//   - SSE connection (single, app-wide EventSource via useSSE)
//   - Topbar with nav + connection badge
//   - Sidebar with quick links + recent projects
//   - Main <Outlet /> where routed pages render
//
// IMPORTANT: there is NO in-page "terminal drawer" here. Per the user's
// explicit ask, the authentic Claude execution lives in the standalone
// PowerShell window opened by the backend (claudelauncher/console_windows.go).
// The frontend does NOT recreate a fake terminal — that would be a
// double-track redundancy competing for the user's attention. The
// TerminalDrawer component file is kept only as a recoverable artifact.
export function AppShell() {
  useSSE();
  useAppearance();

  return (
    <div className={s.root}>
      <Topbar />
      <div className={s.shell}>
        <Sidebar />
        <main className={s.main}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
