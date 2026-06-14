import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { useProject } from '@/api/projects';
import { EmptyState } from '@/components/primitive/EmptyState';
import { Icon } from '@/components/primitive/Icon';
import { ZoneTimeline } from '@/components/feature/project/ZoneTimeline';
import { OutputViewer } from '@/components/feature/project/OutputViewer';
import { AgentInvokePanel } from '@/components/feature/agent/AgentInvokePanel';
import { ConfusionPanel } from '@/components/feature/explain/ConfusionPanel';
import { PracticeFlow } from '@/components/feature/practice/PracticeFlow';
import { ExtendPage } from '@/components/feature/extend/ExtendPage';
import { SummaryPage } from '@/components/feature/summary/SummaryPage';
import { useProjectStore } from '@/store/slices/project';
import { ZONE_DISPLAY, type ZoneName } from '@/types/domain';

import s from './ProjectPage.module.css';

export function ProjectPage() {
  const { id, zone: zoneParam } = useParams<{ id: string; zone?: string }>();
  const navigate = useNavigate();

  const slug = id ?? '';
  const { data, isLoading, error } = useProject(slug);
  const project = data?.project;

  const storedZone = useProjectStore((s) => s.currentZone);
  const setZone = useProjectStore((s) => s.setZone);
  const selectProject = useProjectStore((s) => s.selectProject);

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  useEffect(() => {
    if (slug) selectProject(slug);
  }, [slug, selectProject]);

  const zone: ZoneName = ((zoneParam as ZoneName) ||
    storedZone ||
    project?.activeZone ||
    'Explain') as ZoneName;

  useEffect(() => {
    if (zone && zone !== storedZone) setZone(zone);
  }, [zone, storedZone, setZone]);

  if (!slug) {
    return (
      <div className={s.host}>
        <EmptyState
          title="未指定项目"
          description="URL 中缺少项目 slug，回到总览选择一个项目。"
        />
      </div>
    );
  }

  if (isLoading) {
    return <div className={s.host}>加载中…</div>;
  }

  if (error || !project) {
    return (
      <div className={s.host}>
        <EmptyState
          title={`项目 "${slug}" 不存在`}
          description={(error as Error)?.message ?? '请从总览页选择一个已存在的项目。'}
        />
      </div>
    );
  }

  const handleZoneSelect = (z: ZoneName) => {
    navigate(`/project/${slug}/${z}`);
  };

  // Zone-specific content renderer.
  const renderZoneContent = () => {
    switch (zone) {
      case 'Explain':
        return (
          <div className={s.zoneLayout}>
            <div className={s.zoneMain}>
              <OutputViewer slug={slug} zone={zone} />
            </div>
            <div className={s.zoneSide}>
              <ConfusionPanel projectSlug={slug} />
            </div>
          </div>
        );
      case 'Practice':
        return <PracticeFlow projectSlug={slug} />;
      case 'Extend':
        return <ExtendPage projectSlug={slug} />;
      case 'Summary':
        return <SummaryPage projectSlug={slug} />;
      default:
        // Intro and fallback — just the output viewer.
        return <OutputViewer slug={slug} zone={zone} />;
    }
  };

  return (
    <div className={`${s.host} ${sidebarCollapsed ? s.hostAsideHidden : ''}`}>
      {!sidebarCollapsed && (
        <aside className={s.aside}>
          <div className={s.asideHead}>
            <div className={s.asideHeadBody}>
              <h3 className={s.projTitle}>{project.title}</h3>
              <code className={s.projSlug}>{project.slug}</code>
            </div>
            <button
              type="button"
              className={s.asideToggle}
              onClick={() => setSidebarCollapsed(true)}
              title="折叠侧栏"
              aria-label="折叠侧栏"
            >
              <Icon name="x" size={12} />
            </button>
          </div>
          <ZoneTimeline
            currentZone={zone}
            hasOutput={new Set(
              (project.lastArtifacts ?? []).map((a) => a.zoneName),
            )}
            onSelect={handleZoneSelect}
          />
        </aside>
      )}

      <main className={s.main}>
        {sidebarCollapsed && (
          <button
            type="button"
            className={s.expandAside}
            onClick={() => setSidebarCollapsed(false)}
            title="展开侧栏"
          >
            <Icon name="plus" size={12} /> 阶段
          </button>
        )}

        <header className={s.header}>
          <div className={s.headerTop}>
            <div className={s.headerBody}>
              <h1 className={s.title}>{project.title}</h1>
              <p className={s.subtitle}>
                当前阶段：<strong>{ZONE_DISPLAY[zone]}</strong>
              </p>
            </div>
            {zone !== 'Practice' && (
              <div className={s.headerActions}>
                <AgentInvokePanel slug={slug} zone={zone} />
              </div>
            )}
          </div>
        </header>

        <div className={s.scroll}>
          {renderZoneContent()}
        </div>
      </main>
    </div>
  );
}
