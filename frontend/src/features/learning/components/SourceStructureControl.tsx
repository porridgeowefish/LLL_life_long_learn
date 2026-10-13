import { useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { http } from '@/shared/client';
import { subscribeToSSE } from '@/app/hooks/useSSE';
import { SSE_EVENTS } from '@/shared/lib/constants';
import type { LearningSource } from '../api/learningWorkspace';
interface StructureJob {id:string;status:'queued'|'running'|'succeeded'|'failed'|'cancelled'|'interrupted'}
export function SourceStructureControl({slug,source}:{slug:string;source:LearningSource}) {
 const qc=useQueryClient(),storageKey=`lll.source.structure.${slug}.${source.sourceId}`;
 const [jobId,setJobId]=useState(()=>localStorage.getItem(storageKey)??'');
 const operation=useRef('');
 useEffect(()=>{operation.current='';},[source.currentRevisionId,source.derivedVersionId]);
 const endpoint=`/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(source.sourceId)}/structure-jobs`;
 const query=useQuery({queryKey:['source-structure',slug,source.sourceId,jobId],enabled:Boolean(jobId),queryFn:()=>http.get<{job:StructureJob}>(`${endpoint}/${encodeURIComponent(jobId)}`),refetchInterval:q=>q.state.data?.job.status==='queued'||q.state.data?.job.status==='running'?1500:false});
 const active=query.data?.job.status==='queued'||query.data?.job.status==='running';
 useEffect(()=>subscribeToSSE(SSE_EVENTS.sourceStructureJobUpdated,payload=>{if((payload as {projectSlug?:string;sourceId?:string}).projectSlug===slug&&(payload as {sourceId?:string}).sourceId===source.sourceId){void query.refetch();void qc.invalidateQueries({queryKey:['learning-workspace',slug,'sources']});}}),[qc,slug,source.sourceId,jobId]);
 useEffect(()=>{if(query.data?.job.status==='succeeded')void qc.invalidateQueries({queryKey:['learning-workspace',slug,'sources']});},[query.data?.job.status,qc,slug]);
 const action=useMutation({mutationFn:(cancel:boolean)=>cancel?http.post<{job:StructureJob}>(`${endpoint}/${encodeURIComponent(jobId)}/cancel`,{}):http.post<{job:StructureJob}>(endpoint,{operationId:operation.current||(operation.current=crypto.randomUUID()),revisionId:source.currentRevisionId,baseDerivedVersionId:source.derivedVersionId}),onSuccess:result=>{operation.current='';setJobId(result.job.id);localStorage.setItem(storageKey,result.job.id);void qc.invalidateQueries({queryKey:['source-structure',slug,source.sourceId]});}});
 if(!source.derivedVersionId||!(/\.pdf$/i.test(source.displayName)))return null;
 return <div>{active?<><span role="status">正在识别扫描页</span><button type="button" disabled={action.isPending} onClick={()=>action.mutate(true)}>取消识别</button></>:<button type="button" disabled={action.isPending} onClick={()=>action.mutate(false)}>{action.isPending?'正在准备…':query.data?.job.status==='failed'||query.data?.job.status==='interrupted'?'重试识别':source.coverage?.unreadPages||source.status==='failed'?'识别扫描页':'继续识别'}</button>}{query.data?.job.status==='succeeded'&&<small role="status">本批已完成；其余页可继续分批识别</small>}{action.isError&&<span role="alert">识别未开始：Windows 需安装简体中文 OCR 语言包；也可在设置中选择文档识别模型。已有识别任务时请等待其完成。</span>}</div>;
}
