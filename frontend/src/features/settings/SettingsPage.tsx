import { useMemo } from 'react';
import { useAppearance, useUpdateAppearance } from './appearance';
import { useAgentRuntimeSettings, useUpdateAgentRuntime } from './settings';
import { ModelsPage } from './ModelsPage';
import { TeacherNotesSettings } from './TeacherNotesSettings';
import { Button } from '@/shared/primitive/Button';
import { Icon } from '@/shared/primitive/Icon';
import type { AgentRuntimeID, AgentRuntimeProvider } from '@/shared/types/domain';
import type { ThemePreference } from '@/shared/lib/theme';
import { useHealth } from '@/app/api/health';
import { failedAttempt, useCheckUpdate, useInstallUpdate, useUpdateInfo } from './update';
import s from './SettingsPage.module.css';

const ORDER: AgentRuntimeID[] = ['codebuddy', 'workbuddy', 'hermes', 'codex', 'trae', 'claude'];
const RUNTIME_LOGOS: Record<AgentRuntimeID, string> = {
  claude: '/img/agent-runtimes/claude.png', codebuddy: '/img/agent-runtimes/codebuddy.png',
  workbuddy: '/img/agent-runtimes/codebuddy.png', hermes: '/img/agent-runtimes/hermes.png',
  codex: '/img/agent-runtimes/codex.png', trae: '/img/agent-runtimes/trae.png',
};

export function SettingsPage() {
  const runtime = useAgentRuntimeSettings();
  const update = useUpdateAgentRuntime();
  const providers = useMemo(() => [...(runtime.data?.providers ?? [])].sort((a, b) => ORDER.indexOf(a.id) - ORDER.indexOf(b.id)), [runtime.data?.providers]);
  const selected = update.data?.selected ?? runtime.data?.selected;
  return <div className={s.host}><div className={s.scroll} data-testid="settings-scroll">
    <header className={s.header}><h1 className={s.title}>设置</h1></header>
    <ModelsPage />
    <section className={s.runtimeSection} aria-labelledby="runtime-title">
      <div className={s.sectionHeader}><h2 id="runtime-title">助教 CLI</h2><Button size="sm" variant="ghost" loading={runtime.isFetching} onClick={() => { void runtime.refetch(); }}>重新检测</Button></div>
      {runtime.isLoading && <p className={s.loading}>检测中…</p>}
      {runtime.isError && <p className={s.error} role="alert">配置读取失败：{(runtime.error as Error).message}</p>}
      <div className={s.grid} role="radiogroup" aria-label="默认助教 CLI">{providers.map(provider => <RuntimeChoice key={provider.id} provider={provider} selected={selected === provider.id} saving={update.isPending} onSelect={() => update.mutate(provider.id)} />)}</div>
      {update.isError && <p className={s.error} role="alert">保存失败：{(update.error as Error).message}</p>}
    </section>
    <TeacherNotesSettings />
    <AppearanceSection />
    <VersionAndDataSection />
    <OpenSourceSection />
  </div></div>;
}
const THEME_OPTIONS: Array<{ id: ThemePreference; name: string; description: string; swatches: string[] }> = [
  { id: 'lychee-paper', name: '荔枝暖纸', description: '温暖、安静的默认阅读主题', swatches: ['#faf8f4', '#fffdf9', '#60775a'] },
  { id: 'mountain-mist', name: '远山雾蓝', description: '清凉克制，适合长时间专注', swatches: ['#f4f7f7', '#fbfdfd', '#557987'] },
  { id: 'wisteria-gray', name: '云海天蓝', description: '白色为主、天蓝点缀的轻盈主题', swatches: ['#f7fbff', '#ffffff', '#69aeda'] },
  { id: 'night-ink', name: '夜读墨绿', description: '低光环境下的舒缓深色主题', swatches: ['#202521', '#292f2a', '#a7b99f'] },
];

