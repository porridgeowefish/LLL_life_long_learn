import { ChangeEvent, useRef, useState } from 'react';

import { sourceFileURL, usePermanentlyDeleteSource, useSource, useSources, useTombstoneSource, useUploadSource } from '@/features/learning/api/learningWorkspace';
import { GeneratedMaterials } from './GeneratedMaterials';

import s from './SourcesView.module.css';

export function SourcesView({ slug }: { slug: string }) {
  const input = useRef<HTMLInputElement>(null);
  const sources = useSources(slug);
  const upload = useUploadSource(slug);
	const tombstone = useTombstoneSource(slug);
	const permanent = usePermanentlyDeleteSource(slug);
	const [pending, setPending] = useState<File | null>(null);
	const [cloudAccepted, setCloudAccepted] = useState(false);
	const [selectedSourceId, setSelectedSourceId] = useState('');
	const selectedSource = useSource(slug, selectedSourceId);

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
      <section className={s.librarySection}>
        <div className={s.sectionHeading}><div><span>上传资料</span><h3>原件与解析内容</h3></div><small>{(sources.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted').length} 份</small></div>
      <div className={s.list}>
        {(sources.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted').map((source) => (
          <article key={source.sourceId}>
            <div className={s.fileIcon}>{extension(source.displayName)}</div>
            <div><strong>{source.displayName}</strong><span>{statusLabel(source.status)}</span>{source.status === 'ready' && <button type="button" className={s.inspect} onClick={() => setSelectedSourceId((current) => current === source.sourceId ? '' : source.sourceId)}>查看解析内容</button>}</div>
			<div className={s.trailing}><time>{new Date(source.updatedAt).toLocaleString()}</time><details><summary>管理</summary><div>
			  <button type="button" onClick={() => { if (window.confirm('从资料库隐藏这份资料？原件和来源记录仍会保留。')) tombstone.mutate(source.sourceId); }}>移出资料库</button>
			  <button type="button" className={s.danger} onClick={() => { if (window.confirm('永久删除原件和派生文件？此操作无法恢复。')) permanent.mutate(source.sourceId); }}>永久删除</button>
			</div></details></div>
          </article>
        ))}
        {!sources.isLoading && (sources.data?.length ?? 0) === 0 && <div className={s.empty}>尚未添加资料。可以上传 PDF、Word、文本、代码或图片，助教会在可行时生成派生内容。</div>}
      </div>
      {selectedSourceId && <div className={s.sourceDetail}>
        {selectedSource.isLoading ? <div className={s.empty}>正在读取资料详情…</div> : selectedSource.data && <>
          <header><div><span>资料内容</span><h3>{selectedSource.data.source.displayName}</h3></div><button type="button" onClick={() => setSelectedSourceId('')}>关闭</button></header>
          <div className={s.fileLinks}>
            <a href={sourceFileURL(slug, selectedSourceId, selectedSource.data.revision.revisionId, 'original')} target="_blank" rel="noreferrer"><strong>原始文件</strong><small>{selectedSource.data.revision.original.filename || selectedSource.data.revision.original.mediaType}</small></a>
            {(selectedSource.data.revision.derivedFiles ?? []).map((file) => <a key={file.key || file.path} href={sourceFileURL(slug, selectedSourceId, selectedSource.data.revision.revisionId, file.key || file.path)} target="_blank" rel="noreferrer"><strong>{derivedLabel(file.key, file.filename)}</strong><small>{file.mediaType}</small></a>)}
          </div>
          {!selectedSource.data.revision.derivedFiles?.length && <div className={s.empty}>这份资料还没有可阅读的解析结果。</div>}
        </>}
      </div>}
      </section>
      <section className={s.librarySection}>
        <div className={s.sectionHeading}><div><span>助教生成资料</span><h3>总结、调研、实验与图表</h3></div></div>
        <GeneratedMaterials slug={slug} emptyText="助教完成并通过校验的总结或材料会自动出现在这里。" />
      </section>
    </section>
  );
}

function extension(name: string) { const ext = name.split('.').pop(); return (ext && ext !== name ? ext : 'FILE').slice(0, 4).toUpperCase(); }
function statusLabel(status: string) { return ({ stored: '原件已保存', processing: '助教解析中', ready: '可供引用', opaque: '作为原件保存', failed: '解析失败' } as Record<string, string>)[status] ?? status; }
function derivedLabel(key?: string, filename?: string) { return filename || ({ text: '解析文本', metadata: '资料元数据', summary: '内容摘要' } as Record<string, string>)[key ?? ''] || key || '派生内容'; }
