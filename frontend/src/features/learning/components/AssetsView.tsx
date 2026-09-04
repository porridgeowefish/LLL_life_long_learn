import { useEffect, useState } from 'react';

import {
  useAsset,
  useAssets,
  useGeneratedArtifacts,
  useSaveAsset,
} from '@/features/learning/api/learningWorkspace';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import { ExplainReader } from '@/features/learning/components/ExplainReader';
import { PracticeFlow } from '@/features/legacy-zones';
import { GeneratedMaterials } from './GeneratedMaterials';
import { MarkdownBodyReader } from './MarkdownBodyReader';

import s from './AssetsView.module.css';

type AssetKey = 'intro' | 'body' | 'practice';

export function AssetsView({ slug }: { slug: string }) {
  const [view, setView] = useState<AssetKey | 'generated'>('body');
  const key: AssetKey = view === 'generated' ? 'body' : view;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');
  const assets = useAssets(slug);
  const asset = useAsset(slug, key);
  const save = useSaveAsset(slug, key);
  const generated = useGeneratedArtifacts(slug);
  const structuredBody = key === 'body' && asset.data?.contentKind === 'explain-pages';
  const structuredPractice = key === 'practice' && asset.data?.contentKind === 'practice-set';
  const contractBacked = structuredBody || structuredPractice;

  useEffect(() => { if (!editing) setDraft(asset.data?.content ?? ''); }, [asset.data?.content, editing]);

  const commit = () => {
    if (!asset.data) return;
    save.mutate({ baseEditRevision: asset.data.meta.editRevision, content: draft }, { onSuccess: () => setEditing(false) });
  };

  return (
    <section className={s.assets}>
      <div className={s.assetTabs}>
        {(assets.data ?? []).map((meta) => <button key={meta.key} className={view === meta.key ? s.active : ''} onClick={() => { setView(meta.key); setEditing(false); }}>{meta.title}</button>)}
        <button className={view === 'generated' ? s.active : ''} onClick={() => { setView('generated'); setEditing(false); }}>
          助教成果{generated.data?.length ? ` ${generated.data.length}` : ''}
        </button>
      </div>
      {view === 'generated' ? (
        <GeneratedMaterials slug={slug} />
      ) : <div className={`${s.layout} ${key === 'body' && !structuredBody ? s.withNotes : ''}`}>
        <article className={s.document}>
          <header>
            <div><span>教学资产</span><h2>{asset.data?.meta.title ?? '加载中…'}</h2></div>
            <div className={s.actions}>
              {editing ? <><button onClick={() => { setEditing(false); setDraft(asset.data?.content ?? ''); }}>取消</button><button className={s.primary} onClick={commit} disabled={save.isPending}>保存</button></> : !contractBacked && <button onClick={() => setEditing(true)} disabled={!asset.data}>编辑</button>}
            </div>
          </header>
          {save.isError && <div className={s.error}>保存失败：资产可能已经被更新，请刷新后再编辑。</div>}
          {editing ? <textarea className={s.editor} value={draft} onChange={(event) => setDraft(event.target.value)} spellCheck={false} /> : (
            structuredBody
              ? <div className={s.contractContent}><ExplainReader projectSlug={slug} embedded /></div>
              : structuredPractice
                ? <div className={s.contractContent}><PracticeFlow projectSlug={slug} /></div>
                : asset.data?.content ? (key === 'body'
                  ? <MarkdownBodyReader slug={slug} content={asset.data.content} assetVersionId={asset.data.meta.currentVersionId} />
                  : key === 'practice'
                    ? <PracticeAsset content={asset.data.content} />
                    : <MarkdownView source={asset.data.content} className={s.markdown} />
                ) : <div className={s.empty}>这份资产还是空的。你可以直接编辑，也可以在教师对话中请助教根据学习上下文沉淀。</div>
          )}
        </article>
      </div>}
    </section>
  );
}

const PRACTICE_LABELS: Record<string, string> = {
  title: '标题', question: '题目', prompt: '题目', content: '内容', type: '题型',
  options: '选项', answer: '参考答案', correctAnswer: '参考答案', explanation: '解析',
  hint: '提示', difficulty: '难度', points: '分值', tasks: '练习', questions: '练习',
  exercises: '练习', items: '练习', id: '编号',
};

function PracticeAsset({ content }: { content: string }) {
  const match = content.match(/```json\s*([\s\S]*?)```/i);
  if (!match) return <MarkdownView source={content} className={s.markdown} />;
  try {
    const parsed = JSON.parse(match[1]) as unknown;
    return <div className={s.practice}><PracticeValue value={parsed} root /></div>;
  } catch {
    return <MarkdownView source={content} className={s.markdown} />;
  }
}

function PracticeValue({ value, root = false }: { value: unknown; root?: boolean }) {
  if (Array.isArray(value)) {
    return <div className={root ? s.practiceList : s.optionList}>{value.map((item, index) => (
      typeof item === 'object' && item !== null
        ? <article className={s.exercise} key={index}><span className={s.exerciseNo}>练习 {String(index + 1).padStart(2, '0')}</span><PracticeValue value={item} /></article>
        : <div className={s.option} key={index}><b>{String.fromCharCode(65 + index)}</b><span>{String(item)}</span></div>
    ))}</div>;
  }
  if (value && typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>);
    const collection = entries.find(([key, item]) => ['tasks', 'questions', 'exercises', 'items'].includes(key) && Array.isArray(item));
    return <>
      {entries.filter(([key]) => key !== collection?.[0]).map(([key, item]) => (
        <div className={s.practiceField} key={key}>
          <strong>{PRACTICE_LABELS[key] ?? key}</strong>
          {typeof item === 'string' ? <MarkdownView source={item} className={s.fieldMarkdown} /> : <PracticeValue value={item} />}
        </div>
      ))}
      {collection && <PracticeValue value={collection[1]} root />}
    </>;
  }
  return <span>{value == null ? '—' : String(value)}</span>;
}
