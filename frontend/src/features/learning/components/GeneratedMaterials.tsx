import { useEffect, useState } from 'react';

import {
  generatedArtifactOpenURL,
  useGeneratedArtifactEntry,
  useGeneratedArtifacts,
} from '@/features/learning/api/learningWorkspace';
import { MarkdownView } from '@/shared/primitive/MarkdownView';

import s from './GeneratedMaterials.module.css';

export function GeneratedMaterials({ slug, emptyText = '助教完成并通过校验的材料会自动出现在这里。' }: { slug: string; emptyText?: string }) {
  const artifacts = useGeneratedArtifacts(slug);
  const [selectedId, setSelectedId] = useState('');
  const selected = artifacts.data?.find((item) => item.artifactId === selectedId) ?? artifacts.data?.[0];
  const entry = useGeneratedArtifactEntry(slug, selected);

  useEffect(() => {
    if (selectedId && !(artifacts.data ?? []).some((item) => item.artifactId === selectedId)) setSelectedId('');
  }, [artifacts.data, selectedId]);

  if (artifacts.isLoading) return <div className={s.empty}>正在读取助教资料…</div>;
  if (!artifacts.data?.length) return <div className={s.empty}>{emptyText}</div>;

  return <div className={s.layout}>
    <aside className={s.list} aria-label="助教生成资料">
      {artifacts.data.map((artifact) => <button key={artifact.artifactId} className={artifact.artifactId === selected?.artifactId ? s.active : ''} onClick={() => setSelectedId(artifact.artifactId)}>
        <strong>{artifact.title}</strong>
        <span>{artifact.description || `${artifact.fileCount} 个文件`}</span>
        <small>{new Date(artifact.createdAt).toLocaleString()}</small>
      </button>)}
    </aside>
    <article className={s.preview}>
      {selected && <>
        <header>
          <div><span>助教资料 · {selected.kind || 'artifact'}</span><h3>{selected.title}</h3></div>
          <a href={generatedArtifactOpenURL(slug, selected.artifactId)} target="_blank" rel="noreferrer">打开原始成果</a>
        </header>
        {selected.description && <p className={s.description}>{selected.description}</p>}
        {entry.isLoading ? <div className={s.empty}>正在加载内容…</div>
          : entry.data ? <MarkdownView source={entry.data} className={s.markdown} />
            : <div className={s.empty}>该成果包含 {selected.fileCount} 个文件；入口不是 Markdown，请打开原始成果查看或下载。</div>}
      </>}
    </article>
  </div>;
}
