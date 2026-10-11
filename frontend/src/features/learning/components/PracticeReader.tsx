import { useMemo } from 'react';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import s from './AssetsView.module.css';

export interface PracticeQuestion { question: string; answer: string }
export function parsePractice(source: string): {intro:string;questions:PracticeQuestion[]} | null {
 const lines=source.split(/\r?\n/), boundaries:number[]=[], answers=new Map<number,number>(), explanations=new Map<number,number>();
 let fence: {char:string;length:number}|null=null;
 for(let i=0;i<lines.length;i++){
  const marker=lines[i].match(/^ {0,3}(`{3,}|~{3,})/);
  if(marker){if(!fence)fence={char:marker[1][0],length:marker[1].length};else if(marker[1][0]===fence.char&&marker[1].length>=fence.length)fence=null;continue;}
  if(fence)continue;
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
  questions.push({question:lines.slice(start,answer).join('\n'),answer:answerText});
 }
 return {intro:lines.slice(0,boundaries[0]).join('\n'),questions};
}
export function PracticeReader({content,versionId}:{content:string;versionId:string}) {
 const parsed=useMemo(()=>parsePractice(content),[content]);
 if(!parsed)return <MarkdownView source={content} className={s.markdown}/>;
 return <div className={s.practice} key={versionId}>{parsed.intro&&<MarkdownView source={parsed.intro}/>}<div className={s.practiceList}>{parsed.questions.map((question,index)=><section className={s.exercise} key={`${versionId}-${index}`}><MarkdownView source={question.question}/><details className={s.answer}><summary>参考答案与解析</summary><MarkdownView source={question.answer}/></details></section>)}</div></div>;
}
