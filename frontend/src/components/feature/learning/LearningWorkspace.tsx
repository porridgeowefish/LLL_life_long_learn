import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { NavLink, useNavigate, useParams } from 'react-router-dom';

import type { ProjectState } from '@/types/domain';
import { TeacherView } from './TeacherView';
import { AssetsView } from './AssetsView';
import { SourcesView } from './SourcesView';
import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { learningWorkspaceKeys } from '@/api/learningWorkspace';

import s from './LearningWorkspace.module.css';

const tabs = [
  { key: 'teacher', label: '教师' },
  { key: 'assets', label: '资产' },
  { key: 'sources', label: '资料' },
] as const;

export function LearningWorkspace({ project }: { project: ProjectState }) {
  const { zone } = useParams<{ zone?: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const active = tabs.some((tab) => tab.key === zone) ? zone! : 'teacher';

  useEffect(() => {
    if (zone !== active) navigate(`/project/${project.slug}/${active}`, { replace: true });
  }, [active, navigate, project.slug, zone]);

  useEffect(() => {
    const forProject = (refresh: () => void) => (payload: unknown) => {
      if ((payload as { projectSlug?: string })?.projectSlug === project.slug) {
        refresh();
      }
    };
    const unsubscribers = [
      subscribeToSSE(SSE_EVENTS.assistantTaskUpdated, forProject(() => { void queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.tasks(project.slug) }); })),
      subscribeToSSE(SSE_EVENTS.generatedArtifactUpdated, forProject(() => { void queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.generated(project.slug) }); })),
      subscribeToSSE(SSE_EVENTS.learningAssetUpdated, forProject(() => { void queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.assets(project.slug) }); })),
      subscribeToSSE(SSE_EVENTS.sourceUpdated, forProject(() => { void queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.sources(project.slug) }); })),
      subscribeToSSE(SSE_EVENTS.annotationUpdated, forProject(() => { void queryClient.invalidateQueries({ queryKey: learningWorkspaceKeys.annotations(project.slug) }); })),
    ];
    return () => unsubscribers.forEach((unsubscribe) => unsubscribe());
  }, [project.slug, queryClient]);

  return (
    <div className={s.workspace}>
      <header className={s.header}>
        <div className={s.identity}>
          <h1>{project.title}</h1>
          <span>学习单元</span>
        </div>
        <nav className={s.tabs} aria-label="学习单元">
          {tabs.map((tab) => (
            <NavLink key={tab.key} to={`/project/${project.slug}/${tab.key}`} className={({ isActive }) => isActive ? s.tabActive : s.tab}>
              {tab.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className={s.content}>
        {active === 'teacher' && <TeacherView slug={project.slug} title={project.title} />}
        {active === 'assets' && <AssetsView slug={project.slug} />}
        {active === 'sources' && <SourcesView slug={project.slug} />}
      </main>
    </div>
  );
}
