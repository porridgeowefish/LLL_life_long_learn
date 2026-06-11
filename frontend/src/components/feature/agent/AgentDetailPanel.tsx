import type { Agent } from '@/types/domain';
import { Card } from '@/components/primitive/Card';
import { Tag } from '@/components/primitive/Tag';

import s from './AgentDetailPanel.module.css';

interface AgentDetailPanelProps {
  agent: Agent;
}

export function AgentDetailPanel({ agent }: AgentDetailPanelProps) {
  return (
    <div className={s.host}>
      <header className={s.header}>
        <span className={s.icon}>{agent.icon}</span>
        <div className={s.headBody}>
          <h3 className={s.name}>{agent.name}</h3>
          <p className={s.desc}>{agent.description ?? '—'}</p>
        </div>
      </header>

      <Card variant="flat" className={s.section}>
        <h5 className={s.sectionTitle}>用户故事</h5>
        <p className={s.userStory}>{agent.userStory}</p>
      </Card>

      <Card variant="flat" className={s.section}>
        <h5 className={s.sectionTitle}>允许阶段</h5>
        <div className={s.tagRow}>
          {agent.allowedZones.map((z) => (
            <Tag key={z} tone="accent">
              {z}
            </Tag>
          ))}
        </div>
      </Card>

      <Card variant="flat" className={s.section}>
        <h5 className={s.sectionTitle}>Required Primitives</h5>
        {agent.primitives.required && agent.primitives.required.length > 0 ? (
          <ul className={s.list}>
            {agent.primitives.required.map((p) => (
              <li key={p}>
                <code className={s.code}>{p}</code>
              </li>
            ))}
          </ul>
        ) : (
          <span className={s.muted}>（无）</span>
        )}
      </Card>

      {agent.primitives.optional && agent.primitives.optional.length > 0 && (
        <Card variant="flat" className={s.section}>
          <h5 className={s.sectionTitle}>Optional Primitives</h5>
          <ul className={s.list}>
            {agent.primitives.optional.map((p) => (
              <li key={p}>
                <code className={s.code}>{p}</code>
              </li>
            ))}
          </ul>
        </Card>
      )}

      <Card variant="flat" className={s.section}>
        <h5 className={s.sectionTitle}>输出目标</h5>
        <ul className={s.list}>
          {agent.defaultOutputTargets.map((t) => (
            <li key={`${t.zone}/${t.filename}`}>
              <code className={s.code}>
                {t.zone.toLowerCase()}/{t.filename}
              </code>
            </li>
          ))}
        </ul>
      </Card>

      {agent.charterText && (
        <Card variant="flat" className={s.section}>
          <h5 className={s.sectionTitle}>章程全文</h5>
          <pre className={s.charter}>{agent.charterText}</pre>
        </Card>
      )}
    </div>
  );
}
