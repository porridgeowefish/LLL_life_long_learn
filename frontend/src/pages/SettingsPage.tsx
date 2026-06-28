import { useMemo } from 'react';

import { useAgentRuntimeSettings, useUpdateAgentRuntime } from '@/api/settings';
import { Button } from '@/components/primitive/Button';
import { Card } from '@/components/primitive/Card';
import { Icon } from '@/components/primitive/Icon';
import { Tag } from '@/components/primitive/Tag';
import type { AgentRuntimeID, AgentRuntimeProvider } from '@/types/domain';

import s from './SettingsPage.module.css';

const ORDER: AgentRuntimeID[] = ['workbuddy', 'hermes', 'codex', 'trae', 'claude'];

const RUNTIME_LOGOS: Record<AgentRuntimeID, string> = {
  claude: '/img/agent-runtimes/claude.png',
  workbuddy: '/img/agent-runtimes/workbuddy.png',
  hermes: '/img/agent-runtimes/hermes.png',
  codex: '/img/agent-runtimes/codex.png',
  trae: '/img/agent-runtimes/trae.png',
};

export function SettingsPage() {
  const { data, isLoading, isError, error } = useAgentRuntimeSettings();
  const update = useUpdateAgentRuntime();

  const providers = useMemo(() => {
    const list = data?.providers ?? [];
    return [...list].sort((a, b) => ORDER.indexOf(a.id) - ORDER.indexOf(b.id));
  }, [data?.providers]);

  const selected = update.data?.selected ?? data?.selected;

  return (
    <div className={s.host}>
      <div className={s.scroll}>
        <header className={s.header}>
          <div>
            <h1 className={s.title}>统一配置</h1>
            <p className={s.subtitle}>选择底层 Agent CLI。模型、账号和密钥继续在各 CLI 自己的配置里管理。</p>
          </div>
          <Button
            variant="outline"
            iconLeft={<Icon name="terminal" size={14} />}
            onClick={() => window.location.reload()}
          >
            刷新探测
          </Button>
        </header>

        {isLoading && <div className={s.loading}>加载配置中…</div>}
        {isError && (
          <div className={s.error} role="alert">
            配置读取失败：{(error as Error).message}
          </div>
        )}

        <section className={s.grid}>
          {providers.map((provider) => (
            <RuntimeCard
              key={provider.id}
              provider={provider}
              selected={provider.id === selected}
              saving={update.isPending}
              onSelect={() => update.mutate(provider.id)}
            />
          ))}
        </section>

        {update.isError && (
          <div className={s.error} role="alert">
            保存失败：{(update.error as Error).message}
          </div>
        )}

        <section className={s.notes}>
          <div className={s.note}>
            <span className={s.noteIcon}><Icon name="check" size={14} /></span>
            <div>
              <h3>学习 Agent 不变</h3>
              <p>Intro / Explain / Practice / Extend / Summary 仍然负责学习流程，只是执行它们的 CLI 可切换。</p>
            </div>
          </div>
          <div className={s.note}>
            <span className={s.noteIcon}><Icon name="terminal" size={14} /></span>
            <div>
              <h3>命令名可覆盖</h3>
              <p>需要自定义路径时，在系统环境变量里设置对应的 *_BIN；平台不会要求你在页面里填写模型参数。</p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

interface RuntimeCardProps {
  provider: AgentRuntimeProvider;
  selected: boolean;
  saving: boolean;
  onSelect: () => void;
}

function RuntimeCard({ provider, selected, saving, onSelect }: RuntimeCardProps) {
  return (
    <Card variant="outlined" className={`${s.card} ${selected ? s.cardSelected : ''}`}>
      <div className={s.cardTop}>
        <div className={s.cardTitle}>
          <span className={s.cardIcon}>
            <img src={RUNTIME_LOGOS[provider.id]} alt="" aria-hidden="true" />
          </span>
          <h2>{provider.name}</h2>
        </div>
        <Tag tone={provider.available ? 'accent' : 'orange'}>
          {provider.available ? '已就绪' : '未探测到'}
        </Tag>
      </div>
      <p className={s.desc}>{provider.description}</p>
      <div className={s.meta}>
        <span>命令：<code>{provider.bin}</code></span>
        <span>覆盖：<code>{provider.binEnv}</code></span>
        <span>{provider.supportsHeadless ? '支持后台任务' : '仅交互终端'}</span>
      </div>
      <Button
        variant={selected ? 'outline' : 'primary'}
        disabled={selected || saving}
        onClick={onSelect}
      >
        {selected ? '当前选择' : '选择'}
      </Button>
    </Card>
  );
}
