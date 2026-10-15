import { failedAttempt, useInstallUpdate, usePrepareUpdate, useUpdateInfo } from '@/features/settings/update';

import s from './UpdateNotice.module.css';

// Sidebar notice: shows the running version and, when GitHub has a newer
// release, prepares a download and waits for an explicit restart.
export function UpdateNotice() {
  const { data: info } = useUpdateInfo();
  const install = useInstallUpdate();
  const prepare = usePrepareUpdate();
  if (!info) return null;
  const failed = failedAttempt(info);
  const start = () => {
    if (info.phase !== 'ready') { prepare.mutate(); return; }
    if (window.confirm(`重启并更新到 Learn-With-AI ${info.preparedVersion || info.latest}？\n\n安装完成后自动重新打开，学习数据会保留。`)) install.mutate();
  };
  return (
    <div className={s.root}>
      {info.available && (
        <div className={s.card} role="status">
          <strong>发现新版本 {info.latest}</strong>
          {info.phase === 'downloading' && <span>正在下载并校验安装包…</span>}
          {install.isSuccess && <span>正在安装，完成后会自动重新打开。</span>}
          {install.isError && <span className={s.error}>{(install.error as Error).message}</span>}
          {prepare.isError && <span className={s.error}>{(prepare.error as Error).message}</span>}
          {info.error && <span className={s.error}>{info.error}</span>}
          {!install.isPending && !install.isSuccess && (
            <div className={s.actions}>
              {info.canInstall
                ? <button className={s.primary} disabled={prepare.isPending || info.phase === 'downloading' || info.phase === 'installing'} onClick={start}>{info.phase === 'ready' ? '重启并更新' : info.phase === 'downloading' ? '正在下载…' : '下载更新'}</button>
                : <a className={s.primary} href="https://github.com/porridgeowefish/Learn-With-AI/releases/latest" target="_blank" rel="noreferrer">前往下载</a>}
              {info.notesUrl && <a href={info.notesUrl} target="_blank" rel="noreferrer">更新内容</a>}
            </div>
          )}
        </div>
      )}
      {failed && !info.available && (
        <div className={s.card} role="alert">
          <strong className={s.error}>更新到 {failed.version} 未完成</strong>
          <span>安装程序返回 {failed.exitCode}，当前仍为 {info.current}。可稍后重试，或在 GitHub 手动下载。</span>
        </div>
      )}
      <span className={s.version}>v{info.current}</span>
    </div>
  );
}
