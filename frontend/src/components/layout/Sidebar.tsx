import { Link } from 'react-router-dom';
import { useProjects } from '@/api/projects';
import { Icon, type IconName } from '@/components/primitive/Icon';

import s from './Sidebar.module.css';

const QUICK_LINKS: ReadonlyArray<{ to: string; label: string; icon: IconName }> = [
  { to: '/', label: '学习总览', icon: 'zap' },
  { to: '/agents', label: '智能体管理', icon: 'bot' },
  { to: '/memory', label: '记忆系统', icon: 'brain' },
];

export function Sidebar() {
  const { data: projectsData, isLoading } = useProjects();
  const projects = projectsData?.projects ?? [];

  return (
    <aside className={s.sidebar}>
      <div className={s.head}>
        <span>导航</span>
      </div>
      <div className={s.scroll}>
        <div className={s.section}>快捷入口</div>
        {QUICK_LINKS.map((q) => (
          <Link key={q.to} to={q.to} className={s.link}>
            <Icon name={q.icon} size={15} className={s.linkIcon} />
            {q.label}
          </Link>
        ))}

        <div className={s.section}>最近项目</div>
        {isLoading && <div className={s.muted}>加载中…</div>}
        {!isLoading && projects.length === 0 && (
          <div className={s.muted}>尚无项目，去总览创建一个 →</div>
        )}
        {projects.map((p) => (
          <Link key={p.slug} to={`/project/${p.slug}`} className={s.link}>
            <Icon name="folder" size={15} className={s.linkIcon} />
            <span className={s.title}>{p.title}</span>
          </Link>
        ))}
      </div>
      <div className={s.foot}>v0.1 · React 18 + TS</div>
    </aside>
  );
}
