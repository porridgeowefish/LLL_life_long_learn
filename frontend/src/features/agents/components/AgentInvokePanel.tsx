import { useEffect, useState } from 'react';

import { useAgents, useInvokeAgent, useResumeExplainSession } from '@/features/agents/api/agents';
import { useSessionStore } from '@/shared/store/slices/session';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';
import { PERMISSION_MODES } from '@/shared/lib/constants';
import type { ZoneName } from '@/shared/types/domain';

import s from './AgentInvokePanel.module.css';

interface AgentInvokePanelProps {
  slug: string;
  zone: ZoneName;
  sourceRefs?: string[];
  onInvoked?: () => void;
}

// AgentInvokePanel — horizontal invoke control rendered in the page
// header (top of the main column) for zones without their own entry
// (Intro/Explain/Extend/Summary). Practice uses its own GenerationEntry.
// Layout (left to right): agent select | optional guidance toggle/input | invoke button.
// Permission mode is intentionally not exposed; agent launches use Claude auto
// mode by default.
export function AgentInvokePanel({ slug, zone, sourceRefs, onInvoked }: AgentInvokePanelProps) {
  const { data: agentsData } = useAgents();
  const invoke = useInvokeAgent();
  const resumeExplain = useResumeExplainSession();
  const setActiveSession = useSessionStore((s) => s.setActiveSession);

  const compatibleAgents = (agentsData?.agents ?? []).filter((a) =>
    a.allowedZones.includes(zone),
  );

  const [selectedAgentId, setSelectedAgentId] = useState<string>(
    compatibleAgents[0]?.id ?? '',
  );
  const [intent, setIntent] = useState('');
  const [showIntentInput, setShowIntentInput] = useState(false);
  const [justInvoked, setJustInvoked] = useState(false);
  const [justResumed, setJustResumed] = useState(false);

  useEffect(() => {
    if (compatibleAgents.length === 0) return;
    const stillCompatible = compatibleAgents.find((a) => a.id === selectedAgentId);
    if (!stillCompatible) {
      setSelectedAgentId(compatibleAgents[0].id);
    }
  }, [compatibleAgents, selectedAgentId]);

  const handleInvoke = async () => {
    if (!selectedAgentId) return;
    const res = await invoke.mutateAsync({
      agentId: selectedAgentId,
      payload: {
        projectId: slug,
        zone,
        intent: intent.trim() || undefined,
        permissionMode: PERMISSION_MODES.auto,
        sourceRefs,
      },
    });
    setActiveSession(res.session.id);
    setJustInvoked(true);
    onInvoked?.();
    window.setTimeout(() => setJustInvoked(false), 6000);
  };

  const handleResumeExplain = async () => {
    const res = await resumeExplain.mutateAsync(slug);
    if (res.session?.id) {
      setActiveSession(res.session.id);
    }
    setJustResumed(true);
    window.setTimeout(() => setJustResumed(false), 6000);
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

      <button
        type="button"
        className={s.intentToggle}
        onClick={() => setShowIntentInput((v) => !v)}
        aria-expanded={showIntentInput}
      >
        {showIntentInput ? '收起补充说明' : '补充说明（可选）'}
      </button>

      {showIntentInput && (
        <input
          type="text"
          className={s.intentInput}
          placeholder={`补充告诉 ${zone} Agent 你的目标、约束或偏好…`}
          value={intent}
          onChange={(e) => setIntent(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              handleInvoke();
            }
          }}
          aria-label="补充说明"
        />
      )}

      {(invoke.error || resumeExplain.error) && (
        <span
          className={s.errorInline}
          title={((invoke.error || resumeExplain.error) as Error).message}
        >
          操作失败
        </span>
      )}
      {justInvoked && (
        <span className={s.invokedInline}>
          <Icon name="check" size={12} /> 已启动
        </span>
      )}
      {justResumed && (
        <span className={s.resumedInline}>
          <Icon name="terminal" size={12} /> 已打开继续会话
        </span>
      )}

      <Button
        variant="primary"
        onClick={handleInvoke}
        loading={invoke.isPending}
        disabled={!selectedAgentId}
        className={s.invokeBtn}
      >
        调用
      </Button>
      {zone === 'Explain' && (
        <Button
          variant="outline"
          onClick={handleResumeExplain}
          loading={resumeExplain.isPending}
          className={s.resumeBtn}
          iconLeft={<Icon name="terminal" size={14} />}
          title="打开 Claude Code 最近一次会话，不重新生成"
        >
          继续上次会话
        </Button>
      )}
    </div>
  );
}
