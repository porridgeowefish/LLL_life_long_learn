import { useEffect } from 'react';
import { useParams } from 'react-router-dom';

import { useProject } from '@/features/projects/api/projects';
import { useHealth } from '@/app/api/health';
import { DisciplineOverview } from '@/features/projects/components/DisciplineOverview';
import { LearningWorkspace } from '@/features/learning';
import { LegacyProjectView } from '@/features/projects/components/LegacyProjectView';
import { EmptyState } from '@/shared/primitive/EmptyState';
import { useProjectStore } from '@/shared/store/slices/project';

import s from './ProjectPage.module.css';

export function ProjectPage() {
  const { id } = useParams<{ id: string }>();
  const slug = id ?? '';
  const { data, isLoading, error } = useProject(slug);
  const health = useHealth();
  const selectProject = useProjectStore((state) => state.selectProject);

  useEffect(() => { if (slug) selectProject(slug); }, [selectProject, slug]);

  if (!slug) return <div className={s.host}><EmptyState title="未指定学习单元" description="回到主页选择一个学习单元。" /></div>;
  if (isLoading || health.isLoading) return <div className={s.host}>加载中…</div>;
  if (error || !data?.project) return <div className={s.host}><EmptyState title={`学习单元「${slug}」不存在`} description={(error as Error)?.message ?? '请从主页选择一个已有项目。'} /></div>;

  if (data.project.projectType === 'discipline-map') return <DisciplineOverview slug={slug} title={data.project.title} />;
  if (health.data?.learningWorkspace?.failedProjects?.includes(slug)) {
    return <LegacyProjectView project={data.project} />;
  }
  return <LearningWorkspace project={data.project} />;
}
