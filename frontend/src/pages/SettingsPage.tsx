import { useEffect, useMemo, useState } from 'react';

import { useAgentRuntimeSettings, useUpdateAgentRuntime } from '@/api/settings';
import {
  useAskAiSettings,
  useSaveAskAiSettings,
  useProbeAskAi,
  type AskConfig,
  type AskProvider,
} from '@/api/askAi';
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

        <AskAiSection />

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

const MASKED_KEY = '••••';

function newProvider(): AskProvider {
  return {
    id: `prov-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    kind: 'openai',
    name: '',
    baseURL: '',
    apiKey: '',
    model: '',
  };
}

function AskAiSection() {
  const { data, isLoading, isError, error } = useAskAiSettings();
  const save = useSaveAskAiSettings();
  const probe = useProbeAskAi();

  // Local draft, re-seeded whenever the server snapshot (`data`) changes —
  // initial load and after a save (the save mutation invalidates the query,
  // refetch returns a new `data` identity, this effect re-seeds). In-flight
  // edits are preserved between fetches.
  const [draft, setDraft] = useState<AskConfig | null>(null);
  useEffect(() => {
    if (!data) return;
    setDraft({
      default: data.default,
      searchEngine: data.searchEngine,
      providers: data.providers.map((p) => ({ ...p })),
    });
  }, [data]);

  if (isLoading) return <div className={s.loading}>加载 Ask-AI 配置…</div>;
  if (isError) {
    return (
      <div className={s.error} role="alert">
        Ask-AI 配置读取失败：{(error as Error).message}
      </div>
    );
  }
  if (!draft) return null;

  const patchProvider = (id: string, patch: Partial<AskProvider>) => {
    setDraft((d) =>
      d ? { ...d, providers: d.providers.map((p) => (p.id === id ? { ...p, ...patch } : p)) } : d,
    );
  };

  const removeProvider = (id: string) => {
    setDraft((d) => {
      if (!d) return d;
      const providers = d.providers.filter((p) => p.id !== id);
      return { ...d, providers, default: d.default === id ? (providers[0]?.id ?? '') : d.default };
    });
  };

  const addProvider = () => {
    setDraft((d) => {
      if (!d) return d;
      const p = newProvider();
      return { ...d, providers: [...d.providers, p], default: d.default || p.id };
    });
  };

  const setDefault = (id: string) => setDraft((d) => (d ? { ...d, default: id } : d));
  const setSearchEngine = (engine: 'google' | 'bing') =>
    setDraft((d) => (d ? { ...d, searchEngine: engine } : d));

  const onSave = () => save.mutate(draft);

  return (
    <section className={s.askSection} aria-label="Ask-AI 模型源">
      <header className={s.header}>
        <div>
          <h2 className={s.title}>Ask-AI 模型源</h2>
          <p className={s.subtitle}>
            选中教程文字后「问 AI」使用的模型源。API Key 在服务端加密；未修改时回显占位符 {MASKED_KEY}。
          </p>
        </div>
        <div className={s.askActions}>
          <label className={s.askEngine}>
            搜索引擎
            <select
              value={draft.searchEngine}
              onChange={(e) => setSearchEngine(e.target.value as 'google' | 'bing')}
            >
              <option value="google">Google</option>
              <option value="bing">Bing</option>
            </select>
          </label>
          <Button variant="outline" onClick={addProvider}>添加模型源</Button>
          <Button variant="primary" loading={save.isPending} onClick={onSave}>保存</Button>
        </div>
      </header>

      {save.isError && (
        <div className={s.error} role="alert">
          保存失败：{(save.error as Error).message}
        </div>
      )}

      <div className={s.askList}>
        {draft.providers.length === 0 && (
          <div className={s.loading}>还没有模型源。点击「添加模型源」新建一个。</div>
        )}
        {draft.providers.map((p) => (
          <ProviderRow
            key={p.id}
            provider={p}
            isDefault={draft.default === p.id}
            onSetDefault={() => setDefault(p.id)}
            onPatch={(patch) => patchProvider(p.id, patch)}
            onRemove={() => removeProvider(p.id)}
            onProbe={() => probe.mutate({ providerId: p.id })}
            probeResult={probe.variables?.providerId === p.id ? probe.data : undefined}
            probePending={probe.isPending && probe.variables?.providerId === p.id}
            probeError={probe.variables?.providerId === p.id && probe.isError
              ? (probe.error as Error).message
              : undefined}
          />
        ))}
      </div>
    </section>
  );
}

function ProviderRow({
  provider,
  isDefault,
  onSetDefault,
  onPatch,
  onRemove,
  onProbe,
  probeResult,
  probePending,
  probeError,
}: {
  provider: AskProvider;
  isDefault: boolean;
  onSetDefault: () => void;
  onPatch: (patch: Partial<AskProvider>) => void;
  onRemove: () => void;
  onProbe: () => void;
  probeResult?: { ok: boolean; error?: string };
  probePending: boolean;
  probeError?: string;
}) {
  const showThinking = provider.kind === 'anthropic';
  return (
    <Card variant="outlined" className={s.card}>
      <div className={s.cardTop}>
        <label className={s.askField}>
          名称
          <input
            value={provider.name ?? ''}
            onChange={(e) => onPatch({ name: e.target.value })}
            placeholder="如：GPT-4o / Claude"
          />
        </label>
        <label className={s.askField}>
          类型
          <select
            value={provider.kind}
            onChange={(e) => onPatch({ kind: e.target.value as 'openai' | 'anthropic' })}
          >
            <option value="openai">openai</option>
            <option value="anthropic">anthropic</option>
          </select>
        </label>
        <label className={s.askDefault}>
          <input type="radio" checked={isDefault} onChange={onSetDefault} />
          默认
        </label>
      </div>
      <label className={s.askField}>
        Base URL
        <input
          value={provider.baseURL}
          onChange={(e) => onPatch({ baseURL: e.target.value })}
          placeholder="https://api.openai.com/v1"
        />
      </label>
      <label className={s.askField}>
        Model
        <input
          value={provider.model}
          onChange={(e) => onPatch({ model: e.target.value })}
          placeholder="gpt-4o / claude-sonnet-4"
        />
      </label>
      <label className={s.askField}>
        API Key
        <input
          value={provider.apiKey}
          onChange={(e) => onPatch({ apiKey: e.target.value })}
          placeholder={MASKED_KEY}
          type="password"
        />
      </label>
      <div className={s.askFlags}>
        <label>
          <input
            type="checkbox"
            checked={!!provider.reasoning}
            onChange={(e) => onPatch({ reasoning: e.target.checked })}
          />
          reasoning
        </label>
        {showThinking && (
          <label>
            <input
              type="checkbox"
              checked={!!provider.thinking}
              onChange={(e) => onPatch({ thinking: e.target.checked })}
            />
            thinking
          </label>
        )}
      </div>
      <div className={s.askRowActions}>
        <Button variant="outline" size="sm" loading={probePending} onClick={onProbe}>
          探测
        </Button>
        {probeError && <span className={s.askProbeErr}>失败：{probeError}</span>}
        {probeResult?.ok === false && !probeError && (
          <span className={s.askProbeErr}>失败：{probeResult.error ?? '未知'}</span>
        )}
        {probeResult?.ok && <span className={s.askProbeOk}>已连通</span>}
        <Button variant="ghost" size="sm" onClick={onRemove}>删除</Button>
      </div>
    </Card>
  );
}
