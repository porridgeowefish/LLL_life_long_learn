import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { useDisciplineOverview, useGenerateDisciplineOverview } from '@/api/projects';
import { useFolders, useSaveFolders } from '@/api/folders';
import { Button } from '@/components/primitive/Button';
import { EmptyState } from '@/components/primitive/EmptyState';
import { Icon } from '@/components/primitive/Icon';
import { MarkdownView } from '@/components/primitive/MarkdownView';
import { assignProject, type FolderLayout } from '@/lib/folders';
import { useSessionStore } from '@/store/slices/session';
import { CreateProjectModal } from './CreateProjectModal';

import s from './DisciplineOverview.module.css';

interface DisciplineOverviewProps {
  slug: string;
  title: string;
}

export function extractDisciplineTopics(markdown: string): string[] {
  return extractDisciplineOutline(markdown)
    .filter((item) => item.actionable)
    .map((item) => item.title);
}

export interface DisciplineOutlineItem {
  id: string;
  level: 2 | 3;
  number: string;
  title: string;
  actionable: boolean;
}

export function extractDisciplineOutline(markdown: string): DisciplineOutlineItem[] {
  const outline: DisciplineOutlineItem[] = [];
  let section = 0;
  let subsection = 0;
  for (const rawLine of markdown.split(/\r?\n/)) {
    const line = rawLine.trim();
    const h2 = /^##\s+(.+)$/.exec(line);
    if (h2) {
      section += 1;
      subsection = 0;
      const title = cleanHeading(h2[1]);
      outline.push({
        id: `outline-${outline.length + 1}`,
        level: 2,
        number: String(section),
        title,
        actionable: false,
      });
      continue;
    }
    const h3 = /^###\s+(.+)$/.exec(line);
    if (!h3) continue;
    subsection += 1;
    const title = cleanHeading(h3[1]);
    outline.push({
      id: `outline-${outline.length + 1}`,
      level: 3,
      number: `${section}.${subsection}`,
      title,
      actionable: true,
    });
  }
  return outline;
}

