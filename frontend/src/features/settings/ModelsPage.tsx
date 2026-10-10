import { useEffect, useRef, useState } from 'react';
import { useAskAiSettings, useProbeAskAi, useSaveAskAiSettings, type AskConfig, type AskProvider } from './askAi';
import { applyProviderPreset, CODING_PLAN_LINKS, presetForProvider, PROVIDER_PRESETS } from './providerPresets';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';
import { SecretInput } from '@/shared/primitive/SecretInput';
import s from './ModelsPage.module.css';

const copyConfig = (config: AskConfig): AskConfig => ({ ...config, providers: config.providers.map(provider => ({ ...provider })),
  bindings: Object.fromEntries(Object.entries(config.bindings ?? {}).map(([key, value]) => [key, { ...value }])),
});
const freshProvider = (): AskProvider => applyProviderPreset({ id: `provider-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
  kind: 'openai', baseURL: '', apiKey: '', model: '' }, PROVIDER_PRESETS[0]);

export function ModelsPage() {
  const settings = useAskAiSettings();
  const save = useSaveAskAiSettings();
  const probe = useProbeAskAi();
  const [draft, setDraft] = useState<AskConfig | null>(null);
  const [dirty, setDirty] = useState(false);
  const dirtyRef = useRef(false);
  const revision = useRef(0);
  const [customIds, setCustomIds] = useState<Set<string>>(new Set());
  useEffect(() => { if (settings.data && !dirtyRef.current) setDraft(copyConfig(settings.data)); }, [settings.data]);

  const edit = (change: (current: AskConfig) => AskConfig) => {
    dirtyRef.current = true; revision.current += 1; setDirty(true); if (!save.isPending) save.reset();
    setDraft(current => current ? change(current) : current);
  };
  const patchProvider = (id: string, patch: Partial<AskProvider>) => {
    probe.reset();
    edit(current => {
      const bindings = { ...current.bindings };
      if (patch.model !== undefined) {
        for (const key of ['teacher', 'annotationAskAI']) {
          if (bindings[key]?.providerId === id) bindings[key] = { providerId: id };
        }
      }
      return { ...current, bindings, providers: current.providers.map(provider => provider.id === id ? { ...provider, ...patch } : provider) };
    });
  };
  const setBinding = (key: string, providerId: string, model?: string) => edit(current => {
    const bindings = { ...current.bindings };
    if (providerId) bindings[key] = { providerId, ...(model !== undefined ? { model } : {}) };
    else delete bindings[key];
    return { ...current, bindings, ...(key === 'teacher' ? { default: providerId } : {}) };
  });
  const removeProvider = (id: string) => {
    probe.reset();
    edit(current => {
      const providers = current.providers.filter(provider => provider.id !== id);
      const fallback = providers[0]?.id ?? '';
      const bindings = { ...current.bindings };
      for (const [key, binding] of Object.entries(bindings)) {
        if (binding.providerId !== id) continue;
        if (fallback && key !== 'ocr') bindings[key] = { providerId: fallback };
        else delete bindings[key];
      }
      return { ...current, providers, bindings, default: current.default === id ? fallback : current.default };
    });
  };
  const onSave = () => {
    if (!draft) return;
    const savedRevision = revision.current;
    save.mutate(copyConfig(draft), { onSuccess: () => {
      if (revision.current === savedRevision) { dirtyRef.current = false; setDirty(false); }
    } });
  };
  if (settings.isError && !draft) return <section className={s.section} role="alert">模型配置读取失败：{(settings.error as Error).message}</section>;
  if (!draft) return <section className={s.section} aria-busy="true">加载模型配置…</section>;
  const providerOptions = draft.providers.map(provider => <option key={provider.id} value={provider.id}>{provider.name || provider.model || '未命名连接'}</option>);

  return <section className={s.section} aria-labelledby="api-connections-title">
    <header className={s.header}><h2 id="api-connections-title">API 模型</h2><div className={s.actions}>
      <Button size="sm" variant="outline" onClick={() => edit(current => { const provider = freshProvider(); return { ...current, providers: [...current.providers, provider], default: current.default || provider.id }; })}><Icon name="plus" size={14} />添加连接</Button>
      <Button size="sm" loading={save.isPending} onClick={onSave}>保存配置</Button>
    </div></header>
    {save.isError && <p className={s.error} role="alert">保存失败：{(save.error as Error).message}</p>}
    <p className={s.saveStatus} role="status">{dirty ? '有未保存的修改' : save.isSuccess ? '配置已保存' : ''}</p>
    <div className={s.list} aria-label="已配置模型">
      {!draft.providers.length && <p className={s.empty}>尚未添加 API 连接。</p>}
      {draft.providers.map((provider, index) => {
        const preset = customIds.has(provider.id) ? undefined : presetForProvider(provider);
        const currentProbe = probe.variables?.providerId === provider.id && JSON.stringify(probe.variables?.inline) === JSON.stringify(provider);
        const error = currentProbe ? (probe.isError ? (probe.error as Error).message : probe.data?.ok === false ? (probe.data.error || '提供方未返回成功响应') : undefined) : undefined;
        return <article key={provider.id} className={s.card} aria-label={`API 连接 ${index + 1}`}>
          <div className={s.formGrid}>
            <label>提供方<select value={preset?.id ?? 'custom'} onChange={event => {
              const selected = PROVIDER_PRESETS.find(item => item.id === event.target.value);
              setCustomIds(current => { const next = new Set(current); if (selected) next.delete(provider.id); else next.add(provider.id); return next; });
              if (selected) patchProvider(provider.id, applyProviderPreset(provider, selected)); else probe.reset();
            }}>{(['官方', '聚合 / 中转', '套餐'] as const).map(group => <optgroup key={group} label={group}>{PROVIDER_PRESETS.filter(item => item.group === group).map(item => <option key={item.id} value={item.id} disabled={item.disabled}>{item.name}{item.disabled ? '（仅适用编程客户端）' : ''}</option>)}{group === '套餐' && CODING_PLAN_LINKS.map(item => <option key={item.id} value={item.id} disabled>{item.name}（编程 CLI 专用）</option>)}</optgroup>)}<option value="custom">自行配置</option></select></label>
            {preset && <label>模型<select value={provider.model} onChange={event => { const model = preset.models.find(item => item.id === event.target.value)!; patchProvider(provider.id, { model: model.id, contextWindowTokens: model.contextWindowTokens }); }}>{preset.models.map((model, modelIndex) => <option key={model.id} value={model.id}>{model.id}{modelIndex === 0 ? '（推荐）' : ''}</option>)}</select></label>}
            {!preset && <>
              <label>显示名称<input value={provider.name ?? ''} onChange={event => patchProvider(provider.id, { name: event.target.value })} /></label>
              <label>接口协议<select value={provider.kind} onChange={event => patchProvider(provider.id, { kind: event.target.value as AskProvider['kind'] })}><option value="openai">OpenAI 兼容</option><option value="anthropic">Anthropic</option></select></label>
              <label>模型名称<input value={provider.model} onChange={event => patchProvider(provider.id, { model: event.target.value })} /></label>
              <label className={s.wide}>Base URL<input value={provider.baseURL} onChange={event => patchProvider(provider.id, { baseURL: event.target.value })} /></label>
              <label>上下文 Token<input type="number" min={16384} step={1024} value={provider.contextWindowTokens ?? ''} placeholder="留空使用安全估计" onChange={event => patchProvider(provider.id, { contextWindowTokens: event.target.value ? Number(event.target.value) : undefined })} /></label>
              <div className={s.flags}><label><input type="checkbox" checked={!!provider.reasoning} onChange={event => patchProvider(provider.id, { reasoning: event.target.checked })} />显示思考过程</label>{provider.kind === 'anthropic' && <label><input type="checkbox" checked={!!provider.thinking} onChange={event => patchProvider(provider.id, { thinking: event.target.checked })} />启用思考</label>}</div>
            </>}
            <label className={s.wide}>API Key<SecretInput value={provider.apiKey} onChange={value => patchProvider(provider.id, { apiKey: value, apiKeyEnv: '' })} />{provider.apiKeyEnv && !provider.apiKey && <small>已使用环境变量密钥；留空保留。</small>}</label>
          </div>
          <div className={s.cardFooter}><div className={s.links}>{preset && <><a href={preset.keyURL} target="_blank" rel="noreferrer">获取 Key</a><a href={preset.docsURL} target="_blank" rel="noreferrer">官方文档</a>{preset.planURL && <a href={preset.planURL} target="_blank" rel="noreferrer">Token Plan</a>}{preset.note && <span>{preset.note}</span>}</>}</div><div className={s.actions}>
            {currentProbe && probe.data?.ok && <span className={s.connected} role="status">已连通</span>}
            <Button size="sm" variant="outline" loading={!!(currentProbe && probe.isPending)} onClick={() => probe.mutate({ providerId: provider.id, inline: { ...provider } })}>探测连接</Button>
            <Button size="sm" variant="ghost" onClick={() => removeProvider(provider.id)}>删除</Button>
          </div></div>{error && <p className={s.error} role="alert">连接失败：{error}</p>}
        </article>;
      })}
    </div>
    <div className={s.bindings}>
      <label>教师默认模型<select value={draft.bindings?.teacher?.providerId ?? draft.default} disabled={!draft.providers.length} onChange={event => setBinding('teacher', event.target.value)}>{!draft.providers.length && <option value="">先添加连接</option>}{providerOptions}</select></label>
      <label>Ask AI 默认模型<select value={draft.bindings?.annotationAskAI?.providerId ?? draft.default} disabled={!draft.providers.length} onChange={event => setBinding('annotationAskAI', event.target.value)}>{!draft.providers.length && <option value="">先添加连接</option>}{providerOptions}</select></label>
      <label>OCR 图片识别<select value={draft.bindings?.ocr?.providerId ?? ''} onChange={event => setBinding('ocr', event.target.value, '')}><option value="">停用</option>{providerOptions}</select></label>
      {draft.bindings?.ocr?.providerId && <label>OCR 模型覆盖<input value={draft.bindings.ocr.model ?? ''} placeholder="使用连接的模型" onChange={event => setBinding('ocr', draft.bindings!.ocr.providerId, event.target.value)} /></label>}
      <label>搜索引擎<select value={draft.searchEngine} onChange={event => edit(current => ({ ...current, searchEngine: event.target.value as AskConfig['searchEngine'] }))}><option value="google">Google</option><option value="bing">Bing</option></select></label>
    </div>
    <details className={s.planNotes}><summary>编程套餐适用范围</summary>{CODING_PLAN_LINKS.map(item => <a key={item.id} href={item.docsURL} target="_blank" rel="noreferrer">{item.name}：{item.note}</a>)}</details>
  </section>;
}
