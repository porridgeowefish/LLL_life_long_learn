import { useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { ConfusionPanel } from '@/components/feature/explain/ConfusionPanel';
import { ExplainReader } from '@/components/feature/explain/ExplainReader';
import { ExtendPage } from '@/components/feature/extend/ExtendPage';
import { IntroPage } from '@/components/feature/intro/IntroPage';
import { PracticeFlow } from '@/components/feature/practice/PracticeFlow';
import { SummaryPage } from '@/components/feature/summary/SummaryPage';
import { ZoneTimeline } from '@/components/feature/project/ZoneTimeline';
import { useProjectStore } from '@/store/slices/project';
import type { ProjectState, ZoneName } from '@/types/domain';

import s from '@/pages/ProjectPage.module.css';

interface LegacyProjectViewProps {
  project: ProjectState;
}

export function LegacyProjectView({ project }: LegacyProjectViewProps) {
  const navigate = useNavigate();
  const { zone: zoneParam } = useParams<{ zone?: string }>();
  const storedZone = useProjectStore((state) => state.currentZone);
  const setZone = useProjectStore((state) => state.setZone);
  const zone = ((zoneParam as ZoneName) || storedZone || project.activeZone || 'Explain') as ZoneName;

  useEffect(() => {
    if (zone !== storedZone) setZone(zone);
  }, [setZone, storedZone, zone]);

  const content = (() => {
    switch (zone) {
      case 'Intro':
        return <IntroPage projectSlug={project.slug} />;
      case 'Practice':
        return <PracticeFlow projectSlug={project.slug} />;
      case 'Extend':
        return <ExtendPage projectSlug={project.slug} projectTitle={project.title} />;
      case 'Summary':
        return <SummaryPage projectSlug={project.slug} />;
      case 'Explain':
      default:
        return (
          <div className={s.zoneLayout}>
            <div className={s.zoneMain}><ExplainReader projectSlug={project.slug} /></div>
            <div className={s.zoneSide}><ConfusionPanel projectSlug={project.slug} /></div>
          </div>
        );
    }
  })();

  return (
    <div className={s.host}>
      <aside className={s.aside}>
        <div className={s.asideHead}>
          <div className={s.asideHeadBody}>
            <h3 className={s.projTitle}>{project.title}</h3>
            <code className={s.projSlug}>{project.slug}</code>
          </div>
        </div>
        <ZoneTimeline
          currentZone={zone}
          hasOutput={new Set(project.generatedZones ?? [])}
          onSelect={(nextZone) => navigate(`/project/${project.slug}/${nextZone}`)}
        />
      </aside>
      <main className={s.main}>
        <header className={s.header}>
          <h1 className={s.title}>{project.title}</h1>
          <p className={s.migrationWarning} role="status">
            此学习单元尚未完成新版迁移，当前以旧版只读入口展示；原始文件没有被删除。
          </p>
        </header>
        <div className={s.scroll}>{content}</div>
      </main>
    </div>
  );
}
