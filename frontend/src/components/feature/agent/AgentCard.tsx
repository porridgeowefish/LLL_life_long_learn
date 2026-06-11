import clsx from 'clsx';

import type { Agent } from '@/types/domain';
import { Tag } from '@/components/primitive/Tag';

import s from './AgentCard.module.css';

interface AgentCardProps {
  agent: Agent;
  active?: boolean;
  onClick?: () => void;
  className?: string;
}

export function AgentCard({ agent, active, onClick, className }: AgentCardProps) {
  return (
    <div
      className={clsx(s.card, active && s.active, className)}
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onClick?.();
        }
      }}
    >
      <div className={s.head}>
        <span className={s.icon}>{agent.icon}</span>
        <h4 className={s.name}>{agent.name}</h4>
      </div>
      <p className={s.desc}>{agent.description ?? '—'}</p>
      <div className={s.zones}>
        {agent.allowedZones.map((z) => (
          <Tag key={z} tone="accent">
            {z}
          </Tag>
        ))}
      </div>
    </div>
  );
}
