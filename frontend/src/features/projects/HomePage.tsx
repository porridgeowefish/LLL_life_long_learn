import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { useHealth } from '@/app/api/health';
import { useProjects } from '@/features/projects/api/projects';
import { useUiStore } from '@/shared/store/slices/ui';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';
import { EmptyState } from '@/shared/primitive/EmptyState';
import { ProjectCard } from '@/features/projects/components/ProjectCard';
import { LearningRhythm } from '@/features/projects/components/LearningRhythm';
import { CreateProjectModal } from '@/features/projects/components/CreateProjectModal';

import s from './HomePage.module.css';

export function HomePage() {
  const navigate = useNavigate();
  const modalOpen = useUiStore((s) => s.createProjectModalOpen);
  const openModal = useUiStore((s) => s.openCreateProjectModal);
  const closeModal = useUiStore((s) => s.closeCreateProjectModal);

  const { data: health } = useHealth();
  const { data: projectsData, isLoading: projectsLoading } = useProjects();

  const projects = (projectsData?.projects ?? []).filter(project => project.projectType === 'discipline-map');
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

        {/* Projects grid */}
        <section>
          <h2 className={s.sectionTitle}>学习项目</h2>
          {projectsLoading ? (
            <div className={s.loading}>加载中…</div>
          ) : projects.length === 0 ? (
            <EmptyState
              title="还没有学科地图"
              description="创建学科地图来梳理一个领域。对话项目可从左侧栏进入。"
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
