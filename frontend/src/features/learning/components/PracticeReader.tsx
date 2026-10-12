import { useMemo, useState } from 'react';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import s from './AssetsView.module.css';

export interface PracticeQuestion { question: string; answer: string; kind?: string; options?: Array<{id:string;text:string}> }
export const withoutPracticeTitle = (source:string) => source.replace(/^\s*#\s+(?:练习|练习题)[ \t]*(?:\r?\n|$)/, '');

type LegacyAnswers=Record<string,{answer:unknown;explanation?:string}>;
function parseLegacyPractice(source:string,legacyAnswers?:LegacyAnswers): {intro:string;questions:PracticeQuestion[]} | null {
 const candidates=[{raw:source.trim(),whole:source},...Array.from(source.matchAll(/```(?:json)?\s*\n([\s\S]*?)\n```/g),match=>({raw:match[1],whole:match[0]}))];
 for(const candidate of candidates){
  try{
   const data=JSON.parse(candidate.raw) as {tasks?:Array<Record<string,unknown>>;answers?:Record<string,unknown>};
   if(!Array.isArray(data.tasks)||!data.tasks.length)continue;
   const questions=data.tasks.map((task,index):PracticeQuestion=>{
    const kind=typeof task.type==='string'?task.type:undefined;
    const options=Array.isArray(task.options)?task.options.map((option,i)=>{
     if(typeof option==='string')return {id:String.fromCharCode(65+i),text:option};
     const value=option as Record<string,unknown>;
     return {id:String(value.id??value.label??String.fromCharCode(65+i)),text:String(value.text??value.content??value.value??'')};
    }):task.options&&typeof task.options==='object'?Object.entries(task.options).map(([id,text])=>({id,text:String(text)})):undefined;
    const legacy=legacyAnswers?.[String(task.id)];
    const storedAnswer=task.answer??task.correctAnswer??data.answers?.[String(task.id)]??legacy?.answer;
    const answer=storedAnswer&&typeof storedAnswer==='object'?JSON.stringify(storedAnswer,null,2):String(storedAnswer??'这道旧题没有附带参考答案。');
    const difficulty=typeof task.difficulty==='number'?` · 难度：${task.difficulty}/5`:'';
    return {question:`## 第${index+1}题\n${kind?`题型：${kind}${difficulty}\n\n`:''}${String(task.question??task.prompt??task.content??'')}\n${task.hint?`\n提示：${String(task.hint)}\n`:''}`,kind,options,answer:`### 参考答案\n${answer}\n\n### 解析\n${String(task.explanation??legacy?.explanation??'原题未附解析。')}`};
   });
   return {intro:withoutPracticeTitle(source.replace(candidate.whole,'')).trim(),questions};
  }catch{/* Historical free-form Markdown remains readable below. */}
 }
 return null;
}
export function parsePractice(source: string,legacyAnswers?:LegacyAnswers): {intro:string;questions:PracticeQuestion[]} | null {
 const legacy=parseLegacyPractice(source,legacyAnswers);if(legacy)return legacy;
 const lines=source.split(/\r?\n/), boundaries:number[]=[], answers=new Map<number,number>(), explanations=new Map<number,number>();
 let fence: {char:string;length:number}|null=null, math=false;
 const structural=new Set<number>();
 for(let i=0;i<lines.length;i++){
  const marker=lines[i].match(/^ {0,3}(`{3,}|~{3,})/);
  if(marker){if(!fence)fence={char:marker[1][0],length:marker[1].length};else if(marker[1][0]===fence.char&&marker[1].length>=fence.length)fence=null;continue;}
  if(fence)continue;
  if(lines[i].trim().startsWith('$$')){if((lines[i].match(/\$\$/g)??[]).length===1)math=!math;continue;}
  if(math)continue;
  structural.add(i);
  const heading=lines[i].match(/^##\s+第\s*(\d+)\s*题\s*$/);
  if(heading){if(Number(heading[1])!==boundaries.length+1)return null;boundaries.push(i);}
  if(/^###\s+参考答案\s*$/.test(lines[i])&&boundaries.length)answers.set(boundaries.length-1,i);
  if(/^###\s+解析\s*$/.test(lines[i])&&boundaries.length)explanations.set(boundaries.length-1,i);
 }
 if(!boundaries.length)return null;
 const questions:PracticeQuestion[]=[];
 for(let n=0;n<boundaries.length;n++){
  const start=boundaries[n],end=boundaries[n+1]??lines.length,answer=answers.get(n),explanation=explanations.get(n);
  if(answer===undefined||explanation===undefined||explanation<=answer||explanation>=end)return null;
  if(!lines.slice(answer+1,explanation).join('\n').trim()||!lines.slice(explanation+1,end).join('\n').trim())return null;
  const answerText=lines.slice(answer,end).join('\n');
  const questionLines=lines.slice(start,answer),kind=questionLines.find((_,i)=>structural.has(start+i)&&/题型：/.test(questionLines[i]))?.match(/题型：\s*([a-z-]+)/)?.[1];
  const options:Array<{id:string;text:string}>=[];
  const body=questionLines.filter((line,i)=>{
   const match=structural.has(start+i)&&line.match(/^([A-Z])[.、．)]\s*(.+)$/);
   if(match&&(kind==='single-choice'||kind==='multiple-choice')){options.push({id:match[1],text:match[2]});return false;}
   return true;
  });
  questions.push({question:body.join('\n'),answer:answerText,kind,options});
 }
 return {intro:withoutPracticeTitle(lines.slice(0,boundaries[0]).join('\n')),questions};
}
function Exercise({question,index}:{question:PracticeQuestion;index:number}){
 const [selected,setSelected]=useState<string[]>([]),[draft,setDraft]=useState('');
 const options=question.options?.length?question.options:question.kind==='true-false'?[{id:'true',text:'正确'},{id:'false',text:'错误'}]:[];
 return <section className={s.exercise}><MarkdownView source={question.question}/>
  {options.length?<fieldset className={s.optionList}><legend>第{index+1}题作答</legend>{options.map(option=><label key={option.id} className={s.option}><input type={question.kind==='multiple-choice'?'checkbox':'radio'} name={`practice-${index}`} checked={selected.includes(option.id)} onChange={event=>setSelected(question.kind==='multiple-choice'?(event.target.checked?[...selected,option.id]:selected.filter(id=>id!==option.id)):[option.id])}/><span><MarkdownView source={`${option.id==='true'||option.id==='false'?'':`${option.id}. `}${option.text}`}/></span></label>)}</fieldset>:<label className={s.practiceField}>我的作答<textarea aria-label={`第${index+1}题作答`} value={draft} onChange={event=>setDraft(event.target.value)}/></label>}
  <details className={s.answer}><summary>参考答案与解析</summary><MarkdownView source={question.answer}/></details>
 </section>;
}
export function PracticeReader({content,versionId,legacyAnswers}:{content:string;versionId:string;legacyAnswers?:LegacyAnswers}) {
 const parsed=useMemo(()=>parsePractice(content,legacyAnswers),[content,legacyAnswers]);
 if(!parsed)return <MarkdownView source={withoutPracticeTitle(content)} className={s.markdown}/>;
 return <div className={s.practice} key={versionId}>{parsed.intro.trim()&&<MarkdownView source={parsed.intro}/>}<div className={s.practiceList}>{parsed.questions.map((question,index)=><Exercise question={question} index={index} key={`${versionId}-${index}`}/>)}</div></div>;
}
