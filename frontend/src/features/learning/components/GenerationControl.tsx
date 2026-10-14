import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Modal } from '@/shared/primitive/Modal';
import { Button } from '@/shared/primitive/Button';
import { describeGenerationError } from '../api/generationError';
import { useAskAiSettings } from '@/features/settings';
import { useSources } from '../api/learningWorkspace';
import { isGenerationActive, useGenerationAction, useGenerationJobs, type GenerationKind } from '../api/generation';
import { SourceImportControl } from './SourceImportControl';
import s from './GenerationControl.module.css';

const names = {review: '复习稿', practice: '练习题', 'discipline-map': '地图'};
const phases = {waiting:'排队中',preparing:'准备中',generating:'生成中',validating:'校验中',publishing:'保存中',done:'完成'};
export function GenerationControl({slug, kind, disabled = false}: {slug:string;kind:GenerationKind;disabled?:boolean}) {
 const [open,setOpen]=useState(false), [provider,setProvider]=useState(''), [model,setModel]=useState(''), [sourceIds,setSourceIds]=useState<string[]>([]), [count,setCount]=useState(10), [mode,setMode]=useState<'general'|'sources'>('general');
 const settings=useAskAiSettings(), sources=useSources(slug), jobs=useGenerationJobs(slug), action=useGenerationAction(slug);
 const providers=settings.data?.providers??[];
 const operation=useRef<{intent:string;id:string}>();
 const candidates=[...(jobs.data?.jobs??[])].filter(item=>item.kind===kind).sort((a,b)=>b.updatedAt.localeCompare(a.updatedAt));
 const job=candidates.find(isGenerationActive)??candidates[0];
 const active=Boolean(job&&isGenerationActive(job));
 useEffect(()=>{if(open){operation.current=undefined;setProvider('');setModel('');setSourceIds((sources.data??[]).filter(item=>item.status==='ready').slice(0,32).map(item=>item.sourceId));action.reset();}},[open]);
 useEffect(()=>{ if(sources.data) setSourceIds(ids=>ids.filter(id=>sources.data!.some(item=>item.sourceId===id&&item.status==='ready'))); },[sources.data]);
 const submit=()=>{
  const request={kind,...(provider?{providerId:provider}:{}),...(model.trim()?{model:model.trim()}:{}),sourceIds:kind==='discipline-map'&&mode==='general'?[]:sourceIds,...(kind==='practice'?{questionCount:count}:{}),...(kind==='discipline-map'?{mapMode:mode}:{})};
  const intent=JSON.stringify(request);
  if(operation.current?.intent!==intent)operation.current={intent,id:crypto.randomUUID()};
  action.mutate({request:{...request,operationId:operation.current.id}},{onSuccess:()=>setOpen(false)});
 };
 const measured=job&&!job.progress?.indeterminate&&Number(job.progress?.totalUnits)>0;
 return <div className={s.control}>
  <button type="button" disabled={active||disabled} onClick={()=>setOpen(true)}>生成{names[kind]}</button>
  {job&&<div className={s.status} role="status"><span>{active?phases[job.phase]:({succeeded:'已完成',failed:'生成失败',cancelled:'已取消',interrupted:'已中断'} as const)[job.status as 'succeeded']}</span>
   {Number(job.progress?.outputTokens)>0&&<small>{job.progress.tokensEstimated?'约 ':''}{job.progress.outputTokens?.toLocaleString()} token{job.progress.tokensEstimated?'（估算）':''}</small>}
   {active&&<>{measured?<progress max={job.progress.totalUnits} value={job.progress.completedUnits}/>:<progress/>}{measured&&<small>{job.progress.completedUnits}/{job.progress.totalUnits} {job.progress.unitLabel}</small>}<button type="button" disabled={action.isPending} onClick={()=>action.mutate({id:job.id,action:'cancel'})}>取消生成</button></>}
   {(job.status==='failed'||job.status==='interrupted')&&<button type="button" disabled={action.isPending} onClick={()=>action.mutate({id:job.id,action:'retry',operationId:crypto.randomUUID()})}>重试</button>}
   {job.errorCode&&<small>{describeGenerationError(job.errorCode)}</small>}
   {job.status==='succeeded'&&job.checkpoint?.hasMoreConversation&&<small>本批已保存，还有对话可继续{kind==='practice'?'生成下一轮':'总结'}。</small>}
  </div>}
  {!open&&action.isError&&<span role="alert">{describeGenerationError(action.error)}</span>}
  <Modal open={open} onOpenChange={setOpen} title={`生成${names[kind]}`} description={kind==='review'?'从上次保存的位置继续总结，新内容融入原稿。':kind==='practice'?'生成独立的新一轮练习，旧轮次保留，可分页阅读。':'选择本次使用的连接、模型与资料。生成期间旧地图仍可阅读。'} footer={<Button variant="primary" onClick={submit} loading={action.isPending} disabled={active||!providers.length||(kind==='discipline-map'&&mode==='sources'&&!sourceIds.length)}>{action.isPending?'提交中…':'开始生成'}</Button>}>
   <div className={s.fields}>
    {!providers.length?<Link to="/settings">先添加 API 连接</Link>:<><label>API 连接<select value={provider} onChange={event=>{setProvider(event.target.value);setModel(providers.find(item=>item.id===event.target.value)?.model??'');}}><option value="">使用用途默认</option>{providers.map(item=><option key={item.id} value={item.id}>{item.name||item.id} · {item.model}</option>)}</select></label><label>模型<input value={model} onChange={event=>setModel(event.target.value)} placeholder="使用配置中的模型"/></label></>}
    {kind==='practice'&&<label>题数<input type="number" min={1} max={50} value={count} onChange={event=>setCount(Math.max(1,Math.min(50,Number(event.target.value)||1)))}/></label>}
    {kind==='discipline-map'&&<><label>组织方式<select value={mode} onChange={event=>setMode(event.target.value as typeof mode)}><option value="general">由模型组织</option><option value="sources">按参考资料组织</option></select></label>{mode==='sources'&&<><p>根据已提取正文整理章节，无需现成目录。尚未识别的页面不会作为已读资料。</p><SourceImportControl slug={slug}/></>}</>}
    {(kind!=='discipline-map'||mode==='sources')&&<fieldset><legend>本次资料（最多32份）</legend>{(sources.data??[]).filter(item=>item.status!=='deleted'&&item.status!=='tombstoned').map(item=><label key={item.sourceId} className={s.source}><input type="checkbox" disabled={item.status!=='ready'||(!sourceIds.includes(item.sourceId)&&sourceIds.length>=32)} checked={sourceIds.includes(item.sourceId)} onChange={event=>setSourceIds(ids=>event.target.checked?[...ids,item.sourceId]:ids.filter(id=>id!==item.sourceId))}/>{item.displayName}{item.coverage?.totalPages&&<small>已提取 {item.coverage.extractedPages}/{item.coverage.totalPages} 页{item.coverage.unreadPages?` · ${item.coverage.unreadPages} 页待识别`:''}</small>}{item.status!=='ready'&&<small>准备好后可选</small>}</label>)}{!sources.data?.length&&<p>尚无资料，可先生成或上传资料。</p>}</fieldset>}
    {action.isError&&<p role="alert">{describeGenerationError(action.error)}</p>}
   </div>
  </Modal>
 </div>;
}
