import { Link } from 'react-router-dom';
import s from './LearningProgress.module.css';

export interface LearningProgressValue { goal: string; taught: number; total: number; achieved: boolean }
export function readLearningProgress(source: string): LearningProgressValue | undefined {
  const matches = [...source.matchAll(/(?:^|\n)```learning-progress[^\S\r\n]*\r?\n([\s\S]{1,2000}?)\r?\n```(?=\r?\n|$)/g)];
  if (!matches.length) return;
  try {
    const p = JSON.parse(matches[matches.length - 1][1]);
    if (typeof p.goal !== 'string' || !p.goal.trim() || p.goal.length > 200 || !Number.isSafeInteger(p.taught) || !Number.isSafeInteger(p.total) || p.total < 1 || p.taught < 0 || p.taught > p.total || typeof p.achieved !== 'boolean') return;
    return { goal: p.goal.trim(), taught: p.taught, total: p.total, achieved: p.achieved && p.taught === p.total };
  } catch { return; }
}
export function withoutLearningProgress(source: string) {
  return source.replace(/(?:^|\n)```learning-progress[^\n]*\n[\s\S]*?(?:\n```|$)/g, '').trimEnd();
}
export function LearningProgress({ value, slug }: { value?: LearningProgressValue; slug: string }) {
  const percent = value ? Math.round(value.taught / value.total * 100) : undefined;
  return <div className={s.progress}>
    <div className={s.label}><span>{value?.goal ?? '学习进度'}</span><span>{percent === undefined ? '—' : `${percent}%`}</span></div>
    <progress aria-label="学习进度" aria-valuetext={value ? `已讲授 ${value.taught} / ${value.total}，${value.achieved ? '目标已达成' : '尚待验证目标'}` : '尚未评估'} max={100} value={percent} />
    {value?.achieved && <div className={s.complete}><span>恭喜，已达成本轮学习目标！</span><Link to={`/project/${encodeURIComponent(slug)}/assets`} onClick={() => localStorage.setItem(`lll.assets.view.${slug}`, 'review')}>去复习</Link><Link to={`/project/${encodeURIComponent(slug)}/assets`} onClick={() => localStorage.setItem(`lll.assets.view.${slug}`, 'practice')}>去练习</Link></div>}
  </div>;
}
