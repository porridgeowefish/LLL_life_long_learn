import { useEffect, useMemo, useState } from 'react';

import { useActivity, type ActivityDay, type LearningEvent } from '@/features/projects/api/activity';
import type { ProjectMeta } from '@/shared/types/domain';

import s from './LearningRhythm.module.css';

interface Props { projects: ProjectMeta[] }

export function LearningRhythm({ projects }: Props) {
  const [weeks, setWeeks] = useState<26 | 52>(26);
  const [project, setProject] = useState('');
  const activity = useActivity(weeks, project);
  const summary = activity.data;
  const [selectedDate, setSelectedDate] = useState('');

  useEffect(() => {
    if (!summary?.days.length) return;
    const latestActive = [...summary.days].reverse().find((day) => day.activity > 0);
    setSelectedDate((current) => summary.days.some((day) => day.date === current) ? current : (latestActive?.date ?? summary.rangeEnd));
  }, [summary]);

  const selected = summary?.days.find((day) => day.date === selectedDate);
  const weekLabels = useMemo(() => buildWeekLabels(summary?.days ?? []), [summary?.days]);
  const currentWeekLength = summary?.days.length ? ((summary.days.length - 1) % 7) + 1 : 0;
  const thisWeek = currentWeekLength ? summary!.days.slice(-currentWeekLength) : [];

  return (
    <section className={s.section} aria-labelledby="learning-rhythm-title">
      <header className={s.head}>
        <div>
          <h2 id="learning-rhythm-title">学习节律</h2>
          <p>颜色表示每日投入，连续天数记录坚持；成长值只来自可验证的学习成果。</p>
        </div>
        <div className={s.filters}>
          <select value={project} onChange={(event) => setProject(event.target.value)} aria-label="筛选学习项目">
            <option value="">全部项目</option>
            {projects.map((item) => <option key={item.slug} value={item.slug}>{item.title}</option>)}
          </select>
          <button className={weeks === 26 ? s.activeRange : ''} onClick={() => setWeeks(26)}>近26周</button>
          <button className={weeks === 52 ? s.activeRange : ''} onClick={() => setWeeks(52)}>近一年</button>
        </div>
      </header>

      <div className={s.rhythmBody} aria-busy={activity.isLoading}>
        <aside className={s.streak}>
          <div className={s.streakValue}><strong>{summary?.currentStreak ?? 0} 天</strong></div>
          <span>当前连续学习</span>
          <p><b>保持自己的节律</b>稳定回来学习，比追求单日高强度更重要。</p>
        </aside>
        <div className={s.calendarScroll}>
          <div className={s.months} style={{ gridTemplateColumns: `repeat(${weekLabels.length}, 15px)` }}>
            {weekLabels.map((label, index) => <span key={`${label}-${index}`}>{label}</span>)}
          </div>
          <div className={s.calendarLine}>
            <div className={s.weekdays} aria-hidden="true"><span>一</span><span>三</span><span>五</span></div>
            <div className={s.heatmap} role="grid" aria-label={`最近${weeks}周学习投入`}>
              {(summary?.days ?? []).map((day) => (
                <button
                  key={day.date}
                  type="button"
                  role="gridcell"
                  className={`${s.day} ${selectedDate === day.date ? s.selected : ''}`}
                  data-level={activityLevel(day.activity)}
                  title={`${formatDate(day.date)} · 热度 ${day.activity} · ${day.actions} 次学习行动`}
                  aria-label={`${formatDate(day.date)}，投入热度 ${day.activity}，${day.actions} 次学习行动`}
                  onClick={() => setSelectedDate(day.date)}
                />
              ))}
            </div>
          </div>
          <div className={s.legend}><span>点击一天查看具体记录</span><span>少 <i data-level="0" /><i data-level="1" /><i data-level="2" /><i data-level="3" /><i data-level="4" /> 多</span></div>
        </div>
      </div>

      <div className={s.metrics}>
        <Metric label={`近${weeks}周活跃`} value={summary?.activeDays ?? 0} suffix="天" />
        <Metric label="最长连续" value={summary?.longestStreak ?? 0} suffix="天" />
        <Metric label="有效学习行动" value={summary?.totalActions ?? 0} suffix="次" />
        <Metric label="累计成长值" value={summary?.totalGrowth ?? 0} suffix="分" />
      </div>

      <div className={s.detailGrid}>
        <DayDetail day={selected} />
        <WeeklyInvestment days={thisWeek} />
      </div>
    </section>
  );
}

