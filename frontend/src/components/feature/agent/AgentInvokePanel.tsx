import { useEffect, useState } from 'react';

import { useAgents, useInvokeAgent } from '@/api/agents';
import { useSessionStore } from '@/store/slices/session';
import { Button } from '@/components/primitive/Button';
import { Icon } from '@/components/primitive/Icon';
import { PERMISSION_MODES } from '@/lib/constants';
import type { ZoneName } from '@/types/domain';

import s from './AgentInvokePanel.module.css';

interface AgentInvokePanelProps {
  slug: string;
  zone: ZoneName;
  sourceRefs?: string[];
  onInvoked?: () => void;
}

// AgentInvokePanel — now a horizontal bottom-bar, not a side card.
// Renders inline inside the sticky invoke bar at the bottom of the main
// column. Layout (left to right): agent select | intent input | invoke button.
// Permission mode is intentionally NOT exposed — the runtime default
// (acceptEdits) is what 99% of learning sessions want.
export function AgentInvokePanel({ slug, zone, sourceRefs, onInvoked }: AgentInvokePanelProps) {
  const { data: agentsData } = useAgents();
  const invoke = useInvokeAgent();
  const setActiveSession = useSessionStore((s) => s.setActiveSession);

  const compatibleAgents = (agentsData?.agents ?? []).filter((a) =>
    a.allowedZones.includes(zone),
  );

  const [selectedAgentId, setSelectedAgentId] = useState<string>(
    compatibleAgents[0]?.id ?? '',
  );
  const [intent, setIntent] = useState('');
  const [justInvoked, setJustInvoked] = useState(false);

  useEffect(() => {
    if (compatibleAgents.length === 0) return;
    const stillCompatible = compatibleAgents.find((a) => a.id === selectedAgentId);
    if (!stillCompatible) {
      setSelectedAgentId(compatibleAgents[0].id);
    }
  }, [compatibleAgents, selectedAgentId]);

  const handleInvoke = async () => {
    if (!selectedAgentId || !intent.trim()) return;
    const res = await invoke.mutateAsync({
      agentId: selectedAgentId,
      payload: {
        projectId: slug,
        zone,
        intent: intent.trim(),
        permissionMode: PERMISSION_MODES.acceptEdits,
        sourceRefs,
      },
    });
    setActiveSession(res.session.id);
    setJustInvoked(true);
    onInvoked?.();
    window.setTimeout(() => setJustInvoked(false), 6000);
  };

  if (compatibleAgents.length === 0) {
    return (
      <div className={`${s.host} agentInvokeHost`}>
        <p className={s.muted}>当前阶段（{zone}）没有注册兼容的智能体。</p>
      </div>
    );
  }

  return (
    <div className={`${s.host} agentInvokeHost`}>
      <select
        className={s.agentSelect}
        value={selectedAgentId}
        onChange={(e) => setSelectedAgentId(e.target.value)}
        aria-label="选择智能体"
      >
        {compatibleAgents.map((a) => (
          <option key={a.id} value={a.id}>
            {a.icon} {a.name}
          </option>
        ))}
      </select>

      <input
        type="text"
        className={s.intentInput}
        placeholder={`向 ${zone} Agent 描述你想学什么…`}
        value={intent}
        onChange={(e) => setIntent(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleInvoke();
          }
        }}
        aria-label="学习意图"
      />

      {invoke.error && (
        <span className={s.errorInline} title={(invoke.error as Error).message}>
          调用失败
        </span>
      )}
      {justInvoked && (
        <span className={s.invokedInline}>
          <Icon name="check" size={12} /> 已启动
        </span>
      )}

      <Button
        variant="primary"
        onClick={handleInvoke}
        loading={invoke.isPending}
        disabled={!intent.trim() || !selectedAgentId}
        className={s.invokeBtn}
      >
        调用
      </Button>
    </div>
  );
}
