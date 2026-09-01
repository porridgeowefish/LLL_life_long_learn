import { ChangeEvent, useRef, useState } from 'react';

import { usePermanentlyDeleteSource, useSources, useTombstoneSource, useUploadSource } from '@/api/learningWorkspace';

import s from './SourcesView.module.css';

export function SourcesView({ slug }: { slug: string }) {
  const input = useRef<HTMLInputElement>(null);
  const sources = useSources(slug);
  const upload = useUploadSource(slug);
	const tombstone = useTombstoneSource(slug);
	const permanent = usePermanentlyDeleteSource(slug);
	const [pending, setPending] = useState<File | null>(null);
	const [cloudAccepted, setCloudAccepted] = useState(false);

  const choose = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
	if (file) { setPending(file); setCloudAccepted(false); }
    event.target.value = '';
  };

  return (
    <section className={s.sources}>
      <header>
        <div><span>资料库</span><h2>教师与助教可引用的材料</h2><p>原件会保存在当前学习单元；解析是独立助教任务，不阻塞教师对话。</p></div>
        <button onClick={() => input.current?.click()} disabled={upload.isPending}>{upload.isPending ? '正在保存…' : '上传资料'}</button>
        <input ref={input} type="file" hidden onChange={choose} />
      </header>
      {upload.isError && <div className={s.error}>上传失败：{upload.error.message}</div>}
	  {pending && <div className={s.confirm} role="dialog" aria-label="确认资料处理">
		<div><strong>{pending.name}</strong><p>原件会保存在本学习单元。若交给助教解析，文件内容可能发送到当前 CLI Agent 所配置的模型服务。</p></div>
		<label><input type="checkbox" checked={cloudAccepted} onChange={(event) => setCloudAccepted(event.target.checked)} /> 我了解并同意本次解析可能使用云端模型</label>
		<div className={s.confirmActions}>
		  <button type="button" onClick={() => setPending(null)}>取消</button>
		  <button type="button" onClick={() => upload.mutate({ file: pending, parseApproved: false, cloudDisclosureAccepted: false }, { onSuccess: () => setPending(null) })}>仅保存原件</button>
		  <button type="button" className={s.primary} disabled={!cloudAccepted} onClick={() => upload.mutate({ file: pending, parseApproved: true, cloudDisclosureAccepted: true }, { onSuccess: () => setPending(null) })}>保存并解析</button>
		</div>
	  </div>}
      <div className={s.list}>
        {(sources.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted').map((source) => (
          <article key={source.sourceId}>
            <div className={s.fileIcon}>{extension(source.displayName)}</div>
            <div><strong>{source.displayName}</strong><span>{statusLabel(source.status)}</span></div>
			<div className={s.trailing}><time>{new Date(source.updatedAt).toLocaleString()}</time><details><summary>管理</summary><div>
			  <button type="button" onClick={() => { if (window.confirm('从资料库隐藏这份资料？原件和来源记录仍会保留。')) tombstone.mutate(source.sourceId); }}>移出资料库</button>
			  <button type="button" className={s.danger} onClick={() => { if (window.confirm('永久删除原件和派生文件？此操作无法恢复。')) permanent.mutate(source.sourceId); }}>永久删除</button>
			</div></details></div>
          </article>
        ))}
        {!sources.isLoading && (sources.data?.length ?? 0) === 0 && <div className={s.empty}>尚未添加资料。可以上传 PDF、Word、文本、代码或图片，助教会在可行时生成派生内容。</div>}
      </div>
    </section>
  );
}

function extension(name: string) { const ext = name.split('.').pop(); return (ext && ext !== name ? ext : 'FILE').slice(0, 4).toUpperCase(); }
function statusLabel(status: string) { return ({ stored: '原件已保存', processing: '助教解析中', ready: '可供引用', opaque: '作为原件保存', failed: '解析失败' } as Record<string, string>)[status] ?? status; }