function Metric({ label, value, suffix }: { label: string; value: number; suffix: string }) {
  return <div className={s.metric}><span>{label}</span><strong>{value}</strong><small>{suffix}</small></div>;
}

function DayDetail({ day }: { day?: ActivityDay }) {
  return (
    <article className={s.detail}>
      <header><h3>{day ? formatDate(day.date) : '学习记录'}</h3><span>{day?.activity ? `投入热度 ${day.activity}` : '休息日'}</span></header>
      {!day?.events.length ? (
        <p className={s.emptyDay}>这一天没有学习记录。适当休息也是长期学习的一部分。</p>
      ) : (
        <ul>{day.events.map((event) => <EventRow key={`${event.projectSlug}-${event.id}`} event={event} />)}</ul>
      )}
    </article>
  );
}

function EventRow({ event }: { event: LearningEvent }) {
  return (
    <li>
      <span className={s.eventMark}>{eventMark(event.sourceType)}</span>
      <span className={s.eventCopy}><strong>{event.title || fallbackTitle(event.sourceType)}</strong><small>{event.projectTitle}{event.detail ? ` · ${event.detail}` : ''}</small></span>
      <span className={s.eventGain}>{event.delta > 0 ? `成长 +${event.delta}` : `热度 +${event.activityDelta ?? 1}`}</span>
    </li>
  );
}

function WeeklyInvestment({ days }: { days: ActivityDay[] }) {
  const labels = ['一', '二', '三', '四', '五', '六', '日'];
  const max = Math.max(1, ...days.map((day) => day.activity));
  const active = days.filter((day) => day.activity > 0).length;
  return (
    <aside className={s.weekPanel}>
      <header><h3>本周投入</h3><span>{active} / 7 天</span></header>
      <div className={s.bars}>
        {labels.map((label, index) => <i key={label} style={{ height: `${Math.max(2, ((days[index]?.activity ?? 0) / max) * 64)}px` }}><span>{label}</span></i>)}
      </div>
      <p>{active > 0 ? '已经留下本周的学习节律，继续按当前频率推进即可。' : '本周还没有学习记录，从一次短小而明确的学习行动开始。'}</p>
    </aside>
  );
}

function buildWeekLabels(days: ActivityDay[]) {
  const labels: string[] = [];
  let previous = -1;
  for (let i = 0; i < days.length; i += 7) {
    const month = Number(days[i]?.date.slice(5, 7) ?? 0);
    labels.push(month && month !== previous ? `${month}月` : '');
    previous = month || previous;
  }
  return labels;
}

function activityLevel(value: number) { return value === 0 ? 0 : value <= 2 ? 1 : value <= 5 ? 2 : value <= 9 ? 3 : 4; }
function formatDate(value: string) { const [, month, day] = value.split('-'); return `${Number(month)}月${Number(day)}日`; }
function eventMark(type: string) { return type.startsWith('practice') ? '练' : type.startsWith('flashcard') ? '复' : type.startsWith('ask') ? '问' : type === 'reading' ? '阅' : '学'; }
function fallbackTitle(type: string) {
  if (type === 'reading') return '有效阅读';
  if (type === 'practice-correct') return '答题正确';
  if (type === 'practice-evaluation') return '练习评估';
  if (type.startsWith('practice')) return '完成练习';
  if (type.startsWith('flashcard')) return '复习闪卡';
  if (type === 'extend-flower') return '编辑知识花朵';
  return '学习行动';
}
