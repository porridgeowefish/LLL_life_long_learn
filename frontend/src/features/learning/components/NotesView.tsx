import { Link } from 'react-router-dom';
import { useNotes, useRetryNote, useTeacherSettings } from '../api/notes';
import { MarkdownView } from '@/shared/primitive/MarkdownView';
import s from './NotesView.module.css';
const errors: Record<string, string> = { provider_not_configured: '请先配置教师 API 连接', timeout: '生成超时', interrupted: '生成被中断', generation_failed: '生成失败', persist_failed: '保存失败' };
export function NotesView({ slug }: { slug: string }) {
 const notes = useNotes(slug), settings = useTeacherSettings(), retry = useRetryNote(slug);
 return <div className={s.notes} role="region" aria-label="教师笔记" tabIndex={0}><div className={s.body}>
  <header><div><h2>笔记</h2><p>{settings.data?.autoNotes.enabled ? `每 ${settings.data.autoNotes.everyTurns} 个成功对话回合自动记录` : '自动笔记已关闭'}</p></div><Link to="/settings">配置自动笔记</Link></header>
  {notes.isLoading && <p>加载笔记中…</p>}
  {notes.isError && <div role="alert">笔记读取失败。<button onClick={() => void notes.refetch()}>重试</button></div>}
  {notes.data?.entries.length === 0 && <p className={s.empty}>笔记会记录对话中的关键概念、知识点与理解变化。</p>}
  {notes.data?.entries.map(entry => <article key={entry.id}><div className={s.meta}>第 {entry.fromTurn}–{entry.throughTurn} 轮 · {new Date(entry.createdAt).toLocaleString()}</div><MarkdownView source={entry.content} /></article>)}
  {notes.data?.jobs.filter(job => job.status !== 'succeeded').map(job => <div className={s.job} key={job.id}><span>第 {job.fromTurn}–{job.throughTurn} 轮：{job.status === 'failed' ? errors[job.errorCode ?? ''] ?? '生成失败' : job.status === 'running' ? '记录中…' : '等待记录'}</span>{job.status === 'failed' && job.retryable && <button disabled={retry.isPending} onClick={() => retry.mutate(job.id)}>重试笔记</button>}</div>)}
  {retry.isError && <p role="alert">重试失败，请刷新笔记状态。</p>}
 </div></div>;
}
