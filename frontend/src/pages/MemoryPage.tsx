import { useProjects } from '@/api/projects';
import { Card } from '@/components/primitive/Card';
import { EmptyState } from '@/components/primitive/EmptyState';
import { MemoryEditor } from '@/components/feature/memory/MemoryEditor';

import s from './MemoryPage.module.css';

export function MemoryPage() {
  const { data, isLoading } = useProjects();
  const projects = data?.projects ?? [];

  return (
    <div className={s.host}>
      <div className={s.scroll}>
        <header className={s.header}>
          <h1 className={s.title}>记忆系统</h1>
          <p className={s.subtitle}>
            项目级记忆文件。Learner-owned：智能体只读不写，修改请直接编辑下方文本框并保存。
          </p>
        </header>

        {isLoading ? (
          <div className={s.loading}>加载中…</div>
        ) : projects.length === 0 ? (
          <EmptyState
            title="尚无项目记忆"
            description="先到「项目」页创建一个学习项目，记忆文件会自动出现。"
          />
        ) : (
          <div className={s.list}>
            {projects.map((p) => (
              <MemoryEditor key={p.slug} slug={p.slug} title={p.title} />
            ))}
          </div>
        )}

        <Card variant="flat" className={s.rules}>
          <h3 className={s.rulesTitle}>记忆规则</h3>
          <ul className={s.rulesList}>
            <li>记忆<strong>引导个性化</strong>，绝不静默覆盖学习者控制权</li>
            <li>总结文件是<strong>学习者所有</strong>——智能体可提议修改，不可覆盖</li>
            <li>记忆写入是<strong>受控且可审查</strong>的</li>
            <li>项目记忆仅在<strong>有意义的运行后</strong>更新</li>
            <li>学习者记忆<strong>跨项目</strong>随时间演进</li>
          </ul>
        </Card>
      </div>
    </div>
  );
}
