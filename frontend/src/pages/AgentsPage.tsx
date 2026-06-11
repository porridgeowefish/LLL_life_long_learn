import { useMemo, useState } from 'react';

import { useAgents } from '@/api/agents';
import { Card } from '@/components/primitive/Card';
import { EmptyState } from '@/components/primitive/EmptyState';
import { AgentCard } from '@/components/feature/agent/AgentCard';
import { AgentDetailPanel } from '@/components/feature/agent/AgentDetailPanel';

import s from './AgentsPage.module.css';

export function AgentsPage() {
  const { data, isLoading } = useAgents();
  const agents = data?.agents ?? [];
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const selected = useMemo(
    () => agents.find((a) => a.id === selectedId) ?? agents[0] ?? null,
    [agents, selectedId],
  );

  // Group agents by allowed zone for the sidebar category view.
  const byZone = useMemo(() => {
    const m: Record<string, typeof agents> = {};
    for (const a of agents) {
      for (const z of a.allowedZones) {
        (m[z] ||= []).push(a);
      }
    }
    return m;
  }, [agents]);

  return (
    <div className={s.host}>
      <div className={s.scroll}>
        <header className={s.header}>
          <h1 className={s.title}>智能体注册表</h1>
          <p className={s.subtitle}>
            每个智能体都有明确的用户故事、章程和阶段兼容性。点击卡片查看详情。
          </p>
        </header>

        {isLoading ? (
          <div className={s.loading}>加载中…</div>
        ) : agents.length === 0 ? (
          <EmptyState
            title="尚未注册任何智能体"
            description="智能体定义位于 agents/registry/*.json + agents/charters/*.md。重启后端让 registry 重新加载。"
          />
        ) : (
          <>
            <section>
              <h2 className={s.sectionTitle}>已注册智能体（{agents.length}）</h2>
              <div className={s.grid}>
                {agents.map((a) => (
                  <AgentCard
                    key={a.id}
                    agent={a}
                    active={selected?.id === a.id}
                    onClick={() => setSelectedId(a.id)}
                  />
                ))}
              </div>
            </section>

            {selected && (
              <section>
                <h2 className={s.sectionTitle}>详情：{selected.name}</h2>
                <Card variant="outlined" className={s.detailWrap}>
                  <AgentDetailPanel agent={selected} />
                </Card>
              </section>
            )}

            <section>
              <h2 className={s.sectionTitle}>按阶段分类</h2>
              <Card variant="flat" className={s.catList}>
                {Object.entries(byZone).map(([zone, list]) => (
                  <div key={zone} className={s.catRow}>
                    <span className={s.catName}>{zone}</span>
                    <span className={s.catAgents}>
                      {list.map((a) => a.icon).join(' ')} · {list.length} 个
                    </span>
                  </div>
                ))}
              </Card>
            </section>
          </>
        )}
      </div>
    </div>
  );
}
