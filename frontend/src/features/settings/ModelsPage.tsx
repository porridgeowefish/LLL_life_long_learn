import { useEffect, useState } from 'react';

import {
  useAskAiSettings,
  useProbeAskAi,
  useSaveAskAiSettings,
  type AskConfig,
  type AskProvider,
} from '@/features/settings/askAi';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';

import s from './ModelsPage.module.css';

const MASKED_KEY = '••••';

function freshProvider(): AskProvider {
  return {
    id: `provider-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    kind: 'openai',
    name: '',
    baseURL: 'https://api.openai.com/v1',
    apiKey: '',
    model: '',
    contextWindowTokens: 262144,
  };
}

export function ModelsPage() {
  const settings = useAskAiSettings();
  const save = useSaveAskAiSettings();
  const probe = useProbeAskAi();
  const [draft, setDraft] = useState<AskConfig | null>(null);

  useEffect(() => {
    if (!settings.data) return;
    setDraft({
      ...settings.data,
      providers: settings.data.providers.map((provider) => ({ ...provider })),
      bindings: { ...(settings.data.bindings ?? {}) },
    });
  }, [settings.data]);

  const patchProvider = (id: string, patch: Partial<AskProvider>) => {
    setDraft((current) => current ? {
      ...current,
      providers: current.providers.map((provider) => provider.id === id ? { ...provider, ...patch } : provider),
    } : current);
  };

  const setTeacherDefault = (provider: AskProvider) => {
    setDraft((current) => current ? {
      ...current,
      default: provider.id,
      bindings: {
        ...(current.bindings ?? {}),
        teacher: { providerId: provider.id },
      },
    } : current);
  };

  const removeProvider = (id: string) => {
    setDraft((current) => {
      if (!current) return current;
      const providers = current.providers.filter((provider) => provider.id !== id);
      const fallback = providers[0];
      const bindings = { ...(current.bindings ?? {}) };
      for (const [key, binding] of Object.entries(bindings)) {
        if (binding.providerId === id) {
          if (fallback) bindings[key] = { providerId: fallback.id };
          else delete bindings[key];
        }
      }
      return { ...current, providers, bindings, default: current.default === id ? (fallback?.id ?? '') : current.default };
    });
  };

  const setAskAiProvider = (providerId: string) => {
    setDraft((current) => current ? {
      ...current,
      bindings: { ...(current.bindings ?? {}), annotationAskAI: { providerId } },
    } : current);
  };

  if (settings.isLoading || !draft) return <main className={s.page}>加载模型配置…</main>;
  if (settings.isError) return <main className={s.page} role="alert">模型配置读取失败：{(settings.error as Error).message}</main>;

  return (
    <main className={s.page}>
      <header className={s.hero}>
        <div>
          <span className={s.eyebrow}>MODEL CONNECTIONS</span>
          <h1>API 模型</h1>
          <p>模型连接只在这里管理；配置完成后，教师对话可按学习单元自由切换。</p>
        </div>
        <div className={s.actions}>
          <Button variant="outline" onClick={() => setDraft((current) => current ? { ...current, providers: [...current.providers, freshProvider()] } : current)}>
            <Icon name="plus" size={14} /> 添加模型
          </Button>
          <Button loading={save.isPending} onClick={() => save.mutate(draft)}>保存配置</Button>
        </div>
      </header>

      <ol className={s.steps} aria-label="模型配置流程">
        <li><b>1</b><span>添加模型<small>创建一个 API 连接</small></span></li>
        <li><b>2</b><span>填写与探测<small>确认地址、密钥和模型</small></span></li>
        <li><b>3</b><span>在对话中选择<small>教师输入框直接切换</small></span></li>
      </ol>

      <section className={s.serviceBindings} aria-labelledby="service-bindings-title">
        <div>
          <span>服务绑定</span>
          <h2 id="service-bindings-title">教师与 Ask AI</h2>
          <p>教师默认模型可在下方连接卡片中选择；Ask AI 使用轻量提示词，但复用同一组 API 连接。</p>
        </div>
        <label>
          Ask AI 默认模型
          <select
            value={draft.bindings?.annotationAskAI?.providerId ?? draft.default}
            onChange={(event) => setAskAiProvider(event.target.value)}
            disabled={draft.providers.length === 0}
          >
            {draft.providers.length === 0 && <option value="">请先添加模型</option>}
            {draft.providers.map((provider) => <option key={provider.id} value={provider.id}>{provider.name ? `${provider.name} · ${provider.model}` : provider.model}</option>)}
          </select>
        </label>
      </section>

      {save.isSuccess && <div className={s.success}>配置已保存，对话框中的模型列表会自动更新。</div>}
      {(save.isError || probe.isError) && <div className={s.error} role="alert">操作失败：{((save.error ?? probe.error) as Error).message}</div>}

      <section className={s.list} aria-label="已配置模型">
        {draft.providers.length === 0 && <div className={s.empty}>还没有模型。添加第一个连接后，就能在教师对话中使用。</div>}
        {draft.providers.map((provider, index) => {
          const teacherDefault = (draft.bindings?.teacher?.providerId ?? draft.default) === provider.id;
          const probing = probe.isPending && probe.variables?.providerId === provider.id;
          const probed = probe.data?.ok && probe.variables?.providerId === provider.id;
          return (
            <article key={provider.id} className={s.card}>
              <div className={s.cardHead}>
                <div><span>连接 {String(index + 1).padStart(2, '0')}</span><h2>{provider.name || provider.model || '未命名模型'}</h2></div>
                <label className={s.default}><input type="radio" checked={teacherDefault} onChange={() => setTeacherDefault(provider)} />教师默认</label>
              </div>
              <div className={s.formGrid}>
                <label>显示名称<input value={provider.name ?? ''} placeholder="例如 Claude Sonnet" onChange={(event) => patchProvider(provider.id, { name: event.target.value })} /></label>
                <label>接口协议<select value={provider.kind} onChange={(event) => patchProvider(provider.id, { kind: event.target.value as AskProvider['kind'] })}><option value="openai">OpenAI 兼容</option><option value="anthropic">Anthropic</option></select></label>
                <label className={s.wide}>Base URL<input value={provider.baseURL} placeholder="https://api.openai.com/v1" onChange={(event) => patchProvider(provider.id, { baseURL: event.target.value })} /></label>
                <label>模型名称<input value={provider.model} placeholder="gpt-5 / claude-sonnet" onChange={(event) => patchProvider(provider.id, { model: event.target.value })} /></label>
                <label>上下文 Token<input type="number" min={16384} step={1024} value={provider.contextWindowTokens ?? ''} placeholder="默认安全阈值 256K" onChange={(event) => patchProvider(provider.id, { contextWindowTokens: event.target.value ? Number(event.target.value) : undefined })} /></label>
                <label className={s.wide}>API Key<input type="password" value={provider.apiKey} placeholder={MASKED_KEY} onChange={(event) => patchProvider(provider.id, { apiKey: event.target.value })} /></label>
              </div>
              <div className={s.cardFoot}>
                <div className={s.flags}>
                  <label><input type="checkbox" checked={!!provider.reasoning} onChange={(event) => patchProvider(provider.id, { reasoning: event.target.checked })} />显示模型提供的思考过程</label>
                  {provider.kind === 'anthropic' && <label><input type="checkbox" checked={!!provider.thinking} onChange={(event) => patchProvider(provider.id, { thinking: event.target.checked })} />启用 thinking</label>}
                </div>
                <div className={s.rowActions}>
                  {probed && <span className={s.connected}>已连通</span>}
                  <Button size="sm" variant="outline" loading={probing} onClick={() => probe.mutate({ providerId: provider.id })}>探测连接</Button>
                  <Button size="sm" variant="ghost" onClick={() => removeProvider(provider.id)}>删除</Button>
                </div>
              </div>
            </article>
          );
        })}
      </section>
    </main>
  );
}
