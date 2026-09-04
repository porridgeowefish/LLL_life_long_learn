import { ChangeEvent, useRef, useState } from 'react';

import { useSources, useUploadSource } from '@/features/learning/api/learningWorkspace';

import s from './SourcesView.module.css';

// This page is deliberately a file ledger, not a second content reader. The
// parsed Markdown is available only from the teacher's citation picker.
export function SourcesView({ slug }: { slug: string }) {
  const input = useRef<HTMLInputElement>(null);
  const sources = useSources(slug);
  const upload = useUploadSource(slug);
  const [pending, setPending] = useState<File | null>(null);
  const [cloudAccepted, setCloudAccepted] = useState(false);

  const choose = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    event.target.value = '';
    setPending(file);
    setCloudAccepted(false);
  };

  const submit = () => {
    if (!pending || !cloudAccepted) return;
    upload.mutate({ file: pending, parseApproved: true, cloudDisclosureAccepted: true }, { onSuccess: () => setPending(null) });
  };
  const visible = (sources.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted');

  return (
    <section className={s.sources}>
      <header>
        <div><span>资料库</span><h2>当前文件</h2><p>文件上传后由助教在后台提取为可引用文本；此处不展示解析内容或助教资产。</p></div>
        <button type="button" onClick={() => input.current?.click()} disabled={upload.isPending}>{upload.isPending ? '正在保存…' : '上传资料'}</button>
        <input ref={input} type="file" hidden onChange={choose} />
      </header>
      {upload.isError && <div className={s.error}>上传失败：{upload.error.message}</div>}
      {pending && <div className={s.confirm} role="dialog" aria-label="确认资料处理">
        <div><strong>{pending.name}</strong><p>资料会保存到当前学习单元，并由助教异步提取为 Markdown。解析不阻塞教师对话。</p></div>
        <label><input type="checkbox" checked={cloudAccepted} onChange={(event) => setCloudAccepted(event.target.checked)} /> 我了解并同意本次解析可能使用云端模型</label>
        <div className={s.confirmActions}><button type="button" onClick={() => setPending(null)}>取消</button><button type="button" className={s.primary} disabled={!cloudAccepted || upload.isPending} onClick={submit}>保存并解析</button></div>
      </div>}
      <section className={s.librarySection}>
        <div className={s.sectionHeading}><div><span>上传资料</span><h3>文件项</h3></div><small>{visible.length} 份</small></div>
        <div className={s.list}>
          {visible.map((source) => <article key={source.sourceId}>
            <div className={s.fileIcon}>{extension(source.displayName)}</div>
            <div><strong>{source.displayName}</strong><span>{statusLabel(source.status)}</span></div>
            <div className={s.trailing}><time>{new Date(source.updatedAt).toLocaleString()}</time></div>
          </article>)}
          {!sources.isLoading && !visible.length && <div className={s.empty}>尚未上传资料</div>}
        </div>
      </section>
    </section>
  );
}

function extension(name: string) {
  const value = name.split('.').pop()?.toUpperCase() || 'FILE';
  return value.slice(0, 5);
}

function statusLabel(status: string) {
  const labels: Record<string, string> = { stored: '已保存，未解析', processing: '助教正在解析', ready: '解析完成，可在教师对话中引用', opaque: '已保存，暂不支持解析', failed: '解析失败' };
  return labels[status] ?? status;
}
