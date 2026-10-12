import { useEffect, useState } from 'react';
import { useAsset, useAssets, usePracticeRounds, useSaveAsset } from '../api/learningWorkspace';
import { PracticeReader } from './PracticeReader';
import { GenerationControl } from './GenerationControl';
import { NotesView } from './NotesView';
import { ReviewReader } from './ReviewReader';
import s from './AssetsView.module.css';

type AssetKey = 'review' | 'practice';
export function AssetsView({slug}:{slug:string}) {
 const [view,setViewState]=useState<AssetKey | 'notes'>(()=>{const saved=localStorage.getItem(`lll.assets.view.${slug}`);return saved==='practice'||saved==='notes'?saved:'review';});
 const setView=(next:AssetKey|'notes')=>{setViewState(next);localStorage.setItem(`lll.assets.view.${slug}`,next);};
 const key:AssetKey=view==='notes'?'review':view;
 const [editing,setEditing]=useState(false);
 const [draft,setDraft]=useState('');
 const assets=useAssets(slug);
 const asset=useAsset(slug,key);
 const rounds=usePracticeRounds(slug,key==='practice');
 const [roundId,setRoundId]=useState<string|null>(null);
 const roundItems=rounds.data??[];
 const selectedRound=roundItems.find(item=>item.versionId===roundId)??roundItems[roundItems.length-1];
 const roundIndex=selectedRound?roundItems.indexOf(selectedRound):-1;
 const historical=key==='practice'&&selectedRound&&selectedRound.versionId!==asset.data?.meta.currentVersionId;
 const save=useSaveAsset(slug,key);
 useEffect(()=>{if(!editing)setDraft(asset.data?.content??'');},[asset.data?.content,editing]);
 useEffect(()=>{const saved=localStorage.getItem(`lll.assets.view.${slug}`);setViewState(saved==='practice'||saved==='notes'?saved:'review');setEditing(false);setDraft('');setRoundId(null);},[slug]);
 useEffect(()=>setRoundId(null),[asset.data?.meta.currentVersionId]);
 const commit=()=>{if(!asset.data)return;save.mutate({baseEditRevision:asset.data.meta.editRevision,content:draft},{onSuccess:()=>setEditing(false)});};
 return <section className={s.assets}>
  <div className={s.assetTabs}>
   {(assets.data??[]).map(meta=><button key={meta.key} className={view===meta.key?s.active:''} onClick={()=>{setView(meta.key);setEditing(false);}}>{meta.title}</button>)}
   <button className={view==='notes'?s.active:''} onClick={()=>{setView('notes');setEditing(false);}}>笔记</button>
  </div>
  {view==='notes'?<NotesView slug={slug}/>:<div className={`${s.layout} ${key==='review'?s.withNotes:''}`}>
   <article className={s.document}>
    <header><div><span>教学资产</span><h2>{asset.data?.meta.title??'加载中…'}</h2></div><div className={s.actions}>
     <GenerationControl slug={slug} kind={key} disabled={editing}/>
     {editing?<><button onClick={()=>setEditing(false)}>取消</button><button className={s.primary} onClick={commit} disabled={save.isPending}>保存</button></>:<button onClick={()=>setEditing(true)} disabled={!asset.data||Boolean(historical)}>编辑</button>}
    </div></header>
    {asset.isError&&<div className={s.error} role="alert">资产读取失败，请重试。</div>}
    {save.isError&&<div className={s.error} role="alert">保存失败：资产可能已经被更新，请刷新后再编辑。</div>}
    {key==='practice'&&roundItems.length>0&&!editing&&<nav className={s.roundPager} aria-label="练习轮次"><button onClick={()=>setRoundId(roundItems[roundIndex-1].versionId)} disabled={roundIndex<=0}>上一轮</button><span>第 {roundIndex+1} / {roundItems.length} 轮</span><button onClick={()=>setRoundId(roundItems[roundIndex+1].versionId)} disabled={roundIndex>=roundItems.length-1}>下一轮</button></nav>}
    {key==='practice'&&rounds.isError&&<div className={s.error} role="alert">旧轮次读取失败，请刷新重试。当前轮仍可阅读。</div>}
    {editing?<textarea className={s.editor} value={draft} onChange={event=>setDraft(event.target.value)} spellCheck={false}/>:asset.data&&(asset.data.content||(key==='practice'&&selectedRound?.content))?(key==='review'?<ReviewReader slug={slug} content={asset.data.content} assetVersionId={asset.data.meta.currentVersionId}/>:<PracticeReader content={selectedRound?.content??asset.data.content} versionId={selectedRound?.versionId??asset.data.meta.currentVersionId} legacyAnswers={selectedRound?.legacyAnswers}/>):<div className={s.empty}>这份资产还是空的。可以直接编辑，或使用上方按钮生成。</div>}
   </article>
  </div>}
 </section>;
}