function AppearanceSection() {
  const appearance = useAppearance();
  const update = useUpdateAppearance();
  const selected = update.data?.theme ?? appearance.data?.theme ?? 'lychee-paper';
  return (
    <section className={s.appearance} aria-labelledby="appearance-title">
      <div>
        <h2 id="appearance-title">外观</h2>
        <p>选择一套柔和、可访问的配色。更改会立即应用并保存在本机。</p>
      </div>
      <div className={s.themeChoices} role="radiogroup" aria-label="界面主题">
        {THEME_OPTIONS.map((theme) => (
          <button
            key={theme.id}
            type="button"
            role="radio"
            aria-checked={selected === theme.id}
            className={`${s.themeChoice} ${selected === theme.id ? s.themeSelected : ''}`}
            onClick={() => update.mutate(theme.id)}
          >
            <span className={s.themePreview} aria-hidden="true">
              {theme.swatches.map((color) => <i key={color} style={{ background: color }} />)}
            </span>
            <span><strong>{theme.name}</strong><small>{theme.description}</small></span>
            {selected === theme.id && <Icon name="check" size={14} />}
          </button>
        ))}
      </div>
      {update.isError && <div className={s.error}>主题保存失败：{(update.error as Error).message}</div>}
    </section>
  );
}

function VersionAndDataSection() {
  const { data: health } = useHealth();
  const { data: info } = useUpdateInfo();
  const check = useCheckUpdate();
  const install = useInstallUpdate();
  const current = check.data ?? info;
  const failed = failedAttempt(current);
  let status = '';
  if (check.isPending) status = '正在检查…';
  else if (check.isError) status = `检查失败：${(check.error as Error).message}`;
  else if (current?.error) status = current.error;
  else if (current?.available) status = `发现新版本 ${current.latest}`;
  else if (current) status = '已是最新版本';
  return (
    <section className={s.openSource} aria-labelledby="version-data-title">
      <div>
        <h2 id="version-data-title">版本与数据</h2>
        <p>
          {current ? <>当前版本 <strong>{current.current}</strong>{status && <> · {status}</>}</> : '版本信息仅在桌面应用中提供。'}
        </p>
        {failed && <p role="alert">上次更新到 {failed.version} 未完成（安装程序返回 {failed.exitCode}）。</p>}
        <p>
          学习数据目录 <code>{health?.workspace ?? '…'}</code>。项目、笔记、资料、偏好与 API 配置都保存在这里，升级和卸载都不会删除。
        </p>
      </div>
      {current && (
        <div className={s.openSourceActions}>
          <a href="#check" onClick={(event) => { event.preventDefault(); check.mutate(); }}>检查更新</a>
          {current.available && current.canInstall && !install.isSuccess && (
            <a href="#install" onClick={(event) => { event.preventDefault(); if (!install.isPending && window.confirm(`更新到 ${current.latest}？应用会自动关闭并在安装后重新打开，学习数据不受影响。`)) install.mutate(); }}>
              {install.isPending ? '正在下载…' : '立即更新'}
            </a>
          )}
          {install.isSuccess && <span>正在安装，完成后自动重新打开</span>}
          {install.isError && <span role="alert">{(install.error as Error).message}</span>}
        </div>
      )}
    </section>
  );
}

function OpenSourceSection() {
  const repo = 'https://github.com/porridgeowefish/LLL_life_long_learn';
  return (
    <section className={s.openSource} aria-labelledby="open-source-title">
      <div>
        <h2 id="open-source-title">开源与反馈</h2>
        <p>Learn-With-AI 欢迎问题反馈和代码贡献。</p>
      </div>
      <div className={s.openSourceActions}>
        <a href={`${repo}/issues`} target="_blank" rel="noreferrer">报告问题</a>
        <a href={`${repo}/blob/main/CONTRIBUTING.md`} target="_blank" rel="noreferrer">贡献代码</a>
        <a href={repo} target="_blank" rel="noreferrer">查看源码</a>
      </div>
    </section>
  );
}

function RuntimeChoice({ provider, selected, saving, onSelect }: {
  provider: AgentRuntimeProvider; selected: boolean; saving: boolean; onSelect: () => void;
}) {
  const status = `${provider.name}，${provider.available ? '已检测到' : '未检测到'}${selected ? '，当前选择' : ''}`;
  return <button type="button" role="radio" aria-checked={selected} aria-label={status} title={status}
    className={`${s.runtimeChoice} ${provider.available ? s.detected : s.undetected} ${selected ? s.runtimeSelected : ''}`}
    disabled={saving || selected} onClick={onSelect}>
    <img src={RUNTIME_LOGOS[provider.id]} alt="" aria-hidden="true" /><span>{provider.name}</span>
    {selected && <Icon name="check" size={13} />}
  </button>;
}
