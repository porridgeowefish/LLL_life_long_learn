import { useState } from 'react';
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
import { LearningRhythm } from '@/components/feature/activity/LearningRhythm';
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
  const runtime = health?.agentRuntime?.runtime;

  return (
    <div className={s.host}>
      <div className={s.scroll}>
        <header className={s.header}>
          <div>
            <h1 className={s.title}>学习主页</h1>
            <p className={s.subtitle}>
              本地学习工作台 · 当前工作区 <code>{health?.workspace ?? '…'}</code>
            </p>
          </div>
          <Button variant="primary" iconLeft={<Icon name="plus" size={14} />} onClick={openModal}>
            新建项目
          </Button>
        </header>

        <OnboardingBanner runtimeName={runtime?.name} runtimeReady={runtime?.available ?? false} />

        <LearningRhythm projects={projects} />

        {/* Active sessions hero */}
        {active?.sessions && active.sessions.length > 0 && (
          <section className={s.active}>
            {active.sessions.map((sess) => (
              <div key={sess.id} className={s.activeTile}>
                <span className={s.activeIcon}>
                  <Icon name="bot" size={16} />
                </span>
                <div className={s.activeBody}>
                  <h4>{sess.projectSlug} · {sess.agentId}</h4>
                  <p>
                    <Tag tone="orange">{sess.state}</Tag>{' '}
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
              description="创建学科地图来了解一个领域，或创建系统学习项目进入完整学习流程。"
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
                  onClick={() => navigate(`/project/${sess.projectSlug}/${sess.zoneName}`)}
                >
                  <span className={s.recentIcon}>
                    <Icon name="clock" size={16} />
                  </span>
                  <div className={s.recentBody}>
                    <h4>{sess.projectSlug} · {sess.agentId}</h4>
                    <p>{formatRelativeTime(sess.createdAt)} · {sess.zoneName}</p>
                  </div>
                  <Tag tone={sess.state === 'completed' ? 'accent' : 'muted'}>
                    {sess.state}
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

interface OnboardingBannerProps {
  runtimeName?: string;
  runtimeReady: boolean;
}

function OnboardingBanner({ runtimeName, runtimeReady }: OnboardingBannerProps) {
  const [dismissed, setDismissed] = useState(() => localStorage.getItem('lll-onboarding-dismissed') === '1');
  if (dismissed) return null;
  return (
    <section className={s.onboarding}>
      <div className={s.onboardingHead}>
        <span className={s.onboardingIcon}>
          <img src="/img/lychee-teacher.png" alt="" aria-hidden="true" />
        </span>
        <div>
          <h2>开始之前</h2>
          <p>
            当前运行时：{runtimeName ?? 'Agent CLI'} · {runtimeReady ? '已就绪' : '未就绪，先去统一配置确认安装'}
          </p>
        </div>
      </div>
      <div className={s.onboardingSteps}>
        <span>1. 在统一配置选择 CLI</span>
        <span>2. 创建学习项目</span>
        <span>3. 进入项目调用学习 Agent</span>
      </div>
      <Button
        variant="outline"
        iconLeft={<Icon name="x" size={13} />}
        onClick={() => {
          localStorage.setItem('lll-onboarding-dismissed', '1');
          setDismissed(true);
        }}
      >
        知道了
      </Button>
    </section>
  );
}
