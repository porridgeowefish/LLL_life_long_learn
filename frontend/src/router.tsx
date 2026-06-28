import { lazy, Suspense } from 'react';
import { Navigate, RouteObject, useRoutes } from 'react-router-dom';

import { AppShell } from './components/layout/AppShell';

// Routes are lazy-loaded so each page lands in its own chunk. AppShell
// wraps every page; it owns the topbar, sidebar, SSE connection, and
// the terminal drawer.
const HomePage = lazy(() => import('./pages/HomePage').then((m) => ({ default: m.HomePage })));
const ProjectPage = lazy(() => import('./pages/ProjectPage').then((m) => ({ default: m.ProjectPage })));
const AgentsPage = lazy(() => import('./pages/AgentsPage').then((m) => ({ default: m.AgentsPage })));
const MemoryPage = lazy(() => import('./pages/MemoryPage').then((m) => ({ default: m.MemoryPage })));
const SettingsPage = lazy(() => import('./pages/SettingsPage').then((m) => ({ default: m.SettingsPage })));

function PageFallback() {
  return <div style={{ padding: 24, color: 'var(--muted)' }}>加载中…</div>;
}

function withShell(el: React.ReactNode) {
  return (
    <Suspense fallback={<PageFallback />}>{el}</Suspense>
  );
}

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <AppShell />,
    children: [
      {
        index: true,
        element: withShell(<HomePage />),
      },
      {
        path: 'project/:id/:zone?',
        element: withShell(<ProjectPage />),
      },
      {
        path: 'agents',
        element: withShell(<AgentsPage />),
      },
      {
        path: 'memory',
        element: withShell(<MemoryPage />),
      },
      {
        path: 'settings',
        element: withShell(<SettingsPage />),
      },
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
    ],
  },
];

// AppRoutes — helper component so App.tsx stays boring. useRoutes needs
// to be called from inside a <BrowserRouter>, which is mounted in main.
export function AppRoutes() {
  return useRoutes(routes);
}
