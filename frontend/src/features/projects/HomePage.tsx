import { useRef, useState } from 'react';
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
import { LearningDemo } from '@/features/learning';

import s from './HomePage.module.css';

export function HomePage() {
  const navigate = useNavigate();
  const modalOpen = useUiStore((s) => s.createProjectModalOpen);
  const openModal = useUiStore((s) => s.openCreateProjectModal);
  const closeModal = useUiStore((s) => s.closeCreateProjectModal);
  const [demoOpen, setDemoOpen] = useState(false);
  const demoOpener = useRef<HTMLButtonElement | null>(null);
  const replayButton = useRef<HTMLButtonElement>(null);
  const openDemo = (button: HTMLButtonElement) => { demoOpener.current = button; setDemoOpen(true); };
  const [dismissed, setDismissed] = useState(() => {
    try { return localStorage.getItem('lll.onboarding.v1.dismissed') === '1'; }
    catch { return false; }
  });
  const dismissGuide = () => {
    try { localStorage.setItem('lll.onboarding.v1.dismissed', '1'); }
    catch { /* The current visit still respects dismissal without browser storage. */ }
    setDismissed(true);
    setDemoOpen(false);
  };
  const closeDemo = () => {
    dismissGuide();
    requestAnimationFrame(() => { (demoOpener.current?.isConnected ? demoOpener.current : replayButton.current)?.focus(); });
  };

  const { data: health } = useHealth();
  const { data: projectsData, isLoading: projectsLoading } = useProjects();

  const projects = (projectsData?.projects ?? []).filter(project => project.projectType === 'discipline-map');

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
          <div className={s.headerActions}>
          <Button ref={replayButton} onClick={event => openDemo(event.currentTarget)}>体验示例</Button>
          <Button variant="primary" iconLeft={<Icon name="plus" size={14} />} onClick={openModal}>
            新建项目
          </Button>
          </div>
        </header>

        {!dismissed && <section className={s.onboarding} aria-label="新用户引导">
          <div className={s.onboardingHead}>
            <span className={s.onboardingIcon}><img src="/img/lychee-teacher.png" alt="" aria-hidden="true" /></span>
            <div><h2>先跟着一个好问题，学一次</h2><p>用梯度下降体验提问、复习、练习和笔记。示例已经准备好，无需配置 API。</p></div>
          </div>
          <div className={s.onboardingExample}><strong>为什么沿负梯度更新？</strong><p>梯度指向局部增长最快的方向，反向走一小步就有下降的趋势；但走多远，还要看学习率。</p></div>
          <div className={s.onboardingActions}><Button variant="primary" onClick={event => openDemo(event.currentTarget)}>跟着示例开始</Button><Button variant="ghost" onClick={closeDemo}>跳过引导</Button></div>
        </section>}

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
      {demoOpen && <LearningDemo onClose={closeDemo} onStart={() => { dismissGuide(); openModal(); }} onConfigure={() => { dismissGuide(); navigate('/settings'); }} />}
    </div>
  );
}
