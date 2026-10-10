import { useEffect, useState } from 'react';
import { useAsset, useAssets, useSaveAsset } from '../api/learningWorkspace';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import { NotesView } from './NotesView';
import { ReviewReader } from './ReviewReader';
import s from './AssetsView.module.css';

type AssetKey = 'review' | 'practice';
export function AssetsView({slug}:{slug:string}) {
 const [view,setView]=useState<AssetKey | 'notes'>('review');
 const key:AssetKey=view==='notes'?'review':view;
 const [editing,setEditing]=useState(false);
 const [draft,setDraft]=useState('');
 const assets=useAssets(slug);
 const asset=useAsset(slug,key);
 const save=useSaveAsset(slug,key);
 useEffect(()=>{if(!editing)setDraft(asset.data?.content??'');},[asset.data?.content,editing]);
 useEffect(()=>{setView('review');setEditing(false);setDraft('');},[slug]);
 const commit=()=>{if(!asset.data)return;save.mutate({baseEditRevision:asset.data.meta.editRevision,content:draft},{onSuccess:()=>setEditing(false)});};
 return <section className={s.assets}>
  <div className={s.assetTabs}>
   {(assets.data??[]).map(meta=><button key={meta.key} className={view===meta.key?s.active:''} onClick={()=>{setView(meta.key);setEditing(false);}}>{meta.title}</button>)}
   <button className={view==='notes'?s.active:''} onClick={()=>{setView('notes');setEditing(false);}}>笔记</button>
  </div>
  {view==='notes'?<NotesView slug={slug}/>:<div className={`${s.layout} ${key==='review'?s.withNotes:''}`}>
   <article className={s.document}>
    <header><div><span>教学资产</span><h2>{asset.data?.meta.title??'加载中…'}</h2></div><div className={s.actions}>
     {editing?<><button onClick={()=>setEditing(false)}>取消</button><button className={s.primary} onClick={commit} disabled={save.isPending}>保存</button></>:<button onClick={()=>setEditing(true)} disabled={!asset.data}>编辑</button>}
    </div></header>
    {asset.isError&&<div className={s.error} role="alert">资产读取失败，请重试。</div>}
    {save.isError&&<div className={s.error} role="alert">保存失败：资产可能已经被更新，请刷新后再编辑。</div>}
    {editing?<textarea className={s.editor} value={draft} onChange={event=>setDraft(event.target.value)} spellCheck={false}/>:asset.data?.content?(key==='review'?<ReviewReader slug={slug} content={asset.data.content} assetVersionId={asset.data.meta.currentVersionId}/>:<MarkdownView source={asset.data.content} className={s.markdown}/>):<div className={s.empty}>这份资产还是空的。可以直接编辑，或请助教整理复习稿与练习题。</div>}
   </article>
  </div>}
 </section>;
}