function cleanHeading(value: string): string {
  return value.replace(/[*_`]/g, '').trim();
}

export function DisciplineOverview({ slug, title }: DisciplineOverviewProps) {
  const navigate = useNavigate();
  const setActiveSession = useSessionStore((state) => state.setActiveSession);
  const overview = useDisciplineOverview(slug);
  const generate = useGenerateDisciplineOverview(slug);
  const folders = useFolders();
  const saveFolders = useSaveFolders();
  const [createOpen, setCreateOpen] = useState(false);
  const [initialTitle, setInitialTitle] = useState('');
  const [launchNotice, setLaunchNotice] = useState('');
  const articleRef = useRef<HTMLElement>(null);
  const documentBodyRef = useRef<HTMLDivElement>(null);
  const content = overview.data?.content ?? '';
  const outline = useMemo(() => extractDisciplineOutline(content), [content]);
  const overviewPending = overview.data?.content.includes('尚未生成') ?? true;

  const startDeepDive = (topic = '') => {
    setInitialTitle(topic);
    setCreateOpen(true);
  };

  const launchOverviewAgent = async () => {
    setLaunchNotice('');
    try {
      const result = await generate.mutateAsync();
      setActiveSession(result.session.id);
      setLaunchNotice('百科 Agent CLI 已在可见终端启动；它写入学科总览后，本页会自动刷新。');
    } catch {
      // Mutation error is rendered beside the trigger.
    }
  };

  useEffect(() => {
    const host = documentBodyRef.current;
    if (!host || outline.length === 0) return;
    const headings = Array.from(host.querySelectorAll<HTMLHeadingElement>('h2, h3'));
    const cleanups: Array<() => void> = [];
    headings.forEach((heading, index) => {
      const item = outline[index];
      if (!item) return;
      heading.dataset.outlineId = item.id;
      if (!item.actionable) return;
      heading.classList.add(s.actionableHeading);
      const button = document.createElement('button');
      button.type = 'button';
      button.className = s.inlineDeepDive;
      button.textContent = '深入学习';
      button.setAttribute('aria-label', `深入学习：${item.title}`);
      const handleClick = () => startDeepDive(item.title);
      button.addEventListener('click', handleClick);
      heading.appendChild(button);
      cleanups.push(() => {
        button.removeEventListener('click', handleClick);
        button.remove();
        heading.classList.remove(s.actionableHeading);
        delete heading.dataset.outlineId;
      });
    });
    return () => cleanups.forEach((cleanup) => cleanup());
  }, [outline]);

  const scrollToOutline = (id: string) => {
    articleRef.current
      ?.querySelector<HTMLElement>(`[data-outline-id="${id}"]`)
      ?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <div className={s.host}>
      <header className={s.header}>
        <div>
          <h1>{title}</h1>
          <p>一个统一入口，帮助你看清学科边界、主要领域及可选择的学习路线。</p>
        </div>
        <div className={s.actionStack}>
          <div className={s.actions}>
            <Button variant="outline" onClick={() => void launchOverviewAgent()} loading={generate.isPending}>
              {generate.isPending
                ? '正在启动 Agent CLI'
                : overviewPending
                  ? '调用百科 Agent'
                  : '重新调用百科 Agent'}
            </Button>
          </div>
          {generate.isPending && (
            <div className={s.generationStatus} role="status">
              <Icon name="terminal" size={14} />
              正在创建会话并打开所选 Agent CLI…
            </div>
          )}
          {launchNotice && !generate.isPending && (
            <div className={s.generationStatus} role="status">
              <Icon name="terminal" size={14} />
              {launchNotice}
            </div>
          )}
          {generate.error && (
            <div className={s.generationError} role="alert">
              生成失败：{(generate.error as Error).message}
            </div>
          )}
        </div>
      </header>

      <main className={s.scroll}>
        {overview.isLoading ? (
          <div className={s.loading}>正在读取学科总览…</div>
        ) : overview.error ? (
          <EmptyState title="暂时无法读取学科总览" description={(overview.error as Error).message} />
        ) : overviewPending ? (
          <section className={s.emptyOverview}>
            <Icon name="book" size={28} />
            <h2>还没有学科总览</h2>
            <p>调用百科智能体后，这里会成为该学科统一的总览入口。</p>
            <Button
              variant="primary"
              onClick={() => void launchOverviewAgent()}
              loading={generate.isPending}
            >
              {generate.isPending ? '正在启动 Agent CLI…' : '调用百科 Agent'}
            </Button>
          </section>
        ) : (
          <article ref={articleRef} className={s.paper}>
            {outline.length > 0 && (
              <nav className={s.tableOfContents} aria-label="学科总览目录">
                <h2>目录</h2>
                <ol>
                  {outline.map((item) => (
                    <li key={item.id} className={item.level === 3 ? s.tocSubsection : undefined}>
                      <button type="button" onClick={() => scrollToOutline(item.id)}>
                        <span className={s.tocNumber}>{item.number}</span>
                        <span>{item.title}</span>
                        <i aria-hidden="true" />
                      </button>
                    </li>
                  ))}
                </ol>
              </nav>
            )}
            <div ref={documentBodyRef} className={s.documentBody}>
              <MarkdownView source={content} className={s.markdown} />
            </div>
          </article>
        )}
      </main>

      <CreateProjectModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        fromDisciplineMap
        fixedProjectType="system-learning"
        initialTitle={initialTitle}
        onCreated={(projectSlug) => {
          const layout: FolderLayout = folders.data ?? { folders: [] };
          const mapFolder = layout.folders.find((folder) => folder.mapProjectSlug === slug);
          if (!mapFolder) {
            navigate(`/project/${projectSlug}/Intro`);
            return;
          }
          void saveFolders
            .mutateAsync(assignProject(layout, projectSlug, mapFolder.id))
            .finally(() => navigate(`/project/${projectSlug}/Intro`));
        }}
      />
    </div>
  );
}
