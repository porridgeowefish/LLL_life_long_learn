import { useNavigate } from 'react-router-dom';

import { useHealth } from '@/api/health';
import { useProjects } from '@/api/projects';
import { useRecentSessions, useActiveSessions } from '@/api/sessions';
import { useUiStore } from '@/store/slices/ui';
import { Button } from '@/components/primitive/Button';
import { Card } from '@/components/primitive/Card';
import { Tag } from '@/components/primitive/Tag';
import { Icon } from '@/components/primitive/Icon';
import { EmptyState } from '@/components/primitive/EmptyState';
import { ProjectCard } from '@/components/feature/project/ProjectCard';
import { CreateProjectModal } from '@/components/feature/project/CreateProjectModal';
import { formatRelativeTime } from '@/lib/format';

import s from './HomePage.module.css';

export function HomePage() {
  const navigate = useNavigate();
  const modalOpen = useUiStore((s) => s.createProjectModalOpen);
  const openModal = useUiStore((s) => s.openCreateProjectModal);
  const closeModal = useUiStore((s) => s.closeCreateProjectModal);

  const { data: health } = useHealth();
  const { data: projectsData, isLoading: projectsLoading } = useProjects();
  const { data: recent } = useRecentSessions(5);
  const { data: active } = useActiveSessions();

  const projects = projectsData?.projects ?? [];
  const stats = health?.stats;

  return (
    <div className={s.host}>
      <div className={s.scroll}>
        <header className={s.header}>
          <div>
            <h1 className={s.title}>学习总览</h1>
            <p className={s.subtitle}>
              本地学习工作台 · 当前工作区 <code>{health?.workspace ?? '…'}</code>
            </p>
          </div>
          <Button variant="primary" iconLeft={<Icon name="plus" size={14} />} onClick={openModal}>
            新建项目
          </Button>
        </header>

        {/* Stats hero */}
        <section className={s.statsRow}>
          <StatCard label="学习项目" value={stats?.projects ?? '—'} />
          <StatCard label="会话总数" value={stats?.sessions ?? '—'} />
          <StatCard label="对话轮次" value={stats?.turns ?? '—'} />
          <StatCard label="运行中" value={stats?.activeSessions ?? '—'} tone="orange" />
        </section>

        {/* Active sessions hero */}
        {active?.sessions && active.sessions.length > 0 && (
          <section className={s.active}>
            {active.sessions.map((sess) => (
              <div key={sess.id} className={s.activeTile}>
                <span className={s.activeIcon}>
                  <Icon name="bot" size={16} />
                </span>
                <div className={s.activeBody}>
                  <h4>{sess.projectId} · {sess.agentId}</h4>
                  <p>
                    <Tag tone="orange">{sess.status}</Tag>{' '}
                    <span className={s.activeMeta}>{sess.zoneName} zone</span>
                  </p>
                </div>
              </div>
            ))}
          </section>
        )}

        {/* Projects grid */}
        <section>
          <h2 className={s.sectionTitle}>学习项目</h2>
          {projectsLoading ? (
            <div className={s.loading}>加载中…</div>
          ) : projects.length === 0 ? (
            <EmptyState
              title="还没有任何学习项目"
              description="一个学习项目对应一个你想掌握的主题。在项目里你可以调用 5 个学习智能体（Intro / Explain / Practice / Extend / Summary）来完成完整学习流程。"
              action={
                <Button variant="primary" onClick={openModal}>
                  创建第一个项目
                </Button>
              }
            />
          ) : (
            <div className={s.grid}>
              {projects.map((p) => (
                <ProjectCard key={p.slug} project={p} />
              ))}
              <ProjectCard variant="create" onClick={openModal} />
            </div>
          )}
        </section>

        {/* Recent runs */}
        {recent?.sessions && recent.sessions.length > 0 && (
          <section>
            <h2 className={s.sectionTitle}>最近运行</h2>
            <div className={s.recent}>
              {recent.sessions.map((sess) => (
                <Card
                  key={sess.id}
                  variant="outlined"
                  className={s.recentCard}
                  onClick={() => navigate(`/project/${sess.projectId}`)}
                >
                  <span className={s.recentIcon}>
                    <Icon name="clock" size={16} />
                  </span>
                  <div className={s.recentBody}>
                    <h4>{sess.projectId} · {sess.agentId}</h4>
                    <p>{formatRelativeTime(sess.startedAt)} · {sess.zoneName}</p>
                  </div>
                  <Tag tone={sess.status === 'completed' ? 'accent' : 'muted'}>
                    {sess.status}
                  </Tag>
                </Card>
              ))}
            </div>
          </section>
        )}
      </div>

      <CreateProjectModal
        open={modalOpen}
        onOpenChange={(v) => (v ? openModal() : closeModal())}
        onCreated={(slug) => navigate(`/project/${slug}`)}
      />
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: number | string;
  tone?: 'neutral' | 'orange';
}

function StatCard({ label, value, tone = 'neutral' }: StatCardProps) {
  return (
    <Card variant="outlined" className={s.stat}>
      <div className={s.statLabel}>{label}</div>
      <div className={s.statValue} style={{ color: tone === 'orange' ? 'var(--orange)' : 'var(--fg)' }}>
        {value}
      </div>
    </Card>
  );
}
